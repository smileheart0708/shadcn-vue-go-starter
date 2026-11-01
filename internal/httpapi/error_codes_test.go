package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"main/internal/identity"
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

// TestDomainErrorMappingsAreRegistered keeps the mapping table honest: a typo in a code
// would otherwise surface as an untranslatable value on the wire.
func TestDomainErrorMappingsAreRegistered(t *testing.T) {
	t.Parallel()

	for _, mapping := range domainErrorMappings {
		if mapping.target == nil {
			t.Errorf("mapping to %q has a nil target error", mapping.code)
			continue
		}
		if _, ok := errorDefinitions[mapping.code]; !ok {
			t.Errorf("domain error %v maps to unregistered code %q", mapping.target, mapping.code)
		}
	}
}

// TestWriteDomainErrorMapsWrappedErrors is the guard for the whole table: mapping must
// survive wrapping, a missing row keeps the table's 404, and an unknown error stays a 500.
func TestWriteDomainErrorMapsWrappedErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want ErrorCode
	}{
		{
			name: "wrapped domain error",
			err:  fmt.Errorf("store: %w", identity.ErrUsernameTaken),
			want: CodeUsernameTaken,
		},
		{
			name: "missing user keeps the table entry",
			err:  identity.ErrUserNotFound,
			want: CodeUserNotFound,
		},
		{
			name: "unmapped error falls back to internal",
			err:  errors.New("boom"),
			want: CodeInternalError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			writeDomainError(rec, httptest.NewRequest(http.MethodGet, "/api/test", nil), test.err)

			decodeErrorCode(t, rec.Body.Bytes(), string(test.want))
		})
	}
}

// TestWriteOwnAccountErrorTreatsMissingRowAsDeadSession locks the only documented
// exception to the table. If the exception ever spreads beyond the own-account
// endpoints, this test keeps it visible instead of silently changing management 404s.
func TestWriteOwnAccountErrorTreatsMissingRowAsDeadSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want ErrorCode
	}{
		{
			name: "own row vanished means the session lost its user",
			err:  fmt.Errorf("identity: lookup: %w", identity.ErrUserNotFound),
			want: CodeUnauthorized,
		},
		{
			name: "other domain errors keep their code",
			err:  identity.ErrEmailTaken,
			want: CodeEmailTaken,
		},
		{
			name: "unmapped error stays internal",
			err:  errors.New("boom"),
			want: CodeInternalError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			writeOwnAccountError(rec, httptest.NewRequest(http.MethodGet, "/api/account/profile", nil), test.err)

			decodeErrorCode(t, rec.Body.Bytes(), string(test.want))
		})
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
