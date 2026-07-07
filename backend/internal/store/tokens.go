package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

var bucketAPITokens = []byte("api_tokens")

// APIToken is a machine/agent access token (secret stored as SHA-256 hash).
type APIToken struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	TokenHash  string     `json:"-"`
	Prefix     string     `json:"prefix"`
	Role       string     `json:"role"`
	Enabled    bool       `json:"enabled"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

func HashAPIToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (s *Store) SaveAPIToken(t APIToken) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketAPITokens).Put([]byte(t.ID), mustJSON(t))
	})
}

func (s *Store) ListAPITokens() ([]APIToken, error) {
	out := []APIToken{}
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketAPITokens).ForEach(func(_, v []byte) error {
			var t APIToken
			if err := json.Unmarshal(v, &t); err != nil {
				return err
			}
			out = append(out, t)
			return nil
		})
	})
	return out, err
}

func (s *Store) GetAPIToken(id string) (APIToken, error) {
	var t APIToken
	err := s.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bucketAPITokens).Get([]byte(id))
		if v == nil {
			return fmt.Errorf("token not found")
		}
		return json.Unmarshal(v, &t)
	})
	return t, err
}

func (s *Store) FindAPITokenByHash(hash string) (APIToken, bool) {
	tokens, err := s.ListAPITokens()
	if err != nil {
		return APIToken{}, false
	}
	for _, t := range tokens {
		if t.TokenHash == hash && t.Enabled {
			if t.ExpiresAt != nil && time.Now().After(*t.ExpiresAt) {
				continue
			}
			return t, true
		}
	}
	return APIToken{}, false
}

func (s *Store) TouchAPIToken(id string) {
	t, err := s.GetAPIToken(id)
	if err != nil {
		return
	}
	now := time.Now()
	t.LastUsedAt = &now
	_ = s.SaveAPIToken(t)
}

func (s *Store) DeleteAPIToken(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketAPITokens).Delete([]byte(id))
	})
}

func (s *Store) PurgeExpiredAPITokens() (int, error) {
	var n int
	return n, s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketAPITokens)
		var keys [][]byte
		now := time.Now()
		_ = b.ForEach(func(k, v []byte) error {
			var t APIToken
			if json.Unmarshal(v, &t) != nil {
				return nil
			}
			if t.ExpiresAt != nil && now.After(*t.ExpiresAt) {
				keys = append(keys, append([]byte{}, k...))
			}
			return nil
		})
		for _, k := range keys {
			_ = b.Delete(k)
			n++
		}
		return nil
	})
}
