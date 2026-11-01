package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type decodeJSONTarget struct {
	Name string `json:"name"`
}

// TestDecodeJSONRejectsAmbiguousBodies pins the transport contract that
// encoding/json/v2 buys: one JSON value, no unknown members, no duplicate keys,
// no invalid UTF-8. v1 silently kept the last duplicate and replaced invalid
// UTF-8 instead of failing, so this test is what stops a regression to the v1
// decoder.
func TestDecodeJSONRejectsAmbiguousBodies(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body string
	}{
		{name: "unknown member", body: `{"name":"a","role":"owner"}`},
		{name: "duplicate member", body: `{"name":"a","name":"b"}`},
		{name: "trailing value", body: `{"name":"a"}{"name":"b"}`},
		{name: "invalid utf-8", body: "{\"name\":\"\xff\"}"},
		{name: "empty body", body: ""},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(testCase.body))
			if err := decodeJSON(request.Context(), httptest.NewRecorder(), request, &decodeJSONTarget{}); err == nil {
				t.Fatalf("expected decodeJSON to reject %q", testCase.body)
			}
		})
	}
}

func TestDecodeJSONAcceptsOneObjectWithSurroundingWhitespace(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(" {\"name\":\"panda\"}\n\t "))
	target := decodeJSONTarget{}
	if err := decodeJSON(request.Context(), httptest.NewRecorder(), request, &target); err != nil {
		t.Fatalf("decodeJSON() error = %v", err)
	}
	if target.Name != "panda" {
		t.Fatalf("expected name %q, got %q", "panda", target.Name)
	}
}
