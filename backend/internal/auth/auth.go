package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kevin/kkcert/internal/store"
	"golang.org/x/crypto/bcrypt"
)

const sessionTTL = 24 * time.Hour

type Principal struct {
	User store.User
}

func LoginLocal(st *store.Store, username, password string) (string, error) {
	u, ok := st.GetUserByUsername(username)
	if !ok || !u.Enabled || u.AuthType != "local" {
		return "", fmt.Errorf("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", fmt.Errorf("invalid credentials")
	}
	return createSession(st, u.ID)
}

func CreateSessionForUser(st *store.Store, userID string) (string, error) {
	return createSession(st, userID)
}

func createSession(st *store.Store, userID string) (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	sess := store.Session{
		ID:        token,
		UserID:    userID,
		ExpiresAt: time.Now().Add(sessionTTL),
		CreatedAt: time.Now(),
	}
	if err := st.SaveSession(sess); err != nil {
		return "", err
	}
	return token, nil
}

func Logout(st *store.Store, token string) {
	_ = st.DeleteSession(token)
}

func Authenticate(st *store.Store, r *http.Request) (*Principal, error) {
	token := bearerToken(r)
	if p, ok := authenticateAPIToken(st, token); ok {
		return p, nil
	}

	if token == "" {
		return nil, fmt.Errorf("unauthorized")
	}

	sess, err := st.GetSession(token)
	if err != nil {
		return nil, fmt.Errorf("unauthorized")
	}
	if time.Now().After(sess.ExpiresAt) {
		_ = st.DeleteSession(token)
		return nil, fmt.Errorf("session expired")
	}
	u, err := st.GetUser(sess.UserID)
	if err != nil || !u.Enabled {
		return nil, fmt.Errorf("unauthorized")
	}
	return &Principal{User: u}, nil
}

func BearerToken(r *http.Request) string {
	return bearerToken(r)
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	return strings.TrimPrefix(h, "Bearer ")
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CanRead returns true if role can read the resource.
func CanRead(role string) bool {
	return role == store.RoleAdmin || role == store.RoleOperator || role == store.RoleViewer
}

func CanWriteDomain(role string) bool {
	return role == store.RoleAdmin || role == store.RoleOperator
}

func CanWriteSettings(role string) bool {
	return role == store.RoleAdmin
}

func CanManageUsers(role string) bool {
	return role == store.RoleAdmin
}

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

func NewLocalUser(username, email, password, role string) (store.User, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return store.User{}, err
	}
	return store.User{
		ID:           uuid.New().String(),
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		Role:         role,
		AuthType:     "local",
		Enabled:      true,
		CreatedAt:    time.Now(),
	}, nil
}
