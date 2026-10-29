package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestErrorCodesMatchFrontendLocale is the only place the backend/frontend error
// contract is enforced. Adding a code without its translations fails here, and so
// does leaving a translation behind after the code is gone.
func TestErrorCodesMatchFrontendLocale(t *testing.T) {
	t.Parallel()

	for _, filename := range []string{"zh-CN.json", "en-US.json"} {
		t.Run(filename, func(t *testing.T) {
			t.Parallel()

			translations := readAPIErrorMessages(t, filename)

			for code := range errorDefinitions {
				if _, ok := translations[string(code)]; !ok {
					t.Errorf("error code %q is registered in Go but missing from %s apiError", code, filename)
				}
			}

			for code, message := range translations {
				if _, ok := errorDefinitions[ErrorCode(code)]; !ok {
					t.Errorf("%s apiError contains %q which is not registered in Go", filename, code)
				}
				if strings.TrimSpace(message) == "" {
					t.Errorf("%s apiError[%q] is empty", filename, code)
				}
			}
		})
	}
}

func TestErrorDefinitionsAreConsistent(t *testing.T) {
	t.Parallel()

	if _, ok := errorDefinitions[CodeInternalError]; !ok {
		t.Fatal("internal_error must always be registered: it is the fallback for unknown codes")
	}

	for code, definition := range errorDefinitions {
		if definition.status < http.StatusBadRequest || definition.status > http.StatusNetworkAuthenticationRequired {
			t.Errorf("error code %q has unexpected status %d", code, definition.status)
		}
		if strings.TrimSpace(definition.message) == "" {
			t.Errorf("error code %q has an empty fallback message", code)
		}
	}
}

func TestWriteErrorFallsBackToInternalError(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)

	writeError(rec, req, ErrorCode("not_registered"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if payload.Error.Code != string(CodeInternalError) {
		t.Fatalf("expected code %q, got %q", CodeInternalError, payload.Error.Code)
	}
}

// readAPIErrorMessages keeps only the flattened `apiError.*` leaves of a locale
// file. The frontend keys that block by the raw error code, so the result maps
// onto errorDefinitions without any case or shape conversion.
func readAPIErrorMessages(t *testing.T, filename string) map[string]string {
	t.Helper()

	messages := make(map[string]string)
	for key, message := range readLocaleMessages(t, filename) {
		if code, ok := strings.CutPrefix(key, "apiError."); ok {
			messages[code] = message
		}
	}
	if len(messages) == 0 {
		t.Fatalf("frontend locale file %s has no apiError block", filename)
	}

	return messages
}
