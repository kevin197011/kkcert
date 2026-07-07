package store

import (
	"encoding/json"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleViewer   = "viewer"
)

var (
	bucketUsers    = []byte("users")
	bucketSessions = []byte("sessions")
	bucketACME     = []byte("acme_accounts")
)

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash,omitempty"`
	Role         string    `json:"role"`
	AuthType     string    `json:"auth_type"` // local | oidc
	OIDCSub      string    `json:"oidc_sub,omitempty"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
}

type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type ACMEAccount struct {
	Email            string    `json:"email"`
	Staging          bool      `json:"staging"`
	AccountKeyPEM    string    `json:"account_key_pem"`
	RegistrationJSON []byte    `json:"registration_json"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func acmeKey(staging bool) []byte {
	if staging {
		return []byte("staging")
	}
	return []byte("production")
}

func (s *Store) initAuth() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		for _, b := range [][]byte{bucketUsers, bucketSessions, bucketACME, bucketAPITokens} {
			if _, err := tx.CreateBucketIfNotExists(b); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) CountUsers() (int, error) {
	n := 0
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketUsers).ForEach(func(_, _ []byte) error {
			n++
			return nil
		})
	})
	return n, err
}

func (s *Store) SaveUser(u User) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketUsers).Put([]byte(u.ID), mustJSON(u))
	})
}

func (s *Store) GetUser(id string) (User, error) {
	var u User
	err := s.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bucketUsers).Get([]byte(id))
		if v == nil {
			return fmt.Errorf("user not found")
		}
		return json.Unmarshal(v, &u)
	})
	return u, err
}

func (s *Store) GetUserByUsername(username string) (User, bool) {
	users, _ := s.ListUsers()
	for _, u := range users {
		if u.Username == username {
			return u, true
		}
	}
	return User{}, false
}

func (s *Store) GetUserByOIDCSub(sub string) (User, bool) {
	users, _ := s.ListUsers()
	for _, u := range users {
		if u.OIDCSub == sub {
			return u, true
		}
	}
	return User{}, false
}

func (s *Store) ListUsers() ([]User, error) {
	out := []User{}
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketUsers).ForEach(func(_, v []byte) error {
			var u User
			if err := json.Unmarshal(v, &u); err != nil {
				return err
			}
			out = append(out, u)
			return nil
		})
	})
	return out, err
}

func (s *Store) DeleteUser(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketUsers).Delete([]byte(id))
	})
}

func (s *Store) CountAdmins() (int, error) {
	users, err := s.ListUsers()
	if err != nil {
		return 0, err
	}
	n := 0
	for _, u := range users {
		if u.Role == RoleAdmin && u.Enabled {
			n++
		}
	}
	return n, nil
}

func (s *Store) SaveSession(sess Session) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketSessions).Put([]byte(sess.ID), mustJSON(sess))
	})
}

func (s *Store) GetSession(id string) (Session, error) {
	var sess Session
	err := s.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bucketSessions).Get([]byte(id))
		if v == nil {
			return fmt.Errorf("session not found")
		}
		return json.Unmarshal(v, &sess)
	})
	return sess, err
}

func (s *Store) DeleteSession(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketSessions).Delete([]byte(id))
	})
}

func (s *Store) GetACMEAccount(staging bool) (ACMEAccount, bool) {
	var acc ACMEAccount
	err := s.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bucketACME).Get(acmeKey(staging))
		if v == nil {
			return fmt.Errorf("not found")
		}
		return json.Unmarshal(v, &acc)
	})
	return acc, err == nil
}

func (s *Store) SaveACMEAccount(acc ACMEAccount) error {
	acc.UpdatedAt = time.Now()
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketACME).Put(acmeKey(acc.Staging), mustJSON(acc))
	})
}

func (s *Store) DeleteACMEAccount(staging bool) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketACME).Delete(acmeKey(staging))
	})
}
