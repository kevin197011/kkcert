package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kevin/kkcert/internal/store"
)

func CreateAPIToken(st *store.Store, name, role string, expiresDays int) (raw string, token store.APIToken, err error) {
	if name == "" {
		return "", store.APIToken{}, fmt.Errorf("name required")
	}
	if role == "" {
		role = store.RoleViewer
	}
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", store.APIToken{}, err
	}
	raw = "kkcert_" + hex.EncodeToString(b)
	now := time.Now()
	t := store.APIToken{
		ID:        uuid.New().String(),
		Name:      name,
		TokenHash: store.HashAPIToken(raw),
		Prefix:    raw[:16] + "...",
		Role:      role,
		Enabled:   true,
		CreatedAt: now,
	}
	if expiresDays > 0 {
		exp := now.Add(time.Duration(expiresDays) * 24 * time.Hour)
		t.ExpiresAt = &exp
	}
	return raw, t, st.SaveAPIToken(t)
}

func authenticateAPIToken(st *store.Store, token string) (*Principal, bool) {
	if token == "" {
		return nil, false
	}
	settings, _ := st.GetSettings()
	if settings.APIToken != "" && token == settings.APIToken {
		return &Principal{User: store.User{ID: "legacy-api", Username: "api-legacy", Role: store.RoleAdmin, Enabled: true}}, true
	}
	if !strings.HasPrefix(token, "kkcert_") {
		return nil, false
	}
	hash := store.HashAPIToken(token)
	t, ok := st.FindAPITokenByHash(hash)
	if !ok {
		return nil, false
	}
	st.TouchAPIToken(t.ID)
	return &Principal{User: store.User{
		ID:       "token:" + t.ID,
		Username: t.Name,
		Role:     t.Role,
		Enabled:  true,
		AuthType: "api_token",
	}}, true
}
