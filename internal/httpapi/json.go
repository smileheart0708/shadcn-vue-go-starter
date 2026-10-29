package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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

func decodeJSON(ctx context.Context, w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	defer func() {
		if closeErr := r.Body.Close(); closeErr != nil {
			slog.WarnContext(ctx, "failed to close request body", "error", closeErr)
		}
	}()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON object")
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if payload == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func writeSuccessJSON[T any](w http.ResponseWriter, status int, data T) {
	writeJSON(w, status, successResponse[T]{
		Success: true,
		Data:    data,
	})
}
