package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kevin/kkcert/internal/auth"
	"github.com/kevin/kkcert/internal/store"
)

type tokenDTO struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Prefix     string  `json:"prefix"`
	Role       string  `json:"role"`
	Enabled    bool    `json:"enabled"`
	CreatedAt  string  `json:"created_at"`
	ExpiresAt  *string `json:"expires_at,omitempty"`
	LastUsedAt *string `json:"last_used_at,omitempty"`
}

type createTokenReq struct {
	Name        string `json:"name"`
	Role        string `json:"role"`
	ExpiresDays int    `json:"expires_days"`
}

type createTokenResp struct {
	tokenDTO
	Token string `json:"token"`
}

func toTokenDTO(t store.APIToken) tokenDTO {
	d := tokenDTO{
		ID:        t.ID,
		Name:      t.Name,
		Prefix:    t.Prefix,
		Role:      t.Role,
		Enabled:   t.Enabled,
		CreatedAt: t.CreatedAt.Format(timeRFC3339),
	}
	if t.ExpiresAt != nil {
		s := t.ExpiresAt.Format(timeRFC3339)
		d.ExpiresAt = &s
	}
	if t.LastUsedAt != nil {
		s := t.LastUsedAt.Format(timeRFC3339)
		d.LastUsedAt = &s
	}
	return d
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

func (s *Server) listTokens(w http.ResponseWriter, _ *http.Request) {
	tokens, err := s.store.ListAPITokens()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	out := []tokenDTO{}
	for _, t := range tokens {
		out = append(out, toTokenDTO(t))
	}
	writeJSON(w, out)
}

func (s *Server) createToken(w http.ResponseWriter, r *http.Request) {
	var req createTokenReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	raw, t, err := auth.CreateAPIToken(s.store, req.Name, req.Role, req.ExpiresDays)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	_ = s.store.AddLog("info", "token_create", "api token created: "+t.Name, "")
	writeJSON(w, createTokenResp{tokenDTO: toTokenDTO(t), Token: raw})
}

func (s *Server) updateToken(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	t, err := s.store.GetAPIToken(id)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	var req struct {
		Enabled *bool  `json:"enabled"`
		Role    string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if req.Enabled != nil {
		t.Enabled = *req.Enabled
	}
	if req.Role != "" {
		t.Role = req.Role
	}
	if err := s.store.SaveAPIToken(t); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, toTokenDTO(t))
}

func (s *Server) deleteToken(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.store.DeleteAPIToken(id); err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	_ = s.store.AddLog("info", "token_delete", "api token revoked", id)
	w.WriteHeader(http.StatusNoContent)
}
