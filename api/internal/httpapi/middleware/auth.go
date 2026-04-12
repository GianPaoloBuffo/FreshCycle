package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/auth"
)

type contextKey string

const authenticatedUserContextKey contextKey = "freshcycle.authenticated_user"

func RequireAuth(validator auth.Validator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if validator == nil {
				writeAuthError(writer, http.StatusServiceUnavailable, "auth_unavailable", "The API auth validator is not configured yet.")
				return
			}

			token := strings.TrimSpace(strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer "))
			user, err := validator.ValidateAccessToken(request.Context(), token)
			if err != nil {
				switch {
				case errors.Is(err, auth.ErrMissingAccessToken):
					writeAuthError(writer, http.StatusUnauthorized, "auth_required", "Sign in again before calling the FreshCycle API.")
				case errors.Is(err, auth.ErrInvalidAccessToken):
					writeAuthError(writer, http.StatusUnauthorized, "invalid_token", "Your session could not be validated. Sign in again and retry.")
				case errors.Is(err, auth.ErrAuthNotConfigured):
					writeAuthError(writer, http.StatusServiceUnavailable, "auth_unavailable", "The API auth validator is not configured yet.")
				default:
					writeAuthError(writer, http.StatusBadGateway, "auth_failed", "FreshCycle could not validate your session token.")
				}
				return
			}

			next.ServeHTTP(writer, request.WithContext(context.WithValue(request.Context(), authenticatedUserContextKey, user)))
		})
	}
}

func AuthenticatedUserFromContext(ctx context.Context) (auth.User, bool) {
	user, ok := ctx.Value(authenticatedUserContextKey).(auth.User)
	return user, ok
}

func writeAuthError(writer http.ResponseWriter, status int, code string, message string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]string{
		"error":   code,
		"message": message,
	})
}
