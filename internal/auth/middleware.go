package auth

import (
	"context"
	"net/http"
	"strings"

	"Flatly/pkg/httpjson"
)

type contextKey string

const principalKey contextKey = "principal"

func Middleware(service *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := strings.TrimSpace(r.Header.Get("Authorization"))
			if !strings.HasPrefix(header, "Bearer ") {
				httpjson.WriteError(w, http.StatusUnauthorized, "unauthorized", ErrUnauthorized.Error(), nil)
				return
			}

			principal, err := service.ParseToken(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
			if err != nil {
				httpjson.WriteError(w, http.StatusUnauthorized, "unauthorized", ErrUnauthorized.Error(), nil)
				return
			}

			ctx := context.WithValue(r.Context(), principalKey, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(role Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok {
				httpjson.WriteError(w, http.StatusUnauthorized, "unauthorized", ErrUnauthorized.Error(), nil)
				return
			}
			if principal.Role != role {
				httpjson.WriteError(w, http.StatusForbidden, "forbidden", ErrForbidden.Error(), nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey).(Principal)
	return principal, ok
}
