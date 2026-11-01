package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"main/internal/auth"
	"main/internal/authorization"
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

// RequireAuth authenticates the bearer token and puts the actor on the request
// context. The authenticator is an interface rather than *auth.Service so the gate
// can be exercised without constructing a full auth service.
func RequireAuth(authenticator requestAuthenticator) Middleware {
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

// RequireCapability rejects the request unless the authenticated actor holds
// capability. It only enforces "may this actor perform this class of action";
// "may this actor act on this particular object" belongs to the owning domain.
func RequireCapability(capability authorization.Capability) Middleware {
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

			// The cause is logged, never returned: the client keeps the generic
			// forbidden message so policy details do not leak.
			writeError(w, r, CodeForbidden, fmt.Errorf(
				"capability %q not granted to user %d with role %q",
				capability, actor.User.ID, actor.User.Role,
			))
		})
	}
}
