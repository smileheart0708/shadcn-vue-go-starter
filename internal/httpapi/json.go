package httpapi

import (
	"context"
	jsonv2 "encoding/json/v2"
	"log/slog"
	"net/http"
)

const maxJSONBodyBytes int64 = 1 << 20

type errorResponse struct {
	Success bool     `json:"success"`
	Error   apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type successResponse[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
}

// decodeJSON decodes the single JSON value in the request body through
// encoding/json/v2 rather than encoding/json. v2 rejects duplicate member names
// and invalid UTF-8, and it requires the input to hold exactly one JSON value
// followed by optional whitespace. That last property is why the hand-written
// "must contain a single JSON object" check the v1 decoder needed is gone: v2
// reports trailing bytes as an error by itself.
func decodeJSON(ctx context.Context, w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	defer func() {
		if closeErr := r.Body.Close(); closeErr != nil {
			slog.WarnContext(ctx, "failed to close request body", "error", closeErr)
		}
	}()

	return jsonv2.UnmarshalRead(r.Body, dst, jsonv2.RejectUnknownMembers(true))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	// Marshal before touching the header: once WriteHeader has put the status
	// line on the wire, an encoding failure can no longer become a clean 500.
	// This branch is reachable: v2 refuses a value it has no encoding for, such
	// as invalid UTF-8 in a string or a time.Duration (which lost its numeric
	// v1 default), so it must not be deleted as dead code.
	data, err := jsonv2.Marshal(payload)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	// The status is already committed, so a failed write cannot change the
	// response any more; the access log records the request, and this line keeps
	// the write failure from disappearing entirely.
	if _, writeErr := w.Write(data); writeErr != nil {
		slog.Warn("failed to write JSON response", "error", writeErr)
	}
}

func writeSuccessJSON[T any](w http.ResponseWriter, status int, data T) {
	writeJSON(w, status, successResponse[T]{
		Success: true,
		Data:    data,
	})
}
