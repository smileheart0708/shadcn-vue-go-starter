package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"main/internal/accountpolicies"
	"main/internal/auth"
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
	actor, ok := api.requireActor(w, r)
	if !ok {
		return
	}

	result, err := api.identities.ListUsers(r.Context(), identity.ListUsersParams{
		Query:    r.URL.Query().Get("q"),
		Status:   r.URL.Query().Get("status"),
		Page:     parsePositiveIntQuery(r, "page"),
		PageSize: parsePositiveIntQuery(r, "pageSize"),
	})
	if err != nil {
		writeError(w, r, CodeInternalError, err)
		return
	}

	items := make([]managedUserResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, api.managedUserViewOf(actor, item))
	}

	writeSuccessJSON(w, http.StatusOK, managementUsersPageResponse{
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
	})
}

func (api *API) createManagementUserHandler(w http.ResponseWriter, r *http.Request) {
	var payload createManagementUserRequest
	if err := decodeJSON(r.Context(), w, r, &payload); err != nil {
		writeError(w, r, CodeInvalidRequest, err)
		return
	}
	created, err := api.auth.CreateManagedUser(r.Context(), auth.CreateManagedUserParams{
		Username: payload.Username,
		Email:    payload.Email,
		Password: payload.Password,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	// A freshly created account is not manageable by the actor: the actor cannot
	// act on itself, and every other case is filtered by ManagedUserActions.
	writeSuccessJSON(w, http.StatusCreated, api.toManagedUserResponse(created, []identity.ManagedUserAction{}))
}

func (api *API) updateManagementUserHandler(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.requireActor(w, r)
	if !ok {
		return
	}

	targetID, ok := parseInt64PathValue(w, r, "id")
	if !ok {
		return
	}

	targetUser, ok := api.loadManagedUser(w, r, targetID)
	if !ok {
		return
	}
	actorTarget := managedUserTargetOf(actor.User)
	if !identity.CanManageUser(actorTarget, managedUserTargetOf(targetUser), identity.ManagedUserActionUpdate) {
		writeError(w, r, CodeForbidden)
		return
	}

	var payload updateManagementUserRequest
	if err := decodeJSON(r.Context(), w, r, &payload); err != nil {
		writeError(w, r, CodeInvalidRequest, err)
		return
	}
	updated, err := api.identities.UpdateManagedUser(r.Context(), targetUser.ID, identity.UpdateManagedUserParams{
		Username: payload.Username,
		Email:    payload.Email,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	writeSuccessJSON(w, http.StatusOK, api.managedUserViewOf(actor, updated))
}

func (api *API) disableManagementUserHandler(w http.ResponseWriter, r *http.Request) {
	api.setManagementUserStatusHandler(w, r, identity.StatusDisabled, identity.ManagedUserActionDisable)
}

func (api *API) enableManagementUserHandler(w http.ResponseWriter, r *http.Request) {
	api.setManagementUserStatusHandler(w, r, identity.StatusActive, identity.ManagedUserActionEnable)
}

func (api *API) setManagementUserStatusHandler(w http.ResponseWriter, r *http.Request, status string, action identity.ManagedUserAction) {
	actor, ok := api.requireActor(w, r)
	if !ok {
		return
	}

	targetID, ok := parseInt64PathValue(w, r, "id")
	if !ok {
		return
	}

	targetUser, ok := api.loadManagedUser(w, r, targetID)
	if !ok {
		return
	}
	actorTarget := managedUserTargetOf(actor.User)
	if !identity.CanManageUser(actorTarget, managedUserTargetOf(targetUser), action) {
		writeError(w, r, CodeForbidden)
		return
	}

	updated, err := api.identities.SetUserStatus(r.Context(), targetUser.ID, status)
	if err != nil {
		// loadManagedUser already succeeded, so a not-found here means the target was
		// deleted in between. The mapping table answers 404, which keeps the race from
		// being misreported as a server fault.
		writeDomainError(w, r, err)
		return
	}

	writeSuccessJSON(w, http.StatusOK, api.managedUserViewOf(actor, updated))
}

func (api *API) loadManagedUser(w http.ResponseWriter, r *http.Request, targetID int64) (identity.User, bool) {
	targetUser, err := api.identities.GetUserByID(r.Context(), targetID)
	if err != nil {
		writeDomainError(w, r, err)
		return identity.User{}, false
	}
	return targetUser, true
}

// managedUserTargetOf is the single conversion from a user to a permission target.
// The same three fields feed both ManagedUserActions and CanManageUser, so assembling
// the struct by hand at each call site is how the list and the single-user response
// drift apart.
func managedUserTargetOf(user identity.User) identity.ManagedUserTarget {
	return identity.ManagedUserTarget{
		UserID: user.ID,
		Role:   user.Role,
		Status: user.Status,
	}
}

// managedUserViewOf keeps every response that carries an action list on one path.
func (api *API) managedUserViewOf(actor auth.Actor, user identity.User) managedUserResponse {
	return api.toManagedUserResponse(user, identity.ManagedUserActions(
		managedUserTargetOf(actor.User),
		managedUserTargetOf(user),
	))
}

func toAccountPoliciesResponse(policies accountpolicies.Policies) accountPoliciesResponse {
	return accountPoliciesResponse{
		PublicRegistrationEnabled:         policies.PublicRegistrationEnabled,
		SelfServiceAccountDeletionEnabled: policies.SelfServiceAccountDeletionEnabled,
	}
}

// parsePositiveIntQuery answers 0 for a value that is absent or unusable. The default
// and the upper bound live in identity.normalizeListParams only: repeating them here
// is how a paging contract ends up with two different sets of numbers.
func parsePositiveIntQuery(r *http.Request, key string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get(key)))
	if err != nil || parsed <= 0 {
		return 0
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
