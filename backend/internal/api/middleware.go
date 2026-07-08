package api

import (
	"context"
	"net/http"

	"github.com/kevin/kkcert/internal/auth"
)

type ctxKey int

const principalCtxKey ctxKey = 1

func withPrincipal(ctx context.Context, p *auth.Principal) context.Context {
	return context.WithValue(ctx, principalCtxKey, p)
}

func principalFrom(r *http.Request) *auth.Principal {
	p, _ := r.Context().Value(principalCtxKey).(*auth.Principal)
	return p
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, err := auth.Authenticate(s.store, r)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), p)))
	})
}

func (s *Server) requireRole(allowed func(string) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := principalFrom(r)
			if p == nil || !allowed(p.User.Role) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
