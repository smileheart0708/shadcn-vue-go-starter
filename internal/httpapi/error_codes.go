package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
)

// ErrorCode is the machine-readable identifier carried in `error.code` of every
// failed API response. It is the contract with the frontend: this set must match
// the `apiError` block of web/src/locales/{zh-CN,en-US}.json exactly.
// error_codes_test.go fails the build when the two drift apart, so adding a code
// here without adding its translations (and vice versa) is not possible.
type ErrorCode string

const (
	CodeInvalidRequest             ErrorCode = "invalid_request"
	CodeInvalidCredentials         ErrorCode = "invalid_credentials"
	CodeInvalidToken               ErrorCode = "invalid_token"
	CodeUnauthorized               ErrorCode = "unauthorized"
	CodeForbidden                  ErrorCode = "forbidden"
	CodeAccountDisabled            ErrorCode = "account_disabled"
	CodeRegistrationDisabled       ErrorCode = "registration_disabled"
	CodeAccountDeleteForbidden     ErrorCode = "account_delete_forbidden"
	CodeUsernameRequired           ErrorCode = "username_required"
	CodeUsernameTaken              ErrorCode = "username_taken"
	CodeEmailTaken                 ErrorCode = "email_taken"
	CodePasswordTooShort           ErrorCode = "password_too_short"
	CodeCurrentPasswordInvalid     ErrorCode = "current_password_invalid"
	CodeInvalidID                  ErrorCode = "invalid_id"
	CodeUserNotFound               ErrorCode = "user_not_found"
	CodeAvatarRequired             ErrorCode = "avatar_required"
	CodeAvatarInvalidType          ErrorCode = "avatar_invalid_type"
	CodeAvatarTooLarge             ErrorCode = "avatar_too_large"
	CodeSetupRequired              ErrorCode = "setup_required"
	CodeSetupCompleted             ErrorCode = "setup_completed"
	CodeSystemLogStreamUnavailable ErrorCode = "system_log_stream_unavailable"
	CodeInternalError              ErrorCode = "internal_error"
)

type errorDefinition struct {
	status int
	// message is a developer-facing fallback (curl, logs, third-party clients).
	// The UI never renders it: the frontend translates `code` itself.
	message string
}

// errorDefinitions is the single source of truth for status + fallback text.
//
// The code set is deliberately narrow: a code is kept only when the caller needs
// different copy or a different follow-up action. Technical failures that all mean
// "the server broke, try again later" collapse into CodeInternalError — their
// per-site detail belongs in the log, not in the response body.
var errorDefinitions = map[ErrorCode]errorDefinition{
	CodeInvalidRequest:             {status: http.StatusBadRequest, message: "Request body is missing or malformed."},
	CodeInvalidCredentials:         {status: http.StatusUnauthorized, message: "Username/email or password is invalid."},
	CodeInvalidToken:               {status: http.StatusUnauthorized, message: "Session is invalid or expired."},
	CodeUnauthorized:               {status: http.StatusUnauthorized, message: "Authentication is required."},
	CodeForbidden:                  {status: http.StatusForbidden, message: "You do not have permission to access this resource."},
	CodeAccountDisabled:            {status: http.StatusForbidden, message: "This account is disabled."},
	CodeRegistrationDisabled:       {status: http.StatusForbidden, message: "Registration is currently disabled."},
	CodeAccountDeleteForbidden:     {status: http.StatusForbidden, message: "This account cannot delete itself."},
	CodeUsernameRequired:           {status: http.StatusBadRequest, message: "Username is required."},
	CodeUsernameTaken:              {status: http.StatusConflict, message: "Username is already in use."},
	CodeEmailTaken:                 {status: http.StatusConflict, message: "Email is already in use."},
	CodePasswordTooShort:           {status: http.StatusBadRequest, message: "Password must be at least 8 characters."},
	CodeCurrentPasswordInvalid:     {status: http.StatusBadRequest, message: "Current password is incorrect."},
	CodeInvalidID:                  {status: http.StatusBadRequest, message: "Invalid resource identifier."},
	CodeUserNotFound:               {status: http.StatusNotFound, message: "User not found."},
	CodeAvatarRequired:             {status: http.StatusBadRequest, message: "Avatar file is required."},
	CodeAvatarInvalidType:          {status: http.StatusBadRequest, message: "Avatar file must be a JPG, PNG, or WebP image."},
	CodeAvatarTooLarge:             {status: http.StatusRequestEntityTooLarge, message: "Avatar file must be 5MB or smaller."},
	CodeSetupRequired:              {status: http.StatusServiceUnavailable, message: "The application must be initialized before this endpoint can be used."},
	CodeSetupCompleted:             {status: http.StatusConflict, message: "The application has already been initialized."},
	CodeSystemLogStreamUnavailable: {status: http.StatusServiceUnavailable, message: "System log stream is unavailable."},
	CodeInternalError:              {status: http.StatusInternalServerError, message: "The server encountered an unexpected error."},
}

// writeError emits the standard error envelope for code.
//
// Server-side failures are logged with their cause; client-side rejections are
// logged at debug level so bad requests stay diagnosable without flooding the
// log stream. Callers that have a concrete error to explain must pass it.
func writeError(w http.ResponseWriter, r *http.Request, code ErrorCode, cause ...error) {
	definition, known := errorDefinitions[code]
	if !known {
		// An unregistered code would leak an untranslatable value to the client.
		definition = errorDefinitions[CodeInternalError]
		code = CodeInternalError
	}

	logErrorCause(r, code, definition.status, cause)

	writeJSON(w, definition.status, errorResponse{
		Success: false,
		Error: apiError{
			Code:    string(code),
			Message: definition.message,
		},
	})
}

func logErrorCause(r *http.Request, code ErrorCode, status int, cause []error) {
	if len(cause) == 0 {
		return
	}

	ctx := r.Context()
	logger := loggerFromContext(ctx)
	joined := errors.Join(cause...)
	attrs := []any{"code", string(code), "route", r.URL.Path}

	if status >= http.StatusInternalServerError {
		logger.ErrorContext(ctx, "request failed", append(attrs, "error", joined)...)
		return
	}

	logger.DebugContext(ctx, "request rejected", append(attrs, "error", joined)...)
}

type errorLoggerContextKey struct{}

// withErrorLogger exposes the application logger to writeError without threading
// it through every handler and middleware signature.
func withErrorLogger(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), errorLoggerContextKey{}, logger)))
		})
	}
}

func loggerFromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(errorLoggerContextKey{}).(*slog.Logger); ok {
		return logger
	}

	return slog.Default()
}
