package httpapi

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"main/internal/accountpolicies"
	"main/internal/auth"
	"main/internal/authorization"
	"main/internal/identity"
	"main/internal/logging"
	"main/internal/setup"
)

type API struct {
	auth          *auth.Service
	authorization *authorization.Service
	identities    *identity.Service
	setup         *setup.Service
	policies      *accountpolicies.Service
	dataDir       string
	logger        *slog.Logger
	logStream     *logging.Stream
}

type viewerIdentityResponse struct {
	ID        int64   `json:"id"`
	Username  string  `json:"username"`
	Email     *string `json:"email"`
	AvatarURL *string `json:"avatarUrl"`
	Status    string  `json:"status"`
}

// viewerAuthorizationResponse is the transport shape of a permission set.
//
// It is a separate type from authorization.ViewerAuthorization on purpose: the
// domain model stores typed capabilities, while the wire contract stays a list of
// strings. Renaming a JSON field therefore never reaches the permission model.
type viewerAuthorizationResponse struct {
	Role         string   `json:"role"`
	Capabilities []string `json:"capabilities"`
}

type viewerResponse struct {
	Identity      viewerIdentityResponse      `json:"identity"`
	Authorization viewerAuthorizationResponse `json:"authorization"`
}

type sessionResponse struct {
	AccessToken string         `json:"accessToken"`
	TokenType   string         `json:"tokenType"`
	ExpiresAt   time.Time      `json:"expiresAt"`
	Viewer      viewerResponse `json:"viewer"`
}

type managedUserResponse struct {
	ID           int64      `json:"id"`
	Username     string     `json:"username"`
	Email        *string    `json:"email"`
	AvatarURL    *string    `json:"avatarUrl"`
	Role         string     `json:"role"`
	Status       string     `json:"status"`
	LastActiveAt *time.Time `json:"lastActiveAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	Actions      []string   `json:"actions"`
}

func toViewerAuthorizationResponse(viewer authorization.ViewerAuthorization) viewerAuthorizationResponse {
	capabilities := make([]string, len(viewer.Capabilities))
	for index, capability := range viewer.Capabilities {
		capabilities[index] = string(capability)
	}

	return viewerAuthorizationResponse{
		Role:         viewer.Role,
		Capabilities: capabilities,
	}
}

type installStateResponse struct {
	SetupCompleted bool       `json:"setupCompleted"`
	OwnerUserID    *int64     `json:"ownerUserId"`
	CompletedAt    *time.Time `json:"completedAt"`
}

func (api *API) currentActor(ctx context.Context) (auth.Actor, bool) {
	return auth.ActorFromContext(ctx)
}

func (api *API) toViewerResponse(viewer auth.Viewer) viewerResponse {
	return viewerResponse{
		Identity: viewerIdentityResponse{
			ID:        viewer.User.ID,
			Username:  viewer.User.Username,
			Email:     viewer.User.Email,
			AvatarURL: api.toAvatarURL(viewer.User.AvatarPath),
			Status:    viewer.User.Status,
		},
		Authorization: toViewerAuthorizationResponse(viewer.Authorization),
	}
}

func (api *API) toSessionResponse(result auth.SessionResult) sessionResponse {
	return sessionResponse{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
		ExpiresAt:   result.ExpiresAt,
		Viewer:      api.toViewerResponse(result.Viewer),
	}
}

func (api *API) toManagedUserResponse(user identity.User, actions []identity.ManagedUserAction) managedUserResponse {
	return managedUserResponse{
		ID:           user.ID,
		Username:     user.Username,
		Email:        user.Email,
		AvatarURL:    api.toAvatarURL(user.AvatarPath),
		Role:         user.Role,
		Status:       user.Status,
		LastActiveAt: user.LastActiveAt,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Actions:      managedUserActionsToStrings(actions),
	}
}

// managedUserActionsToStrings always returns a non-nil slice so the response
// encodes an empty array instead of null.
func managedUserActionsToStrings(actions []identity.ManagedUserAction) []string {
	values := make([]string, len(actions))
	for index, action := range actions {
		values[index] = string(action)
	}

	return values
}

func (api *API) toAvatarURL(avatarPath *string) *string {
	if avatarPath == nil {
		return nil
	}
	filename := filepath.Base(strings.TrimSpace(*avatarPath))
	if filename == "" || filename == "." {
		return nil
	}
	url := "/api/avatars/" + filename
	return &url
}
