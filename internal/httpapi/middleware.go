package httpapi

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"main/internal/auth"
	"main/internal/setup"
)

type Middleware func(http.Handler) http.Handler

type requestAuthenticator interface {
	AuthenticateRequest(context.Context, string) (auth.Actor, error)
}

func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	wrapped := handler
	for _, middleware := range slices.Backward(middlewares) {
		wrapped = middleware(wrapped)
	}
	return wrapped
}

func RequireSetupCompleted(setupService *setup.Service) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			state, err := setupService.GetState(r.Context())
			if err != nil {
				writeError(w, r, CodeInternalError, err)
				return
			}
			if !state.SetupCompleted {
				writeError(w, r, CodeSetupRequired)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireAuth(authService *auth.Service) Middleware {
	return RequireAuthWithAuthenticator(authService)
}

func RequireAuthWithAuthenticator(authenticator requestAuthenticator) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, err := authenticator.AuthenticateRequest(r.Context(), r.Header.Get("Authorization"))
			if err != nil {
				if errors.Is(err, auth.ErrMissingBearerToken) {
					writeError(w, r, CodeUnauthorized, err)
					return
				}
				writeError(w, r, CodeInvalidToken, err)
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.WithActor(r.Context(), actor)))
		})
	}
}

func RequireCapability(capability string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := auth.ActorFromContext(r.Context())
			if !ok {
				writeError(w, r, CodeUnauthorized)
				return
			}

			if slices.Contains(actor.Authorization.Capabilities, capability) {
				next.ServeHTTP(w, r)
				return
			}

			writeError(w, r, CodeForbidden)
		})
	}
}
