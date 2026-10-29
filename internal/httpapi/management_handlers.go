package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"main/internal/accountpolicies"
	"main/internal/auth"
	"main/internal/authorization"
	"main/internal/identity"
)

type accountPoliciesResponse struct {
	PublicRegistrationEnabled         bool `json:"publicRegistrationEnabled"`
	SelfServiceAccountDeletionEnabled bool `json:"selfServiceAccountDeletionEnabled"`
}

type updateAccountPoliciesRequest struct {
	PublicRegistrationEnabled         *bool `json:"publicRegistrationEnabled"`
	SelfServiceAccountDeletionEnabled *bool `json:"selfServiceAccountDeletionEnabled"`
}

type managementUsersPageResponse struct {
	Items    []managedUserResponse `json:"items"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
	Total    int                   `json:"total"`
}

type createManagementUserRequest struct {
	Username string  `json:"username"`
	Email    *string `json:"email"`
	Password string  `json:"password"`
}

type updateManagementUserRequest struct {
	Username string  `json:"username"`
	Email    *string `json:"email"`
}

func (api *API) getAccountPoliciesHandler(w http.ResponseWriter, r *http.Request) {
	policies, err := api.policies.Get(r.Context())
	if err != nil {
		writeError(w, r, CodeInternalError, err)
		return
	}
	writeSuccessJSON(w, http.StatusOK, toAccountPoliciesResponse(policies))
}

func (api *API) updateAccountPoliciesHandler(w http.ResponseWriter, r *http.Request) {
	var payload updateAccountPoliciesRequest
	if err := decodeJSON(r.Context(), w, r, &payload); err != nil {
		writeError(w, r, CodeInvalidRequest, err)
		return
	}

	policies, err := api.policies.Update(r.Context(), accountpolicies.UpdateInput{
		PublicRegistrationEnabled:         payload.PublicRegistrationEnabled,
		SelfServiceAccountDeletionEnabled: payload.SelfServiceAccountDeletionEnabled,
	})
	if err != nil {
		writeError(w, r, CodeInternalError, err)
		return
	}

	writeSuccessJSON(w, http.StatusOK, toAccountPoliciesResponse(policies))
}

func (api *API) listManagementUsersHandler(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.currentActor(r.Context())
	if !ok {
		writeError(w, r, CodeUnauthorized)
		return
	}

	result, err := api.identities.ListUsers(r.Context(), identity.ListUsersParams{
		Query:    r.URL.Query().Get("q"),
		Status:   r.URL.Query().Get("status"),
		Page:     parsePositiveIntQuery(r, "page", 1),
		PageSize: parsePositiveIntQuery(r, "pageSize", 20),
	})
	if err != nil {
		writeError(w, r, CodeInternalError, err)
		return
	}

	actorTarget := authorization.UserTarget{
		UserID: actor.User.ID,
		Role:   actor.User.Role,
		Status: actor.User.Status,
	}

	items := make([]managedUserResponse, 0, len(result.Items))
	for _, item := range result.Items {
		target := authorization.UserTarget{
			UserID: item.ID,
			Role:   item.Role,
			Status: item.Status,
		}
		items = append(items, api.toManagedUserResponse(item, api.authorization.ManagedUserActions(actorTarget, target)))
	}

	writeSuccessJSON(w, http.StatusOK, managementUsersPageResponse{
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
	})
}

func (api *API) createManagementUserHandler(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.currentActor(r.Context())
	if !ok {
		writeError(w, r, CodeUnauthorized)
		return
	}

	var payload createManagementUserRequest
	if err := decodeJSON(r.Context(), w, r, &payload); err != nil {
		writeError(w, r, CodeInvalidRequest, err)
		return
	}
	if strings.TrimSpace(payload.Username) == "" {
		writeError(w, r, CodeUsernameRequired)
		return
	}
	if len(strings.TrimSpace(payload.Password)) < auth.MinPasswordLength {
		writeError(w, r, CodePasswordTooShort)
		return
	}

	passwordHash, err := auth.HashPassword(payload.Password)
	if err != nil {
		writeError(w, r, CodeInternalError, err)
		return
	}

	created, err := api.identities.CreateUser(r.Context(), identity.CreateUserParams{
		Username:         payload.Username,
		Email:            payload.Email,
		PasswordHash:     passwordHash,
		Role:             identity.RoleUser,
		AssignedByUserID: new(actor.User.ID),
	})
	if err != nil {
		switch {
		case errors.Is(err, identity.ErrUsernameTaken):
			writeError(w, r, CodeUsernameTaken)
		case errors.Is(err, identity.ErrEmailTaken):
			writeError(w, r, CodeEmailTaken)
		default:
			writeError(w, r, CodeInternalError, err)
		}
		return
	}

	writeSuccessJSON(w, http.StatusCreated, api.toManagedUserResponse(created, []string{}))
}

func (api *API) updateManagementUserHandler(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.currentActor(r.Context())
	if !ok {
		writeError(w, r, CodeUnauthorized)
		return
	}

	targetID, ok := parseInt64PathValue(w, r, "id")
	if !ok {
		return
	}

	targetUser, actorTarget, targetTarget, ok := api.loadManagedUserTargets(w, r, actor, targetID)
	if !ok {
		return
	}
	if !api.authorization.CanManageUser(actorTarget, targetTarget, authorization.UserActionUpdate) {
		writeError(w, r, CodeForbidden)
		return
	}

	var payload updateManagementUserRequest
	if err := decodeJSON(r.Context(), w, r, &payload); err != nil {
		writeError(w, r, CodeInvalidRequest, err)
		return
	}
	if strings.TrimSpace(payload.Username) == "" {
		writeError(w, r, CodeUsernameRequired)
		return
	}

	updated, err := api.identities.UpdateManagedUser(r.Context(), targetUser.ID, identity.UpdateManagedUserParams{
		Username: payload.Username,
		Email:    payload.Email,
	})
	if err != nil {
		switch {
		case errors.Is(err, identity.ErrUsernameTaken):
			writeError(w, r, CodeUsernameTaken)
		case errors.Is(err, identity.ErrEmailTaken):
			writeError(w, r, CodeEmailTaken)
		default:
			writeError(w, r, CodeInternalError, err)
		}
		return
	}

	writeSuccessJSON(w, http.StatusOK, api.toManagedUserResponse(updated, api.authorization.ManagedUserActions(actorTarget, authorization.UserTarget{
		UserID: updated.ID,
		Role:   updated.Role,
		Status: updated.Status,
	})))
}

func (api *API) disableManagementUserHandler(w http.ResponseWriter, r *http.Request) {
	api.setManagementUserStatusHandler(w, r, identity.StatusDisabled, authorization.UserActionDisable)
}

func (api *API) enableManagementUserHandler(w http.ResponseWriter, r *http.Request) {
	api.setManagementUserStatusHandler(w, r, identity.StatusActive, authorization.UserActionEnable)
}

func (api *API) setManagementUserStatusHandler(w http.ResponseWriter, r *http.Request, status string, action authorization.UserAction) {
	actor, ok := api.currentActor(r.Context())
	if !ok {
		writeError(w, r, CodeUnauthorized)
		return
	}

	targetID, ok := parseInt64PathValue(w, r, "id")
	if !ok {
		return
	}

	targetUser, actorTarget, targetTarget, ok := api.loadManagedUserTargets(w, r, actor, targetID)
	if !ok {
		return
	}
	if !api.authorization.CanManageUser(actorTarget, targetTarget, action) {
		writeError(w, r, CodeForbidden)
		return
	}

	updated, err := api.identities.SetUserStatus(r.Context(), targetUser.ID, status)
	if err != nil {
		writeError(w, r, CodeInternalError, err)
		return
	}

	writeSuccessJSON(w, http.StatusOK, api.toManagedUserResponse(updated, api.authorization.ManagedUserActions(actorTarget, authorization.UserTarget{
		UserID: updated.ID,
		Role:   updated.Role,
		Status: updated.Status,
	})))
}

func (api *API) loadManagedUserTargets(w http.ResponseWriter, r *http.Request, actor auth.Actor, targetID int64) (identity.User, authorization.UserTarget, authorization.UserTarget, bool) {
	targetUser, err := api.identities.GetUserByID(r.Context(), targetID)
	if err != nil {
		if errors.Is(err, identity.ErrUserNotFound) {
			writeError(w, r, CodeUserNotFound)
		} else {
			writeError(w, r, CodeInternalError, err)
		}
		return identity.User{}, authorization.UserTarget{}, authorization.UserTarget{}, false
	}

	actorTarget := authorization.UserTarget{
		UserID: actor.User.ID,
		Role:   actor.User.Role,
		Status: actor.User.Status,
	}
	targetTarget := authorization.UserTarget{
		UserID: targetUser.ID,
		Role:   targetUser.Role,
		Status: targetUser.Status,
	}
	return targetUser, actorTarget, targetTarget, true
}

func toAccountPoliciesResponse(policies accountpolicies.Policies) accountPoliciesResponse {
	return accountPoliciesResponse{
		PublicRegistrationEnabled:         policies.PublicRegistrationEnabled,
		SelfServiceAccountDeletionEnabled: policies.SelfServiceAccountDeletionEnabled,
	}
}

func parsePositiveIntQuery(r *http.Request, key string, fallback int) int {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseInt64PathValue(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	raw := strings.TrimSpace(r.PathValue(key))
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed <= 0 {
		writeError(w, r, CodeInvalidID)
		return 0, false
	}
	return parsed, true
}
