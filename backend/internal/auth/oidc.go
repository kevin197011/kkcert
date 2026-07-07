package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/kevin/kkcert/internal/store"
	"golang.org/x/oauth2"
)

// ponytail: in-memory oauth state; single-instance only
var oauthStates = struct {
	sync.Mutex
	m map[string]time.Time
}{m: make(map[string]time.Time)}

type OIDCHandler struct {
	Store *store.Store
}

func (h *OIDCHandler) provider(ctx context.Context, settings store.Settings) (*oidc.Provider, *oauth2.Config, error) {
	if !settings.OIDCEnabled || settings.OIDCIssuer == "" || settings.OIDCClientID == "" {
		return nil, nil, fmt.Errorf("oidc not configured")
	}
	provider, err := oidc.NewProvider(ctx, settings.OIDCIssuer)
	if err != nil {
		return nil, nil, err
	}
	redirect := settings.OIDCRedirectURL
	if redirect == "" {
		return nil, nil, fmt.Errorf("oidc redirect url required")
	}
	cfg := &oauth2.Config{
		ClientID:     settings.OIDCClientID,
		ClientSecret: settings.OIDCClientSecret,
		RedirectURL:  redirect,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	return provider, cfg, nil
}

func (h *OIDCHandler) LoginURL(w http.ResponseWriter, r *http.Request) (string, error) {
	settings, err := h.Store.GetSettings()
	if err != nil {
		return "", err
	}
	_, cfg, err := h.provider(r.Context(), settings)
	if err != nil {
		return "", err
	}
	state, err := newState()
	if err != nil {
		return "", err
	}
	return cfg.AuthCodeURL(state), nil
}

func (h *OIDCHandler) Callback(w http.ResponseWriter, r *http.Request) (string, error) {
	settings, err := h.Store.GetSettings()
	if err != nil {
		return "", err
	}
	if !consumeState(r.URL.Query().Get("state")) {
		return "", fmt.Errorf("invalid oauth state")
	}
	provider, cfg, err := h.provider(r.Context(), settings)
	if err != nil {
		return "", err
	}
	oauth2Token, err := cfg.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		return "", fmt.Errorf("token exchange: %w", err)
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return "", fmt.Errorf("no id_token")
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: settings.OIDCClientID})
	idToken, err := verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		return "", fmt.Errorf("verify id_token: %w", err)
	}
	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return "", err
	}

	u, ok := h.Store.GetUserByOIDCSub(claims.Sub)
	if !ok {
		role := settings.OIDCDefaultRole
		if role == "" {
			role = store.RoleViewer
		}
		username := claims.Email
		if username == "" {
			username = claims.Name
		}
		if username == "" {
			username = "oidc-" + claims.Sub[:8]
		}
		u = store.User{
			ID:        uuid.New().String(),
			Username:  username,
			Email:     claims.Email,
			Role:      role,
			AuthType:  "oidc",
			OIDCSub:   claims.Sub,
			Enabled:   true,
			CreatedAt: time.Now(),
		}
		if err := h.Store.SaveUser(u); err != nil {
			return "", err
		}
	}
	if !u.Enabled {
		return "", fmt.Errorf("user disabled")
	}
	return CreateSessionForUser(h.Store, u.ID)
}

func newState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	state := base64.RawURLEncoding.EncodeToString(b)
	oauthStates.Lock()
	oauthStates.m[state] = time.Now().Add(10 * time.Minute)
	oauthStates.Unlock()
	return state, nil
}

func consumeState(state string) bool {
	oauthStates.Lock()
	defer oauthStates.Unlock()
	exp, ok := oauthStates.m[state]
	if !ok || time.Now().After(exp) {
		return false
	}
	delete(oauthStates.m, state)
	return true
}
