package api

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kevin/kkcert/internal/auth"
	"github.com/kevin/kkcert/internal/certpack"
	"github.com/kevin/kkcert/internal/certsvc"
	"github.com/kevin/kkcert/internal/scheduler"
	"github.com/kevin/kkcert/internal/store"
)

type Server struct {
	store     *store.Store
	scheduler *scheduler.Scheduler
	dataDir   string
	oidc      *auth.OIDCHandler
	static    fs.FS
	handler   http.Handler
}

func NewServer(st *store.Store, sched *scheduler.Scheduler, dataDir string, static fs.FS) *Server {
	s := &Server{
		store:     st,
		scheduler: sched,
		dataDir:   dataDir,
		oidc:      &auth.OIDCHandler{Store: st},
		static:    static,
	}
	s.handler = s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
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

func (s *Server) listDomains(w http.ResponseWriter, _ *http.Request) {
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

func (s *Server) deleteDomain(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.store.DeleteDomain(id); err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) renewDomain(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
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

func (s *Server) listCertificates(w http.ResponseWriter, _ *http.Request) {
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

func (s *Server) syncAllCertsGit(w http.ResponseWriter, _ *http.Request) {
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

func (s *Server) downloadDomainCert(w http.ResponseWriter, r *http.Request) {
	domainID := chi.URLParam(r, "id")
	if _, err := s.store.GetDomain(domainID); err != nil {
		http.Error(w, "domain not found", 404)
		return
	}
	cert, ok := s.store.GetActiveCert(domainID)
	if !ok {
		http.Error(w, "该域名尚未签发证书，无法下载", http.StatusBadRequest)
		return
	}
	switch certsvc.CertStatus(cert.ExpiresAt) {
	case "expired":
		http.Error(w, "证书已过期，无法下载，请先续签", http.StatusBadRequest)
		return
	case "warning":
		http.Error(w, "证书即将过期，无法下载，请先续签", http.StatusBadRequest)
		return
	}

	zipPath, err := certpack.CreateDomainZip(certpack.DownloadsDir(s.dataDir), cert)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	filename := fmt.Sprintf("%s-cert.zip", cert.Domain)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	http.ServeFile(w, r, zipPath)
}

func (s *Server) getSettings(w http.ResponseWriter, _ *http.Request) {
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

func (s *Server) listLogs(w http.ResponseWriter, _ *http.Request) {
	logs, err := s.store.ListLogs()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, logs)
}

func (s *Server) listUsers(w http.ResponseWriter, _ *http.Request) {
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

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u, err := s.store.GetUser(id)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	if u.Username == "admin" {
		http.Error(w, "bootstrap admin account cannot be modified", http.StatusForbidden)
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
	if u.Role == store.RoleAdmin {
		if req.Role != "" && req.Role != store.RoleAdmin {
			n, _ := s.store.CountAdmins()
			if n <= 1 {
				http.Error(w, "cannot demote last admin", 400)
				return
			}
		}
		if req.Enabled != nil && !*req.Enabled {
			n, _ := s.store.CountAdmins()
			if n <= 1 {
				http.Error(w, "cannot disable last admin", 400)
				return
			}
		}
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

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u, err := s.store.GetUser(id)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	if u.Username == "admin" {
		http.Error(w, "cannot delete bootstrap admin account", http.StatusBadRequest)
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
