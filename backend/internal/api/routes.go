package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kevin/kkcert/internal/auth"
)

func (s *Server) routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	})
	r.Get("/api/openapi.yaml", serveOpenAPI)
	r.Get("/api/docs", serveSwaggerUI)

	r.Route("/api/auth", func(r chi.Router) {
		r.Get("/config", s.authConfig)
		r.Post("/login", s.authLogin)
		r.Get("/oidc/login", s.oidcLogin)
		r.Get("/oidc/callback", s.oidcCallback)
		r.Group(func(r chi.Router) {
			r.Use(s.requireAuth)
			r.Get("/me", s.authMe)
			r.Post("/logout", s.authLogout)
		})
	})

	r.Route("/api", func(r chi.Router) {
		r.Use(s.requireAuth)

		r.With(s.requireRole(auth.CanRead)).Get("/domains", s.listDomains)
		r.With(s.requireRole(auth.CanWriteDomain)).Post("/domains", s.createDomains)
		r.With(s.requireRole(auth.CanRead)).Get("/domains/{id}/download", s.downloadDomainCert)
		r.With(s.requireRole(auth.CanWriteDomain)).Post("/domains/{id}/renew", s.renewDomain)
		r.With(s.requireRole(auth.CanWriteDomain)).Delete("/domains/{id}", s.deleteDomain)

		r.Get("/certificates", s.listCertificates)
		r.With(s.requireRole(auth.CanWriteDomain)).Post("/certificates/sync-git", s.syncAllCertsGit)

		r.Get("/settings", s.getSettings)
		r.With(s.requireRole(auth.CanWriteSettings)).Put("/settings", s.putSettings)
		r.With(s.requireRole(auth.CanWriteSettings)).Post("/settings/acme/reset", s.resetACME)

		r.Get("/logs", s.listLogs)

		r.With(s.requireRole(auth.CanWriteDomain)).Post("/check/run", s.runCheck)
		r.With(s.requireRole(auth.CanWriteSettings)).Post("/cleanup/run", s.runCleanup)

		r.Route("/tokens", func(r chi.Router) {
			r.Use(s.requireRole(auth.CanManageUsers))
			r.Get("/", s.listTokens)
			r.Post("/", s.createToken)
			r.Put("/{id}", s.updateToken)
			r.Delete("/{id}", s.deleteToken)
		})

		r.Route("/users", func(r chi.Router) {
			r.Use(s.requireRole(auth.CanManageUsers))
			r.Get("/", s.listUsers)
			r.Post("/", s.createUser)
			r.Put("/{id}", s.updateUser)
			r.Delete("/{id}", s.deleteUser)
		})
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") {
			http.NotFound(w, r)
			return
		}
		if s.static != nil {
			serveSPA(w, r, s.static)
			return
		}
		http.NotFound(w, r)
	})

	return r
}

func (s *Server) authConfig(w http.ResponseWriter, _ *http.Request) {
	settings, err := s.store.GetSettings()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, map[string]bool{"oidc_enabled": settings.OIDCEnabled})
}

func (s *Server) authLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	token, err := auth.LoginLocal(s.store, req.Username, req.Password)
	if err != nil {
		http.Error(w, "invalid credentials", 401)
		return
	}
	writeJSON(w, map[string]string{"token": token})
}

func (s *Server) oidcLogin(w http.ResponseWriter, r *http.Request) {
	url, err := s.oidc.LoginURL(w, r)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

func (s *Server) oidcCallback(w http.ResponseWriter, r *http.Request) {
	token, err := s.oidc.Callback(w, r)
	if err != nil {
		http.Error(w, err.Error(), 401)
		return
	}
	http.Redirect(w, r, "/login?token="+token, http.StatusFound)
}

func (s *Server) authMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, toUserView(principalFrom(r).User))
}

func (s *Server) authLogout(w http.ResponseWriter, r *http.Request) {
	auth.Logout(s.store, auth.BearerToken(r))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) runCheck(w http.ResponseWriter, _ *http.Request) {
	go func() { _ = s.scheduler.RunCheck() }()
	writeJSON(w, map[string]string{"status": "started"})
}

func (s *Server) runCleanup(w http.ResponseWriter, _ *http.Request) {
	go func() { _ = s.scheduler.RunCleanup() }()
	writeJSON(w, map[string]string{"status": "started"})
}
