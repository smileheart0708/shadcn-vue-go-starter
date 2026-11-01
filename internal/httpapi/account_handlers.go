package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"main/internal/auth"
	"main/internal/authorization"
	"main/internal/identity"
)

const (
	maxAvatarFileSizeBytes  = 5 * 1024 * 1024
	maxAvatarUploadBodySize = maxAvatarFileSizeBytes + (1 << 20)
	avatarDirPermissions    = 0o750
	avatarFilePermissions   = 0o600
)

type updateProfileRequest struct {
	Username string  `json:"username"`
	Email    *string `json:"email"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type deleteAccountResponse struct {
	Deleted bool `json:"deleted"`
}

func (api *API) updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.currentActor(r.Context())
	if !ok {
		writeError(w, r, CodeUnauthorized)
		return
	}

	var payload updateProfileRequest
	if err := decodeJSON(r.Context(), w, r, &payload); err != nil {
		writeError(w, r, CodeInvalidRequest, err)
		return
	}
	if strings.TrimSpace(payload.Username) == "" {
		writeError(w, r, CodeUsernameRequired)
		return
	}

	updated, err := api.identities.UpdateProfile(r.Context(), actor.User.ID, payload.Username, payload.Email)
	if err != nil {
		switch {
		case errors.Is(err, identity.ErrUsernameTaken):
			writeError(w, r, CodeUsernameTaken)
		case errors.Is(err, identity.ErrEmailTaken):
			writeError(w, r, CodeEmailTaken)
		case errors.Is(err, identity.ErrUserNotFound):
			writeError(w, r, CodeUnauthorized)
		default:
			writeError(w, r, CodeInternalError, err)
		}
		return
	}

	viewer, err := api.auth.BuildViewer(r.Context(), updated.ID)
	if err != nil {
		writeError(w, r, CodeInternalError, err)
		return
	}
	writeSuccessJSON(w, http.StatusOK, api.toViewerResponse(viewer))
}

func (api *API) updateAvatarHandler(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.currentActor(r.Context())
	if !ok {
		writeError(w, r, CodeUnauthorized)
		return
	}

	currentAvatarPath := ""
	if actor.User.AvatarPath != nil {
		currentAvatarPath = avatarPathFromStoredName(api.dataDir, *actor.User.AvatarPath)
	}

	avatarFileName, err := api.storeAvatarUpload(r.Context(), w, r)
	if err != nil {
		// errors.AsType replaces the "declare, then match" idiom and avoids the temp var.
		if avatarError, isUploadError := errors.AsType[*avatarUploadError](err); isUploadError {
			writeError(w, r, avatarError.code, avatarError)
			return
		}
		writeError(w, r, CodeInternalError, err)
		return
	}

	updated, err := api.identities.UpdateAvatar(r.Context(), actor.User.ID, &avatarFileName)
	if err != nil {
		api.removeAvatarFile(r.Context(), filepath.Join(identity.AvatarDir(api.dataDir), avatarFileName), "rollback_avatar_upload", actor.User.ID)
		writeError(w, r, CodeInternalError, err)
		return
	}

	if currentAvatarPath != "" {
		api.removeAvatarFile(r.Context(), currentAvatarPath, "replace_previous_avatar", actor.User.ID)
	}

	viewer, err := api.auth.BuildViewer(r.Context(), updated.ID)
	if err != nil {
		writeError(w, r, CodeInternalError, err)
		return
	}
	writeSuccessJSON(w, http.StatusOK, api.toViewerResponse(viewer))
}

func (api *API) changePasswordHandler(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.currentActor(r.Context())
	if !ok {
		writeError(w, r, CodeUnauthorized)
		return
	}

	var payload changePasswordRequest
	if err := decodeJSON(r.Context(), w, r, &payload); err != nil {
		writeError(w, r, CodeInvalidRequest, err)
		return
	}

	err := api.auth.ChangePassword(r.Context(), actor, payload.CurrentPassword, payload.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrCurrentPasswordInvalid):
			writeError(w, r, CodeCurrentPasswordInvalid)
		case errors.Is(err, auth.ErrPasswordTooShort):
			writeError(w, r, CodePasswordTooShort)
		default:
			writeError(w, r, CodeInternalError, err)
		}
		return
	}

	api.auth.ClearRefreshCookie(w, r)
	writeSuccessJSON(w, http.StatusOK, passwordChangedResponse{PasswordChanged: true})
}

func (api *API) deleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.currentActor(r.Context())
	if !ok {
		writeError(w, r, CodeUnauthorized)
		return
	}
	// Deliberately checked here instead of via RequireCapability middleware:
	// account.delete_self is derived from account_policies at request time, and a
	// denial must surface CodeAccountDeleteForbidden rather than a generic 403 so
	// the UI can explain which policy is off. Do not "unify" this into the
	// middleware without keeping both behaviours.
	if !api.authorization.HasCapability(actor.Authorization.Capabilities, authorization.CapabilityAccountDeleteSelf) {
		writeError(w, r, CodeAccountDeleteForbidden)
		return
	}

	currentAvatarPath := ""
	if actor.User.AvatarPath != nil {
		currentAvatarPath = avatarPathFromStoredName(api.dataDir, *actor.User.AvatarPath)
	}

	if err := api.identities.DeleteUser(r.Context(), actor.User.ID); err != nil {
		// The row disappeared between the capability check and the delete. Report it the
		// same way as a missing viewer elsewhere in this file: the session is gone.
		if errors.Is(err, identity.ErrUserNotFound) {
			writeError(w, r, CodeUnauthorized)
			return
		}
		writeError(w, r, CodeInternalError, err)
		return
	}

	if currentAvatarPath != "" {
		api.removeAvatarFile(r.Context(), currentAvatarPath, "delete_account_avatar_cleanup", actor.User.ID)
	}

	api.auth.ClearRefreshCookie(w, r)
	writeSuccessJSON(w, http.StatusOK, deleteAccountResponse{Deleted: true})
}

func (api *API) avatarHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
	fullPath, err := resolveAvatarPath(api.dataDir, filename)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if _, statErr := os.Stat(fullPath); statErr != nil {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, fullPath)
}

// avatarUploadError carries a client-facing rejection out of storeAvatarUpload.
// It only holds the code: status and fallback text live in errorDefinitions.
type avatarUploadError struct {
	code    ErrorCode
	message string
}

func (err *avatarUploadError) Error() string {
	return err.message
}

func (api *API) storeAvatarUpload(ctx context.Context, w http.ResponseWriter, r *http.Request) (string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarUploadBodySize)

	file, _, err := r.FormFile("avatar")
	if err != nil {
		switch {
		case errors.Is(err, http.ErrMissingFile):
			return "", &avatarUploadError{code: CodeAvatarRequired, message: "avatar file is required"}
		// MaxBytesReader reports overflow as *http.MaxBytesError; matching its
		// text would silently turn a 413 back into a 500 on any wording change.
		case errors.As(err, new(*http.MaxBytesError)):
			return "", &avatarUploadError{code: CodeAvatarTooLarge, message: "avatar file exceeds 5MB"}
		case errors.Is(err, http.ErrNotMultipart):
			// A non-multipart body is a client mistake, not a server failure.
			return "", &avatarUploadError{code: CodeInvalidRequest, message: "request body is not multipart form data"}
		default:
			return "", fmt.Errorf("read avatar upload: %w", err)
		}
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			api.loggerOrDefault().WarnContext(ctx, "failed to close avatar upload file", "error", closeErr)
		}
	}()

	data, err := io.ReadAll(io.LimitReader(file, maxAvatarFileSizeBytes+1))
	if err != nil {
		return "", fmt.Errorf("read avatar file: %w", err)
	}
	if len(data) == 0 {
		return "", &avatarUploadError{code: CodeAvatarRequired, message: "avatar file is required"}
	}
	if len(data) > maxAvatarFileSizeBytes {
		return "", &avatarUploadError{code: CodeAvatarTooLarge, message: "avatar file exceeds 5MB"}
	}

	contentType := http.DetectContentType(data[:min(512, len(data))])
	extension, allowed := avatarExtensionForContentType(contentType)
	if !allowed {
		return "", &avatarUploadError{code: CodeAvatarInvalidType, message: "avatar file is not an allowed image type"}
	}

	fileName, err := randomAvatarFileName(extension)
	if err != nil {
		return "", err
	}

	avatarDir := identity.AvatarDir(api.dataDir)
	if mkdirErr := os.MkdirAll(avatarDir, avatarDirPermissions); mkdirErr != nil {
		return "", fmt.Errorf("create avatar dir: %w", mkdirErr)
	}

	fullPath, err := resolveAvatarPath(api.dataDir, fileName)
	if err != nil {
		return "", err
	}
	if writeErr := os.WriteFile(fullPath, data, avatarFilePermissions); writeErr != nil {
		return "", fmt.Errorf("write avatar file: %w", writeErr)
	}

	return fileName, nil
}

func avatarExtensionForContentType(contentType string) (string, bool) {
	switch contentType {
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}

func randomAvatarFileName(extension string) (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate avatar filename: %w", err)
	}
	return hex.EncodeToString(randomBytes) + extension, nil
}

func resolveAvatarPath(dataDir string, storedName string) (string, error) {
	trimmed := strings.TrimSpace(storedName)
	if trimmed == "" {
		return "", fmt.Errorf("avatar path is empty")
	}
	if trimmed != filepath.Base(trimmed) || strings.Contains(trimmed, `\`) {
		return "", fmt.Errorf("avatar path contains invalid separators")
	}

	avatarDir := filepath.Clean(identity.AvatarDir(dataDir))
	fullPath := filepath.Join(avatarDir, trimmed)
	relativePath, err := filepath.Rel(avatarDir, fullPath)
	if err != nil {
		return "", fmt.Errorf("resolve avatar path: %w", err)
	}
	if relativePath == "." || relativePath == "" || strings.HasPrefix(relativePath, "..") || filepath.IsAbs(relativePath) {
		return "", fmt.Errorf("avatar path escapes avatar dir")
	}
	return fullPath, nil
}

func avatarPathFromStoredName(dataDir string, storedName string) string {
	fullPath, err := resolveAvatarPath(dataDir, storedName)
	if err != nil {
		return ""
	}
	return fullPath
}

func (api *API) removeAvatarFile(ctx context.Context, fullPath string, reason string, userID int64) {
	if strings.TrimSpace(fullPath) == "" {
		return
	}
	if err := os.Remove(fullPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		api.loggerOrDefault().WarnContext(ctx, "failed to remove avatar file", "reason", reason, "user_id", userID, "file_name", filepath.Base(fullPath), "error", err)
	}
}

func (api *API) loggerOrDefault() *slog.Logger {
	if api != nil && api.logger != nil {
		return api.logger
	}
	return slog.Default()
}
