package httpapi

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// pngSignature is the prefix http.DetectContentType keys on to report image/png.
var pngSignature = []byte("\x89PNG\r\n\x1a\n")

// TestAvatarUpload classifies every client-side upload failure the way the UI
// expects, and keeps the happy path working. The 400/413 split comes from
// unwrapping *http.MaxBytesError and http.ErrNotMultipart rather than matching
// error text, which is exactly the kind of mapping that silently regresses to a
// 500 on an untouched call site.
func TestAvatarUpload(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)
	session, _ := performSetup(t, ctx)

	t.Run("non multipart body", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/account/avatar", strings.NewReader(`{"avatar":"x"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+session.AccessToken)

		rec := httptest.NewRecorder()
		ctx.handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
		}
		decodeErrorCode(t, rec.Body.Bytes(), string(CodeInvalidRequest))
	})

	t.Run("multipart body without the avatar part", func(t *testing.T) {
		req := newAvatarUploadRequest(t, session.AccessToken, func(writer *multipart.Writer) {
			if err := writer.WriteField("other", "value"); err != nil {
				t.Fatalf("write form field: %v", err)
			}
		})

		rec := httptest.NewRecorder()
		ctx.handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
		}
		decodeErrorCode(t, rec.Body.Bytes(), string(CodeAvatarRequired))
	})

	t.Run("body over the upload limit", func(t *testing.T) {
		req := newAvatarUploadRequest(t, session.AccessToken, func(writer *multipart.Writer) {
			part, err := writer.CreateFormFile("avatar", "oversized.png")
			if err != nil {
				t.Fatalf("create form file: %v", err)
			}
			if _, err := part.Write(bytes.Repeat([]byte("A"), int(maxAvatarUploadBodySize))); err != nil {
				t.Fatalf("write form file: %v", err)
			}
		})

		rec := httptest.NewRecorder()
		ctx.handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected status %d, got %d: %s", http.StatusRequestEntityTooLarge, rec.Code, rec.Body.String())
		}
		decodeErrorCode(t, rec.Body.Bytes(), string(CodeAvatarTooLarge))
	})

	t.Run("valid image is stored", func(t *testing.T) {
		uploadAvatarForTest(t, ctx, session.AccessToken)
	})
}

func newAvatarUploadRequest(t *testing.T, token string, writeParts func(*multipart.Writer)) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	writeParts(writer)
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/account/avatar", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	return req
}

// TestAvatarServingIsPublicAndCacheable locks the cache contract: every upload gets
// a new URL, and that URL must be fetchable without a bearer token and cacheable by
// browsers and shared caches. A missing file must stay outside that contract so a
// deleted avatar is not pinned in a cache for a year.
func TestAvatarServingIsPublicAndCacheable(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)
	session, _ := performSetup(t, ctx)

	firstURL := uploadAvatarForTest(t, ctx, session.AccessToken)
	secondURL := uploadAvatarForTest(t, ctx, session.AccessToken)
	if firstURL == secondURL {
		t.Fatalf("expected a new avatar URL for every upload, got %q twice", firstURL)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, secondURL, nil)
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: expected status %d, got %d: %s", secondURL, http.StatusOK, rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != avatarCacheControl {
		t.Fatalf("GET %s: Cache-Control = %q, want %q", secondURL, got, avatarCacheControl)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("GET %s: X-Content-Type-Options = %q, want nosniff", secondURL, got)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "image/png") {
		t.Fatalf("GET %s: Content-Type = %q, want image/png", secondURL, got)
	}

	missingReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/avatars/missing.png", nil)
	missingRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(missingRec, missingReq)

	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("GET missing avatar: expected status %d, got %d", http.StatusNotFound, missingRec.Code)
	}
	if got := missingRec.Header().Get("Cache-Control"); got == avatarCacheControl {
		t.Fatalf("missing avatar must not be cached as immutable, got Cache-Control = %q", got)
	}
}

func uploadAvatarForTest(t *testing.T, ctx *testContext, token string) string {
	t.Helper()

	req := newAvatarUploadRequest(t, token, func(writer *multipart.Writer) {
		part, err := writer.CreateFormFile("avatar", "avatar.png")
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := part.Write(append(pngSignature, make([]byte, 32)...)); err != nil {
			t.Fatalf("write form file: %v", err)
		}
	})

	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload avatar: expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var envelope testSuccessEnvelope[viewerResponse]
	decodeJSONResponse(t, rec.Body.Bytes(), &envelope)
	avatarURL := envelope.Data.Identity.AvatarURL
	if avatarURL == nil || *avatarURL == "" {
		t.Fatalf("upload avatar: expected a stored avatar URL, got %v", avatarURL)
	}

	return *avatarURL
}
