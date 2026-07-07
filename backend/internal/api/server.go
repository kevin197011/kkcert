package api

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kevin/kkcert/internal/auth"
	"github.com/kevin/kkcert/internal/certsvc"
	"github.com/kevin/kkcert/internal/scheduler"
	"github.com/kevin/kkcert/internal/store"
)

type Server struct {
	store     *store.Store
	scheduler *scheduler.Scheduler
	oidc      *auth.OIDCHandler
	static    fs.FS
}

func NewServer(st *store.Store, sched *scheduler.Scheduler, static fs.FS) *Server {
	return &Server{
		store:     st,
		scheduler: sched,
		oidc:      &auth.OIDCHandler{Store: st},
		static:    static,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/health" {
		writeJSON(w, map[string]string{"status": "ok"})
		return
	}
	if r.URL.Path == "/api/openapi.yaml" {
		serveOpenAPI(w, r)
		return
	}
	if r.URL.Path == "/api/docs" {
		serveSwaggerUI(w, r)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/api/auth/") {
		s.handleAuth(w, r)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/api/") {
		principal, err := auth.Authenticate(s.store, r)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		s.handleAPI(w, r, principal)
		return
	}

	if s.static != nil {
		serveSPA(w, r, s.static)
		return
	}
	http.NotFound(w, r)
}

func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/auth")
	switch {
	case path == "/config" && r.Method == http.MethodGet:
		settings, err := s.store.GetSettings()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, map[string]bool{"oidc_enabled": settings.OIDCEnabled})
	case path == "/login" && r.Method == http.MethodPost:
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
	case path == "/oidc/login" && r.Method == http.MethodGet:
		url, err := s.oidc.LoginURL(w, r)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		http.Redirect(w, r, url, http.StatusFound)
	case path == "/oidc/callback" && r.Method == http.MethodGet:
		token, err := s.oidc.Callback(w, r)
		if err != nil {
			http.Error(w, err.Error(), 401)
			return
		}
		http.Redirect(w, r, "/login?token="+token, http.StatusFound)
	case path == "/me" && r.Method == http.MethodGet:
		principal, err := auth.Authenticate(s.store, r)
		if err != nil {
			http.Error(w, "unauthorized", 401)
			return
		}
		writeJSON(w, toUserView(principal.User))
	case path == "/logout" && r.Method == http.MethodPost:
		auth.Logout(s.store, auth.BearerToken(r))
		w.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(w, r)
	}
}

type userDTO struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	AuthType string `json:"auth_type"`
	Enabled  bool   `json:"enabled"`
}

func toUserView(u store.User) userDTO {
	return userDTO{ID: u.ID, Username: u.Username, Email: u.Email, Role: u.Role, AuthType: u.AuthType, Enabled: u.Enabled}
}

func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request, p *auth.Principal) {
	path := strings.TrimPrefix(r.URL.Path, "/api")
	role := p.User.Role

	switch {
	case path == "/domains" && r.Method == http.MethodGet:
		if !auth.CanRead(role) {
			http.Error(w, "forbidden", 403)
			return
		}
		s.listDomains(w)
	case path == "/domains" && r.Method == http.MethodPost:
		if !auth.CanWriteDomain(role) {
			http.Error(w, "forbidden", 403)
			return
		}
		s.createDomains(w, r)
	case strings.HasPrefix(path, "/domains/") && strings.HasSuffix(path, "/renew") && r.Method == http.MethodPost:
		if !auth.CanWriteDomain(role) {
			http.Error(w, "forbidden", 403)
			return
		}
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/domains/"), "/renew")
		s.renewDomain(w, id)
	case strings.HasPrefix(path, "/domains/") && r.Method == http.MethodDelete:
		if !auth.CanWriteDomain(role) {
			http.Error(w, "forbidden", 403)
			return
		}
		id := strings.TrimPrefix(path, "/domains/")
		s.deleteDomain(w, id)
	case path == "/certificates" && r.Method == http.MethodGet:
		s.listCertificates(w)
	case path == "/certificates/sync-git" && r.Method == http.MethodPost:
		if !auth.CanWriteDomain(role) {
			http.Error(w, "forbidden", 403)
			return
		}
		s.syncAllCertsGit(w)
	case path == "/settings" && r.Method == http.MethodGet:
		s.getSettings(w)
	case path == "/settings" && r.Method == http.MethodPut:
		if !auth.CanWriteSettings(role) {
			http.Error(w, "forbidden", 403)
			return
		}
		s.putSettings(w, r)
	case path == "/settings/acme/reset" && r.Method == http.MethodPost:
		if !auth.CanWriteSettings(role) {
			http.Error(w, "forbidden", 403)
			return
		}
		s.resetACME(w, r)
	case path == "/logs" && r.Method == http.MethodGet:
		s.listLogs(w)
	case path == "/check/run" && r.Method == http.MethodPost:
		if !auth.CanWriteDomain(role) {
			http.Error(w, "forbidden", 403)
			return
		}
		go func() { _ = s.scheduler.RunCheck() }()
		writeJSON(w, map[string]string{"status": "started"})
	case path == "/cleanup/run" && r.Method == http.MethodPost:
		if !auth.CanWriteSettings(role) {
			http.Error(w, "forbidden", 403)
			return
		}
		go func() { _ = s.scheduler.RunCleanup() }()
		writeJSON(w, map[string]string{"status": "started"})
	case path == "/tokens" && r.Method == http.MethodGet:
		if !auth.CanManageUsers(role) {
			http.Error(w, "forbidden", 403)
			return
		}
		s.listTokens(w)
	case path == "/tokens" && r.Method == http.MethodPost:
		if !auth.CanManageUsers(role) {
			http.Error(w, "forbidden", 403)
			return
		}
		s.createToken(w, r)
	case strings.HasPrefix(path, "/tokens/") && r.Method == http.MethodPut:
		if !auth.CanManageUsers(role) {
			http.Error(w, "forbidden", 403)
			return
		}
		id := strings.TrimPrefix(path, "/tokens/")
		s.updateToken(w, r, id)
	case strings.HasPrefix(path, "/tokens/") && r.Method == http.MethodDelete:
		if !auth.CanManageUsers(role) {
			http.Error(w, "forbidden", 403)
			return
		}
		id := strings.TrimPrefix(path, "/tokens/")
		s.deleteToken(w, id)
	case path == "/users" && r.Method == http.MethodGet:
		if !auth.CanManageUsers(role) {
			http.Error(w, "forbidden", 403)
			return
		}
		s.listUsers(w)
	case path == "/users" && r.Method == http.MethodPost:
		if !auth.CanManageUsers(role) {
			http.Error(w, "forbidden", 403)
			return
		}
		s.createUser(w, r)
	case strings.HasPrefix(path, "/users/") && r.Method == http.MethodPut:
		if !auth.CanManageUsers(role) {
			http.Error(w, "forbidden", 403)
			return
		}
		id := strings.TrimPrefix(path, "/users/")
		s.updateUser(w, r, id)
	case strings.HasPrefix(path, "/users/") && r.Method == http.MethodDelete:
		if !auth.CanManageUsers(role) {
			http.Error(w, "forbidden", 403)
			return
		}
		id := strings.TrimPrefix(path, "/users/")
		s.deleteUser(w, id)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) listDomains(w http.ResponseWriter) {
	domains, err := s.store.ListDomains(false)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, domains)
}

type createDomainsReq struct {
	Domains  string `json:"domains"`
	Wildcard bool   `json:"wildcard"`
}

func (s *Server) createDomains(w http.ResponseWriter, r *http.Request) {
	var req createDomainsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	lines := splitDomains(req.Domains)
	var created []store.Domain
	for _, d := range lines {
		domain := store.Domain{
			ID:        uuid.New().String(),
			Domain:    d,
			Wildcard:  req.Wildcard,
			Enabled:   true,
			CreatedAt: time.Now(),
		}
		if err := s.store.SaveDomain(domain); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		created = append(created, domain)
		_ = s.store.AddLog("info", "domain_add", "domain added", d)
	}
	writeJSON(w, created)
}

func splitDomains(raw string) []string {
	raw = strings.ReplaceAll(raw, ",", "\n")
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func (s *Server) deleteDomain(w http.ResponseWriter, id string) {
	if err := s.store.DeleteDomain(id); err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) renewDomain(w http.ResponseWriter, id string) {
	settings, err := s.store.GetSettings()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if settings.AcmeEmail == "" {
		http.Error(w, "请先在系统设置中配置 ACME 注册邮箱", http.StatusBadRequest)
		return
	}
	if settings.GoDaddyAPIKey == "" || settings.GoDaddyAPISecret == "" {
		http.Error(w, "请先在系统设置中配置 GoDaddy API Key 和 Secret", http.StatusBadRequest)
		return
	}

	d, err := s.store.GetDomain(id)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}

	go func() {
		if err := s.scheduler.RenewDomain(id); err != nil {
			_ = s.store.AddLog("error", "renew", err.Error(), d.Domain)
		}
	}()
	writeJSON(w, map[string]string{"status": "started", "domain": d.Domain})
}

type certView struct {
	store.Certificate
	DaysLeft int    `json:"days_left"`
	Status   string `json:"status"`
}

func (s *Server) listCertificates(w http.ResponseWriter) {
	certs, err := s.store.ListCertificates()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	var views = []certView{}
	for _, c := range certs {
		if !c.Active {
			continue
		}
		views = append(views, certView{
			Certificate: c,
			DaysLeft:    certsvc.DaysLeft(c.ExpiresAt),
			Status:      certsvc.CertStatus(c.ExpiresAt),
		})
	}
	writeJSON(w, views)
}

func (s *Server) syncAllCertsGit(w http.ResponseWriter) {
	settings, err := s.store.GetSettings()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if settings.GitRepoURL == "" {
		http.Error(w, "请先在系统设置中配置 Git 仓库地址", http.StatusBadRequest)
		return
	}

	n, err := s.scheduler.SyncAllCertsToGit()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, map[string]any{"status": "ok", "count": n})
}

func (s *Server) getSettings(w http.ResponseWriter) {
	settings, err := s.store.GetSettings()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	maskSecrets(&settings)
	writeJSON(w, settings)
}

func (s *Server) putSettings(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	var incoming store.Settings
	if err := json.Unmarshal(body, &incoming); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	current, _ := s.store.GetSettings()
	mergeSecrets(&incoming, current)
	if incoming.RenewBeforeDays == 0 {
		incoming.RenewBeforeDays = 30
	}
	if incoming.CheckCron == "" {
		incoming.CheckCron = "0 3 * * *"
	}
	if incoming.CleanupCron == "" {
		incoming.CleanupCron = "0 4 * * *"
	}
	if incoming.OIDCDefaultRole == "" {
		incoming.OIDCDefaultRole = store.RoleViewer
	}
	if err := s.store.SaveSettings(incoming); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) resetACME(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Staging *bool `json:"staging"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Staging == nil {
		_ = s.store.DeleteACMEAccount(true)
		_ = s.store.DeleteACMEAccount(false)
	} else {
		_ = s.store.DeleteACMEAccount(*req.Staging)
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func maskSecrets(s *store.Settings) {
	s.GoDaddyAPISecret = mask(s.GoDaddyAPISecret)
	s.GitToken = mask(s.GitToken)
	s.OIDCClientSecret = mask(s.OIDCClientSecret)
}

func mergeSecrets(incoming *store.Settings, current store.Settings) {
	if incoming.GoDaddyAPISecret == "" || strings.HasPrefix(incoming.GoDaddyAPISecret, "****") {
		incoming.GoDaddyAPISecret = current.GoDaddyAPISecret
	}
	if incoming.GitToken == "" || strings.HasPrefix(incoming.GitToken, "****") {
		incoming.GitToken = current.GitToken
	}
	if incoming.OIDCClientSecret == "" || strings.HasPrefix(incoming.OIDCClientSecret, "****") {
		incoming.OIDCClientSecret = current.OIDCClientSecret
	}
}

func (s *Server) listLogs(w http.ResponseWriter) {
	logs, err := s.store.ListLogs()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, logs)
}

func (s *Server) listUsers(w http.ResponseWriter) {
	users, err := s.store.ListUsers()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	var views = []userDTO{}
	for _, u := range users {
		views = append(views, toUserView(u))
	}
	writeJSON(w, views)
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if req.Role == "" {
		req.Role = store.RoleViewer
	}
	u, err := auth.NewLocalUser(req.Username, req.Email, req.Password, req.Role)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := s.store.SaveUser(u); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, toUserView(u))
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request, id string) {
	u, err := s.store.GetUser(id)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	var req struct {
		Email    string `json:"email"`
		Role     string `json:"role"`
		Enabled  *bool  `json:"enabled"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if req.Email != "" {
		u.Email = req.Email
	}
	if req.Role != "" {
		u.Role = req.Role
	}
	if req.Enabled != nil {
		u.Enabled = *req.Enabled
	}
	if req.Password != "" && u.AuthType == "local" {
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		u.PasswordHash = hash
	}
	if err := s.store.SaveUser(u); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, toUserView(u))
}

func (s *Server) deleteUser(w http.ResponseWriter, id string) {
	u, err := s.store.GetUser(id)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	if u.Role == store.RoleAdmin {
		n, _ := s.store.CountAdmins()
		if n <= 1 {
			http.Error(w, "cannot delete last admin", 400)
			return
		}
	}
	if err := s.store.DeleteUser(id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func mask(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "****"
	}
	return "****" + s[len(s)-4:]
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
