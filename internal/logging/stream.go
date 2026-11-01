package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	DefaultStreamMaxBytes = 1 * 1024 * 1024
	DefaultRetention      = 7 * 24 * time.Hour
	defaultSubscriberSize = 128
	streamEntryBaseSize   = 64
	truncationMarker      = " [truncated]"
)

type StreamEntry struct {
	ID        int64  `json:"id"`
	Timestamp int64  `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Text      string `json:"text"`
	Source    string `json:"source"`
}

type StreamOptions struct {
	MaxBytes  int
	Retention time.Duration
}

// Stream is an in-memory buffer of recent log entries served to the UI.
//
// Exactly two rules drop entries: the retention window and the byte budget
// (maxBytes). There is deliberately no entry-count cap: the byte budget already
// bounds memory, so a second bound would only make pruning depend on message
// size twice.
type Stream struct {
	mu          sync.RWMutex
	nextEntryID int64
	nextSubID   int64
	maxBytes    int
	retention   time.Duration
	entries     []StreamEntry
	entrySizes  []int
	// headIndex is the first live entry. Slots before it are dropped but still
	// occupy the backing slice until compactIfNeededLocked reclaims them, so
	// every read must start here rather than at entries[0].
	headIndex   int
	buffered    int
	subscribers map[int64]*StreamSubscription
}

type StreamSubscription struct {
	id      int64
	stream  *Stream
	entries chan StreamEntry

	mu     sync.RWMutex
	reason string
	closed bool
}

func NewStream(options StreamOptions) *Stream {
	maxBytes := options.MaxBytes
	if maxBytes <= 0 {
		maxBytes = DefaultStreamMaxBytes
	}
	retention := options.Retention
	if retention <= 0 {
		retention = DefaultRetention
	}

	return &Stream{
		maxBytes:    maxBytes,
		retention:   retention,
		entries:     make([]StreamEntry, 0, 128),
		entrySizes:  make([]int, 0, 128),
		subscribers: make(map[int64]*StreamSubscription),
	}
}

func (s *Stream) Publish(entry StreamEntry) StreamEntry {
	if s == nil {
		return entry
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	s.nextEntryID++
	entry.ID = s.nextEntryID
	if entry.Timestamp <= 0 {
		entry.Timestamp = now.Unix()
	}
	if entry.Level == "" {
		entry.Level = strings.ToUpper(slog.LevelInfo.String())
	}
	if entry.Source == "" {
		entry.Source = "app"
	}
	entry = truncateStreamEntry(entry, s.maxBytes)

	s.entries = append(s.entries, entry)
	entrySize := streamEntrySize(entry)
	s.entrySizes = append(s.entrySizes, entrySize)
	s.buffered += entrySize
	s.pruneLocked(now)

	for id, subscription := range s.subscribers {
		select {
		case subscription.entries <- entry:
		default:
			subscription.closeLocked("slow_consumer")
			delete(s.subscribers, id)
		}
	}

	return entry
}

func (s *Stream) Snapshot(tail int) []StreamEntry {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.pruneLocked(time.Now().UTC())
	activeLen := s.activeLenLocked()
	if tail <= 0 || tail > activeLen {
		tail = activeLen
	}

	start := s.headIndex + activeLen - tail
	snapshot := make([]StreamEntry, tail)
	copy(snapshot, s.entries[start:s.headIndex+activeLen])
	return snapshot
}

func (s *Stream) pruneLocked(now time.Time) {
	if s == nil || len(s.entries) == s.headIndex {
		return
	}

	if s.retention > 0 {
		cutoff := now.Add(-s.retention).Unix()
		for s.headIndex < len(s.entries) && s.entries[s.headIndex].Timestamp < cutoff {
			s.dropOldestLocked(1)
		}
	}
	for s.buffered > s.maxBytes && s.headIndex < len(s.entries) {
		s.dropOldestLocked(1)
	}
	s.compactIfNeededLocked()
}

func (s *Stream) activeLenLocked() int {
	if s == nil || s.headIndex >= len(s.entries) {
		return 0
	}
	return len(s.entries) - s.headIndex
}

func (s *Stream) dropOldestLocked(count int) {
	for count > 0 && s.headIndex < len(s.entries) {
		s.buffered -= s.entrySizes[s.headIndex]
		s.entries[s.headIndex] = StreamEntry{}
		s.entrySizes[s.headIndex] = 0
		s.headIndex++
		count--
	}
	if s.buffered < 0 {
		s.buffered = 0
	}
}

func (s *Stream) compactIfNeededLocked() {
	if s == nil || s.headIndex == 0 {
		return
	}
	if s.headIndex == len(s.entries) {
		s.entries = s.entries[:0]
		s.entrySizes = s.entrySizes[:0]
		s.headIndex = 0
		return
	}
	if s.headIndex < 1024 && s.headIndex < len(s.entries)/2 {
		return
	}

	activeEntries := copy(s.entries, s.entries[s.headIndex:])
	clear(s.entries[activeEntries:])
	s.entries = s.entries[:activeEntries]

	activeSizes := copy(s.entrySizes, s.entrySizes[s.headIndex:])
	clear(s.entrySizes[activeSizes:])
	s.entrySizes = s.entrySizes[:activeSizes]
	s.headIndex = 0
}

func truncateStreamEntry(entry StreamEntry, maxBytes int) StreamEntry {
	if maxBytes <= 0 {
		return entry
	}
	for streamEntrySize(entry) > maxBytes {
		overflow := streamEntrySize(entry) - maxBytes
		field := &entry.Text
		currentLength := len(entry.Text)
		if len(entry.Message) > currentLength {
			field = &entry.Message
			currentLength = len(entry.Message)
		}
		if len(entry.Source) > currentLength {
			field = &entry.Source
			currentLength = len(entry.Source)
		}
		if len(entry.Level) > currentLength {
			field = &entry.Level
			currentLength = len(entry.Level)
		}
		if currentLength == 0 {
			break
		}

		targetLength := max(currentLength-overflow, 0)
		nextValue := truncateStringBytes(*field, targetLength)
		if len(nextValue) >= currentLength {
			nextValue = ""
		}
		*field = nextValue
	}
	return entry
}

func streamEntrySize(entry StreamEntry) int {
	return streamEntryBaseSize + len(entry.Level) + len(entry.Message) + len(entry.Text) + len(entry.Source)
}

func truncateStringBytes(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	if maxBytes <= 0 {
		return ""
	}
	if maxBytes <= len(truncationMarker) {
		return truncationMarker[:maxBytes]
	}

	prefixLength := maxBytes - len(truncationMarker)
	for prefixLength > 0 && prefixLength < len(value) && !utf8.RuneStart(value[prefixLength]) {
		prefixLength--
	}
	return value[:prefixLength] + truncationMarker
}

func (s *Stream) Subscribe() *StreamSubscription {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextSubID++
	subscription := &StreamSubscription{
		id:      s.nextSubID,
		stream:  s,
		entries: make(chan StreamEntry, defaultSubscriberSize),
	}
	s.subscribers[subscription.id] = subscription
	return subscription
}

func (s *Stream) Unsubscribe(subscription *StreamSubscription, reason string) {
	if s == nil || subscription == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.subscribers[subscription.id]
	if !ok || current != subscription {
		return
	}

	subscription.closeLocked(reason)
	delete(s.subscribers, subscription.id)
}

func (s *Stream) SubscriberCount() int {
	if s == nil {
		return 0
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.subscribers)
}

func (s *StreamSubscription) Entries() <-chan StreamEntry {
	if s == nil {
		return nil
	}

	return s.entries
}

func (s *StreamSubscription) Close(reason string) {
	if s == nil || s.stream == nil {
		return
	}

	s.stream.Unsubscribe(s, reason)
}

func (s *StreamSubscription) Reason() string {
	if s == nil {
		return ""
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reason
}

func (s *StreamSubscription) closeLocked(reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		if s.reason == "" && reason != "" {
			s.reason = reason
		}
		return
	}

	s.closed = true
	s.reason = reason
	close(s.entries)
}

type streamHandler struct {
	stream      *Stream
	defaultSrc  string
	prefixAttrs []scopedAttr
	groups      []string
}

type scopedAttr struct {
	groups []string
	attr   slog.Attr
}

type renderedAttr struct {
	key         string
	value       string
	sourceValue string
	hidden      bool
}

func NewStreamHandler(stream *Stream, source string) slog.Handler {
	if source == "" {
		source = "app"
	}

	return &streamHandler{
		stream:     stream,
		defaultSrc: source,
	}
}

func (h *streamHandler) Enabled(context.Context, slog.Level) bool {
	return h.stream != nil
}

func (h *streamHandler) Handle(_ context.Context, record slog.Record) error {
	if h == nil || h.stream == nil {
		return nil
	}

	rendered := make([]renderedAttr, 0, len(h.prefixAttrs)+8)
	for _, attr := range h.prefixAttrs {
		rendered = appendRenderedAttrs(rendered, attr.groups, attr.attr)
	}
	record.Attrs(func(attr slog.Attr) bool {
		rendered = appendRenderedAttrs(rendered, h.groups, attr)
		return true
	})

	source := h.defaultSrc
	parts := make([]string, 0, len(rendered)+1)
	message := sanitizeLogText(record.Message)
	if message != "" {
		parts = append(parts, message)
	}

	for _, attr := range rendered {
		if attr.sourceValue != "" {
			source = attr.sourceValue
		}
		if attr.hidden || attr.key == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", attr.key, attr.value))
	}

	text := strings.TrimSpace(strings.Join(parts, " "))
	if text == "" {
		text = message
	}

	h.stream.Publish(StreamEntry{
		Timestamp: record.Time.Unix(),
		Level:     strings.ToUpper(record.Level.String()),
		Message:   message,
		Text:      sanitizeLogText(text),
		Source:    source,
	})
	return nil
}

func (h *streamHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	clone := h.clone()
	for _, attr := range attrs {
		clone.prefixAttrs = append(clone.prefixAttrs, scopedAttr{
			groups: append([]string(nil), h.groups...),
			attr:   attr,
		})
	}
	return clone
}

func (h *streamHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	clone := h.clone()
	clone.groups = append(append([]string(nil), h.groups...), name)
	return clone
}

func (h *streamHandler) clone() *streamHandler {
	if h == nil {
		return &streamHandler{}
	}

	return &streamHandler{
		stream:      h.stream,
		defaultSrc:  h.defaultSrc,
		prefixAttrs: append([]scopedAttr(nil), h.prefixAttrs...),
		groups:      append([]string(nil), h.groups...),
	}
}

func appendRenderedAttrs(items []renderedAttr, groups []string, attr slog.Attr) []renderedAttr {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return items
	}

	if attr.Value.Kind() == slog.KindGroup {
		nextGroups := groups
		if attr.Key != "" {
			nextGroups = append(append([]string(nil), groups...), attr.Key)
		}
		for _, item := range attr.Value.Group() {
			items = appendRenderedAttrs(items, nextGroups, item)
		}
		return items
	}

	key := attr.Key
	if len(groups) > 0 {
		key = strings.Join(append(append([]string(nil), groups...), key), ".")
	}

	if hiddenLogKey(key) {
		return items
	}

	value := sanitizeLogText(renderSlogValue(attr.Value))
	rendered := renderedAttr{
		key:   key,
		value: value,
	}
	switch key {
	case "log_source", "source":
		rendered.sourceValue = value
		rendered.hidden = true
	}
	return append(items, rendered)
}

func renderSlogValue(value slog.Value) string {
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindString:
		return value.String()
	case slog.KindBool:
		if value.Bool() {
			return "true"
		}
		return "false"
	case slog.KindInt64:
		return fmt.Sprintf("%d", value.Int64())
	case slog.KindUint64:
		return fmt.Sprintf("%d", value.Uint64())
	case slog.KindFloat64:
		return fmt.Sprintf("%v", value.Float64())
	case slog.KindDuration:
		return value.Duration().String()
	case slog.KindTime:
		return value.Time().Format(time.RFC3339Nano)
	case slog.KindGroup:
		return renderSlogGroup(value.Group())
	case slog.KindLogValuer:
		return renderSlogValue(value.Resolve())
	case slog.KindAny:
		fallthrough
	default:
		raw := value.Any()
		if err, ok := raw.(error); ok {
			data, marshalErr := json.Marshal(err.Error())
			if marshalErr != nil {
				return err.Error()
			}
			return string(data)
		}
		data, err := json.Marshal(raw)
		if err != nil {
			return fmt.Sprintf("%v", raw)
		}
		return string(data)
	}
}

func renderSlogGroup(attrs []slog.Attr) string {
	payload := make(map[string]any, len(attrs))
	for _, attr := range attrs {
		payload[attr.Key] = renderSlogValue(attr.Value)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf("%v", payload)
	}
	return string(data)
}

func hiddenLogKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	if lower == "" {
		return true
	}

	sensitiveParts := []string{"token", "secret", "password", "authorization", "cookie"}
	for _, part := range sensitiveParts {
		if strings.Contains(lower, part) {
			return true
		}
	}
	return false
}

func sanitizeLogText(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.Join(strings.Fields(value), " ")
}
