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
		req := httptest.NewRequest(http.MethodPost, "/api/account/avatar", strings.NewReader(`{"avatar":"x"}`))
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
		req := newAvatarUploadRequest(t, session.AccessToken, func(writer *multipart.Writer) {
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
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var envelope testSuccessEnvelope[viewerResponse]
		decodeJSONResponse(t, rec.Body.Bytes(), &envelope)
		avatarURL := envelope.Data.Identity.AvatarURL
		if avatarURL == nil || *avatarURL == "" {
			t.Fatalf("expected a stored avatar URL, got %v", avatarURL)
		}
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

	req := httptest.NewRequest(http.MethodPost, "/api/account/avatar", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	return req
}
