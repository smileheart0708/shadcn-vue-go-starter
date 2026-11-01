package httpapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"main/internal/authorization"
	"main/internal/avatar"
)

// maxAvatarUploadBodySize leaves room for multipart framing on top of the file limit
// the avatar store enforces.
const maxAvatarUploadBodySize = avatar.MaxFileSizeBytes + (1 << 20)

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

	updated, err := api.identities.UpdateProfile(r.Context(), actor.User.ID, payload.Username, payload.Email)
	if err != nil {
		writeOwnAccountError(w, r, err)
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

	data, err := api.readAvatarUpload(r.Context(), w, r)
	if err != nil {
		writeAvatarUploadError(w, r, err)
		return
	}

	avatarFileName, err := api.avatars.Save(data)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	updated, err := api.identities.UpdateAvatar(r.Context(), actor.User.ID, &avatarFileName)
	if err != nil {
		// The row write failed, so the file just written has no owner.
		api.removeAvatarFile(r.Context(), avatarFileName, "rollback_avatar_upload", actor.User.ID)
		writeOwnAccountError(w, r, err)
		return
	}

	if actor.User.AvatarPath != nil {
		api.removeAvatarFile(r.Context(), *actor.User.AvatarPath, "replace_previous_avatar", actor.User.ID)
	}

	viewer, err := api.auth.BuildViewer(r.Context(), updated.ID)
	if err != nil {
		writeError(w, r, CodeInternalError, err)
		return
	}
	writeSuccessJSON(w, http.StatusOK, api.toViewerResponse(viewer))
}

// readAvatarUpload returns the uploaded bytes. Only the shape of the request is judged
// here; what counts as a valid image is the avatar store's decision.
func (api *API) readAvatarUpload(ctx context.Context, w http.ResponseWriter, r *http.Request) ([]byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarUploadBodySize)

	file, _, err := r.FormFile("avatar")
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			api.loggerOrDefault().WarnContext(ctx, "failed to close avatar upload file", "error", closeErr)
		}
	}()

	// Read one byte past the limit so the store reports ErrTooLarge instead of accepting
	// a silently truncated image.
	return io.ReadAll(io.LimitReader(file, avatar.MaxFileSizeBytes+1))
}

// writeAvatarUploadError classifies the multipart mechanics. Rejections that come from
// the avatar store go through writeDomainError instead: the two layers own separate
// vocabularies on purpose.
func writeAvatarUploadError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, http.ErrMissingFile):
		writeError(w, r, CodeAvatarRequired, err)
	case errors.As(err, new(*http.MaxBytesError)):
		// MaxBytesReader reports overflow as *http.MaxBytesError; matching its text
		// would turn a 413 back into a 500 on any wording change.
		writeError(w, r, CodeAvatarTooLarge, err)
	case errors.Is(err, http.ErrNotMultipart):
		// A non-multipart body is a client mistake, not a server failure.
		writeError(w, r, CodeInvalidRequest, err)
	default:
		writeError(w, r, CodeInternalError, err)
	}
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

	if err := api.auth.ChangePassword(r.Context(), actor, payload.CurrentPassword, payload.NewPassword); err != nil {
		writeOwnAccountError(w, r, err)
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

	if err := api.identities.DeleteUser(r.Context(), actor.User.ID); err != nil {
		// The row disappeared between the capability check and the delete: the session
		// lost its user either way.
		writeOwnAccountError(w, r, err)
		return
	}

	if actor.User.AvatarPath != nil {
		api.removeAvatarFile(r.Context(), *actor.User.AvatarPath, "delete_account_avatar_cleanup", actor.User.ID)
	}

	api.auth.ClearRefreshCookie(w, r)
	writeSuccessJSON(w, http.StatusOK, deleteAccountResponse{Deleted: true})
}

func (api *API) avatarHandler(w http.ResponseWriter, r *http.Request) {
	fullPath, err := api.avatars.Path(r.PathValue("filename"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// ServeFile answers 404 on its own when the row still names a file that is gone.
	http.ServeFile(w, r, fullPath) //nolint:gosec // avatar.Store.Path only accepts bare file names, so the path cannot leave its directory
}

// removeAvatarFile deletes a stored file after the database row already changed, so a
// failure is logged and never surfaced: the request has succeeded by then.
func (api *API) removeAvatarFile(ctx context.Context, storedName string, reason string, userID int64) {
	if storedName == "" {
		return
	}

	if err := api.avatars.Remove(storedName); err != nil {
		api.loggerOrDefault().WarnContext(ctx, "failed to remove avatar file", "reason", reason, "user_id", userID, "error", err)
	}
}

func (api *API) loggerOrDefault() *slog.Logger {
	if api != nil && api.logger != nil {
		return api.logger
	}
	return slog.Default()
}
