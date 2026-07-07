package store

import (
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func (s *Store) bootstrapAdmin() error {
	n, err := s.CountUsers()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	pw := os.Getenv("KKCERT_BOOTSTRAP_PASSWORD")
	if pw == "" {
		pw = "changeme"
		slog.Warn("bootstrap admin created with default password; set KKCERT_BOOTSTRAP_PASSWORD")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.SaveUser(User{
		ID:           uuid.New().String(),
		Username:     "admin",
		Email:        "admin@local",
		PasswordHash: string(hash),
		Role:         RoleAdmin,
		AuthType:     "local",
		Enabled:      true,
		CreatedAt:    time.Now(),
	})
}
