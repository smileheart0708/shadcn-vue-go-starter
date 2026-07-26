package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"main/internal/auth"
	"main/internal/identity"
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
		writeAPIError(w, http.StatusInternalServerError, "setup_state_unavailable", "Failed to load install state.")
		return
	}

	writeSuccessJSON(w, http.StatusOK, installStateResponse{
		SetupState:     state.SetupState,
		SetupCompleted: state.SetupCompleted,
		OwnerUserID:    state.OwnerUserID,
		CompletedAt:    state.SetupCompletedAt,
	})
}

func (api *API) installSetupHandler(w http.ResponseWriter, r *http.Request) {
	var payload installSetupRequest
	if err := decodeJSON(r.Context(), w, r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", describeJSONDecodeError(err))
		return
	}
	if strings.TrimSpace(payload.Username) == "" {
		writeAPIError(w, http.StatusBadRequest, "username_required", "Username is required.")
		return
	}
	if len(strings.TrimSpace(payload.Password)) < auth.MinPasswordLength {
		writeAPIError(w, http.StatusBadRequest, "password_too_short", "Password must be at least 8 characters.")
		return
	}

	owner, err := api.setup.Complete(r.Context(), setup.CompleteSetupInput{
		Username: payload.Username,
		Password: payload.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, setup.ErrAlreadyCompleted):
			writeAPIError(w, http.StatusConflict, "setup_completed", "The application has already been initialized.")
		case errors.Is(err, identity.ErrUsernameTaken):
			writeAPIError(w, http.StatusConflict, "username_taken", "Username is already in use.")
		default:
			writeAPIError(w, http.StatusInternalServerError, "setup_failed", "Failed to initialize the application.")
		}
		return
	}

	ip, userAgent := requestMetadata(r)
	session, err := api.auth.IssueSessionForUser(r.Context(), owner, ip, userAgent)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "session_create_failed", "Failed to create owner session.")
		return
	}

	api.auth.SetRefreshCookie(w, r, session.RefreshToken)
	writeSuccessJSON(w, http.StatusCreated, api.toSessionResponse(session))
}

func (api *API) publicAuthConfigHandler(w http.ResponseWriter, r *http.Request) {
	config, err := api.policies.PublicAuthConfig(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "public_auth_config_unavailable", "Failed to load authentication configuration.")
		return
	}
	writeSuccessJSON(w, http.StatusOK, config)
}

func (api *API) loginHandler(w http.ResponseWriter, r *http.Request) {
	var payload loginRequest
	if err := decodeJSON(r.Context(), w, r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", describeJSONDecodeError(err))
		return
	}

	ip, userAgent := requestMetadata(r)
	result, err := api.auth.Login(r.Context(), payload.Identifier, payload.Password, ip, userAgent)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			writeAPIError(w, http.StatusUnauthorized, "invalid_credentials", "Username/email or password is invalid.")
		case errors.Is(err, auth.ErrAccountDisabled):
			writeAPIError(w, http.StatusForbidden, "account_disabled", "This account is disabled.")
		default:
			writeAPIError(w, http.StatusInternalServerError, "login_failed", "Failed to authenticate user.")
		}
		return
	}

	api.auth.SetRefreshCookie(w, r, result.RefreshToken)
	writeSuccessJSON(w, http.StatusOK, api.toSessionResponse(result))
}

func (api *API) registerHandler(w http.ResponseWriter, r *http.Request) {
	var payload registerRequest
	if err := decodeJSON(r.Context(), w, r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", describeJSONDecodeError(err))
		return
	}
	if strings.TrimSpace(payload.Username) == "" {
		writeAPIError(w, http.StatusBadRequest, "username_required", "Username is required.")
		return
	}

	ip, userAgent := requestMetadata(r)
	result, err := api.auth.Register(r.Context(), auth.RegisterParams{
		Username: payload.Username,
		Email:    payload.Email,
		Password: payload.Password,
	}, ip, userAgent)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrRegistrationDisabled):
			writeAPIError(w, http.StatusForbidden, "registration_disabled", "Registration is currently disabled.")
		case errors.Is(err, auth.ErrPasswordTooShort):
			writeAPIError(w, http.StatusBadRequest, "password_too_short", "Password must be at least 8 characters.")
		case errors.Is(err, identity.ErrUsernameTaken):
			writeAPIError(w, http.StatusConflict, "username_taken", "Username is already in use.")
		case errors.Is(err, identity.ErrEmailTaken):
			writeAPIError(w, http.StatusConflict, "email_taken", "Email is already in use.")
		default:
			writeAPIError(w, http.StatusInternalServerError, "registration_failed", "Failed to register user.")
		}
		return
	}

	api.auth.SetRefreshCookie(w, r, result.RefreshToken)
	writeSuccessJSON(w, http.StatusCreated, api.toSessionResponse(result))
}

func (api *API) refreshHandler(w http.ResponseWriter, r *http.Request) {
	rawToken, err := api.auth.ReadRefreshCookie(r)
	if err != nil {
		api.auth.ClearRefreshCookie(w, r)
		writeAPIError(w, http.StatusUnauthorized, "invalid_refresh_token", "Refresh session is invalid or expired.")
		return
	}

	result, err := api.auth.Refresh(r.Context(), rawToken)
	if err != nil {
		api.auth.ClearRefreshCookie(w, r)
		writeAPIError(w, http.StatusUnauthorized, "invalid_refresh_token", "Refresh session is invalid or expired.")
		return
	}

	api.auth.SetRefreshCookie(w, r, result.RefreshToken)
	writeSuccessJSON(w, http.StatusOK, api.toSessionResponse(result))
}

func (api *API) logoutHandler(w http.ResponseWriter, r *http.Request) {
	rawToken, err := api.auth.ReadRefreshCookie(r)
	if err == nil {
		if logoutErr := api.auth.Logout(r.Context(), rawToken); logoutErr != nil && !errors.Is(logoutErr, auth.ErrInvalidRefreshToken) {
			writeAPIError(w, http.StatusInternalServerError, "logout_failed", "Failed to revoke refresh session.")
			return
		}
	}

	api.auth.ClearRefreshCookie(w, r)
	writeSuccessJSON(w, http.StatusOK, logoutResponse{LoggedOut: true})
}

func (api *API) meHandler(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.currentActor(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	writeSuccessJSON(w, http.StatusOK, api.toViewerResponse(auth.Viewer{
		User:          actor.User,
		Authorization: actor.Authorization,
	}))
}
