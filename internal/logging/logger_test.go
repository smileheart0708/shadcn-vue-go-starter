package logging

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestStreamKeepsEntriesWithinByteBudget(t *testing.T) {
	t.Parallel()

	stream := NewStream(StreamOptions{MaxBytes: 260})
	base := time.Now().UTC()

	for index := range 5 {
		stream.Publish(StreamEntry{
			Timestamp: base.Add(time.Duration(index) * time.Second).Unix(),
			Level:     "INFO",
			Message:   "entry",
			Text:      strings.Repeat(fmt.Sprintf("entry-%d ", index), 4),
			Source:    "app",
		})
	}

	if stream.buffered > stream.maxBytes {
		t.Fatalf("expected buffered bytes <= max bytes, got %d > %d", stream.buffered, stream.maxBytes)
	}

	snapshot := stream.Snapshot(0)
	if len(snapshot) == 0 || len(snapshot) >= 5 {
		t.Fatalf("expected byte budget to prune oldest entries, got %d entries", len(snapshot))
	}
	for index := 1; index < len(snapshot); index++ {
		if snapshot[index].Timestamp <= snapshot[index-1].Timestamp {
			t.Fatalf("expected timestamps to increase, got %+v", snapshot)
		}
	}
	if snapshot[0].ID <= 1 {
		t.Fatalf("expected oldest entries to be pruned, got %+v", snapshot)
	}
	if snapshot[0].ID <= 0 || snapshot[len(snapshot)-1].ID <= snapshot[0].ID {
		t.Fatalf("expected ascending ids, got %+v", snapshot)
	}
}

// TestStreamSnapshotsLiveWindowAfterPruning covers the index arithmetic that only
// misbehaves once entries have been dropped: dropped slots keep occupying the
// backing slice until compaction, so Snapshot must start at headIndex instead of
// the beginning of the slice.
func TestStreamSnapshotsLiveWindowAfterPruning(t *testing.T) {
	t.Parallel()

	// Every entry below is exactly 73 bytes (64 base + "INFO" + "m" + "t" + "app"),
	// so a 20-entry budget makes each publish past the 20th drop exactly one entry.
	// Publishing 25 leaves headIndex at 5 with plenty of stale slots in front of it,
	// which is the state compactIfNeededLocked deliberately defers until headIndex
	// reaches half the slice.
	const (
		entryBudget  = 73 * 20
		totalEntries = 25
		liveEntries  = 20
		firstLiveID  = totalEntries - liveEntries + 1
	)

	stream := NewStream(StreamOptions{MaxBytes: entryBudget})
	for range totalEntries {
		stream.Publish(StreamEntry{
			Level:   "INFO",
			Message: "m",
			Text:    "t",
			Source:  "app",
		})
	}

	if dropped := len(stream.entries) - stream.activeLenLocked(); dropped != totalEntries-liveEntries {
		t.Fatalf("expected %d dropped slots retained in the backing slice, got %d", totalEntries-liveEntries, dropped)
	}
	if stream.headIndex == 0 {
		t.Fatal("expected pruning to advance headIndex, so Snapshot has to honour it")
	}

	all := stream.Snapshot(0)
	if len(all) != liveEntries || all[0].ID != firstLiveID || all[liveEntries-1].ID != totalEntries {
		t.Fatalf("expected entries %d..%d after pruning, got %d entries starting at %d", firstLiveID, totalEntries, len(all), all[0].ID)
	}

	tail := stream.Snapshot(3)
	if len(tail) != 3 || tail[0].ID != totalEntries-2 || tail[2].ID != totalEntries {
		t.Fatalf("expected the newest three entries after pruning, got %+v", tail)
	}
}

func TestStreamSnapshotTail(t *testing.T) {
	t.Parallel()

	stream := NewStream(StreamOptions{MaxBytes: 16 * 1024})
	for range 5 {
		stream.Publish(StreamEntry{
			Level:   "INFO",
			Message: "entry",
			Text:    "entry",
			Source:  "app",
		})
	}

	snapshot := stream.Snapshot(2)
	if len(snapshot) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(snapshot))
	}
	if snapshot[0].ID != 4 || snapshot[1].ID != 5 {
		t.Fatalf("expected latest two entries, got %+v", snapshot)
	}
	if all := stream.Snapshot(0); len(all) != 5 {
		t.Fatalf("expected all entries for zero tail, got %d", len(all))
	}
}

func TestStreamTruncatesOversizedEntry(t *testing.T) {
	t.Parallel()

	stream := NewStream(StreamOptions{MaxBytes: 180})
	entry := stream.Publish(StreamEntry{
		Timestamp: time.Now().UTC().Unix(),
		Level:     "INFO",
		Message:   strings.Repeat("message ", 64),
		Text:      strings.Repeat("text ", 256),
		Source:    "app",
	})

	if size := streamEntrySize(entry); size > stream.maxBytes {
		t.Fatalf("expected entry size <= max bytes, got %d > %d", size, stream.maxBytes)
	}
	if !strings.Contains(entry.Text, "truncated") && !strings.Contains(entry.Message, "truncated") {
		t.Fatalf("expected entry text or message to be truncated, got %+v", entry)
	}
	if snapshot := stream.Snapshot(0); len(snapshot) != 1 {
		t.Fatalf("expected oversized latest entry to be retained after truncation, got %d", len(snapshot))
	}
}

func TestStreamPrunesExpiredEntriesByRetention(t *testing.T) {
	t.Parallel()

	stream := NewStream(StreamOptions{MaxBytes: 16 * 1024, Retention: time.Hour})
	now := time.Now().UTC()
	recentTimestamp := now.Add(-30 * time.Minute).Unix()
	stream.Publish(StreamEntry{
		Timestamp: now.Add(-2 * time.Hour).Unix(),
		Level:     "INFO",
		Message:   "expired",
		Text:      "expired",
		Source:    "app",
	})
	stream.Publish(StreamEntry{
		Timestamp: recentTimestamp,
		Level:     "INFO",
		Message:   "recent",
		Text:      "recent",
		Source:    "app",
	})

	snapshot := stream.Snapshot(10)
	if len(snapshot) != 1 {
		t.Fatalf("expected 1 retained entry, got %d", len(snapshot))
	}
	if snapshot[0].Timestamp != recentTimestamp {
		t.Fatalf("expected retained entry to be the recent log, got %+v", snapshot[0])
	}
}

func TestStreamDropsSlowSubscriber(t *testing.T) {
	t.Parallel()

	stream := NewStream(StreamOptions{MaxBytes: 16 * 1024})
	subscription := stream.Subscribe()

	for range defaultSubscriberSize + 1 {
		stream.Publish(StreamEntry{
			Level:   "INFO",
			Message: "entry",
			Text:    "entry",
			Source:  "app",
		})
	}

	if reason := subscription.Reason(); reason != "slow_consumer" {
		t.Fatalf("expected slow_consumer reason, got %q", reason)
	}
	if count := stream.SubscriberCount(); count != 0 {
		t.Fatalf("expected no subscribers, got %d", count)
	}
}

func TestStreamHandlerSanitizesAndRendersEntry(t *testing.T) {
	t.Parallel()

	stream := NewStream(StreamOptions{MaxBytes: 16 * 1024})
	handler := NewStreamHandler(stream, "app")
	logger := slog.New(handler)

	logger.InfoContext(
		context.Background(),
		"line one\nline two",
		"request_id", "req-123",
		"log_source", "http_access",
		"secret", "should-hide",
		"payload", map[string]any{"status": 200},
	)

	entries := stream.Snapshot(1)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	entry := entries[0]
	if entry.Source != "http_access" {
		t.Fatalf("expected source override, got %q", entry.Source)
	}
	if strings.Contains(entry.Text, "should-hide") {
		t.Fatalf("expected sensitive value to be hidden, got %q", entry.Text)
	}
	if strings.Contains(entry.Text, "\n") {
		t.Fatalf("expected single line text, got %q", entry.Text)
	}
	if !strings.Contains(entry.Text, "request_id=req-123") {
		t.Fatalf("expected request id in text, got %q", entry.Text)
	}
}

func TestStreamHandlerRendersErrorValuesAsMessages(t *testing.T) {
	t.Parallel()

	stream := NewStream(StreamOptions{MaxBytes: 16 * 1024})
	logger := slog.New(NewStreamHandler(stream, "app"))

	logger.ErrorContext(
		context.Background(),
		"failed to write auth log",
		"error", fmt.Errorf("users: failed to insert auth log: %w", errors.New("constraint failed: FOREIGN KEY constraint failed (787)")),
		"event_type", "refresh_failed",
	)

	entries := stream.Snapshot(1)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	entry := entries[0]
	if strings.Contains(entry.Text, "error={}") {
		t.Fatalf("expected error details instead of empty object, got %q", entry.Text)
	}
	if !strings.Contains(entry.Text, `error="users: failed to insert auth log: constraint failed: FOREIGN KEY constraint failed (787)"`) {
		t.Fatalf("expected rendered error message in text, got %q", entry.Text)
	}
}

func TestStreamHandlerRendersGroupsAndLogValuers(t *testing.T) {
	t.Parallel()

	stream := NewStream(StreamOptions{MaxBytes: 16 * 1024})
	logger := slog.New(NewStreamHandler(stream, "app"))

	logger.InfoContext(
		context.Background(),
		"render structured values",
		slog.Group("request", slog.String("id", "req-1"), slog.Int("status", 200)),
		slog.Any("lazy", slog.StringValue("resolved")),
	)

	entries := stream.Snapshot(1)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	entry := entries[0]
	if !strings.Contains(entry.Text, "request.id=req-1") || !strings.Contains(entry.Text, "request.status=200") {
		t.Fatalf("expected flattened group fields in text, got %q", entry.Text)
	}
	if !strings.Contains(entry.Text, "lazy=resolved") {
		t.Fatalf("expected resolved log valuer in text, got %q", entry.Text)
	}
}

func TestLogStartupBanner(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{}))

	LogStartupBanner(logger, ":8080", "/tmp/data")

	output := buf.String()
	if strings.Count(output, bannerDivider) != 2 {
		t.Fatalf("expected divider twice, got %q", output)
	}
	if !strings.Contains(output, "listen: http://localhost:8080") {
		t.Fatalf("expected listen line in banner, got %q", output)
	}
	if !strings.Contains(output, "data_dir: /tmp/data") {
		t.Fatalf("expected data_dir line in banner, got %q", output)
	}
}
