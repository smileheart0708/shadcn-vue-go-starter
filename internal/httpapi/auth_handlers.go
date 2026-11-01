package httpapi

import (
	"errors"
	"net/http"

	"main/internal/auth"
	"main/internal/setup"
)

type installSetupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type registerRequest struct {
	Username string  `json:"username"`
	Email    *string `json:"email"`
	Password string  `json:"password"`
}

type logoutResponse struct {
	LoggedOut bool `json:"loggedOut"`
}

type passwordChangedResponse struct {
	PasswordChanged bool `json:"passwordChanged"`
}

func (api *API) installStateHandler(w http.ResponseWriter, r *http.Request) {
	state, err := api.setup.GetState(r.Context())
	if err != nil {
		writeError(w, r, CodeInternalError, err)
		return
	}

	writeSuccessJSON(w, http.StatusOK, installStateResponse{
		SetupCompleted: state.SetupCompleted,
		OwnerUserID:    state.OwnerUserID,
		CompletedAt:    state.SetupCompletedAt,
	})
}

func (api *API) installSetupHandler(w http.ResponseWriter, r *http.Request) {
	var payload installSetupRequest
	if err := decodeJSON(r.Context(), w, r, &payload); err != nil {
		writeError(w, r, CodeInvalidRequest, err)
		return
	}
	owner, err := api.setup.Complete(r.Context(), setup.CompleteSetupInput{
		Username: payload.Username,
		Password: payload.Password,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	ip, userAgent := requestMetadata(r)
	session, err := api.auth.IssueSessionForUser(r.Context(), owner, ip, userAgent)
	if err != nil {
		writeError(w, r, CodeInternalError, err)
		return
	}

	api.auth.SetRefreshCookie(w, r, session.RefreshToken)
	writeSuccessJSON(w, http.StatusCreated, api.toSessionResponse(session))
}

func (api *API) publicAuthConfigHandler(w http.ResponseWriter, r *http.Request) {
	config, err := api.policies.PublicAuthConfig(r.Context())
	if err != nil {
		writeError(w, r, CodeInternalError, err)
		return
	}
	writeSuccessJSON(w, http.StatusOK, config)
}

func (api *API) loginHandler(w http.ResponseWriter, r *http.Request) {
	var payload loginRequest
	if err := decodeJSON(r.Context(), w, r, &payload); err != nil {
		writeError(w, r, CodeInvalidRequest, err)
		return
	}

	ip, userAgent := requestMetadata(r)
	result, err := api.auth.Login(r.Context(), payload.Identifier, payload.Password, ip, userAgent)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	api.auth.SetRefreshCookie(w, r, result.RefreshToken)
	writeSuccessJSON(w, http.StatusOK, api.toSessionResponse(result))
}

func (api *API) registerHandler(w http.ResponseWriter, r *http.Request) {
	var payload registerRequest
	if err := decodeJSON(r.Context(), w, r, &payload); err != nil {
		writeError(w, r, CodeInvalidRequest, err)
		return
	}
	ip, userAgent := requestMetadata(r)
	result, err := api.auth.Register(r.Context(), auth.RegisterParams{
		Username: payload.Username,
		Email:    payload.Email,
		Password: payload.Password,
	}, ip, userAgent)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	api.auth.SetRefreshCookie(w, r, result.RefreshToken)
	writeSuccessJSON(w, http.StatusCreated, api.toSessionResponse(result))
}

func (api *API) refreshHandler(w http.ResponseWriter, r *http.Request) {
	rawToken, err := api.auth.ReadRefreshCookie(r)
	if err != nil {
		api.auth.ClearRefreshCookie(w, r)
		writeError(w, r, CodeInvalidToken, err)
		return
	}

	result, err := api.auth.Refresh(r.Context(), rawToken)
	if err != nil {
		// Only a session the client must discard clears the cookie and answers 401.
		// On infrastructure failures the cookie is kept so a later page load can recover;
		// the SPA still redirects for this request because it resets on any refresh error.
		if auth.IsSessionInvalidError(err) {
			api.auth.ClearRefreshCookie(w, r)
			writeError(w, r, CodeInvalidToken, err)
			return
		}
		writeError(w, r, CodeInternalError, err)
		return
	}

	api.auth.SetRefreshCookie(w, r, result.RefreshToken)
	writeSuccessJSON(w, http.StatusOK, api.toSessionResponse(result))
}

func (api *API) logoutHandler(w http.ResponseWriter, r *http.Request) {
	rawToken, err := api.auth.ReadRefreshCookie(r)
	if err == nil {
		if logoutErr := api.auth.Logout(r.Context(), rawToken); logoutErr != nil && !errors.Is(logoutErr, auth.ErrInvalidRefreshToken) {
			writeError(w, r, CodeInternalError, logoutErr)
			return
		}
	}

	api.auth.ClearRefreshCookie(w, r)
	writeSuccessJSON(w, http.StatusOK, logoutResponse{LoggedOut: true})
}

func (api *API) meHandler(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.currentActor(r.Context())
	if !ok {
		writeError(w, r, CodeUnauthorized)
		return
	}

	writeSuccessJSON(w, http.StatusOK, api.toViewerResponse(auth.Viewer{
		User:          actor.User,
		Authorization: actor.Authorization,
	}))
}
