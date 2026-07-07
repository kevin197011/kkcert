package store

import (
	"encoding/json"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketDomains      = []byte("domains")
	bucketCertificates = []byte("certificates")
	bucketSettings     = []byte("settings")
	bucketLogs         = []byte("logs")
	keySettings        = []byte("global")
)

type Domain struct {
	ID        string    `json:"id"`
	Domain    string    `json:"domain"`
	Wildcard  bool      `json:"wildcard"`
	Enabled   bool      `json:"enabled"`
	Archived  bool      `json:"archived"`
	CreatedAt time.Time `json:"created_at"`
}

type Certificate struct {
	ID        string    `json:"id"`
	DomainID  string    `json:"domain_id"`
	Domain    string    `json:"domain"`
	CertPEM   string    `json:"cert_pem"`
	KeyPEM    string    `json:"key_pem"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedAt  time.Time `json:"issued_at"`
	Active    bool      `json:"active"`
}

type Settings struct {
	AcmeEmail        string `json:"acme_email"`
	AcmeStaging      bool   `json:"acme_staging"`
	GoDaddyAPIKey    string `json:"godaddy_api_key"`
	GoDaddyAPISecret string `json:"godaddy_api_secret"`
	GitRepoURL       string `json:"git_repo_url"`
	GitBranch        string `json:"git_branch"`
	GitAuthType      string `json:"git_auth_type"` // ssh | token
	GitSSHKeyPath    string `json:"git_ssh_key_path"`
	GitToken         string `json:"git_token"`
	GitCertsDir      string `json:"git_certs_dir"`
	RenewBeforeDays  int    `json:"renew_before_days"`
	AutoRenewEnabled bool   `json:"auto_renew_enabled"`
	CheckCron        string `json:"check_cron"`
	CleanupCron      string `json:"cleanup_cron"`
	OIDCEnabled      bool   `json:"oidc_enabled"`
	OIDCIssuer       string `json:"oidc_issuer"`
	OIDCClientID     string `json:"oidc_client_id"`
	OIDCClientSecret string `json:"oidc_client_secret"`
	OIDCRedirectURL  string `json:"oidc_redirect_url"`
	OIDCDefaultRole  string `json:"oidc_default_role"`
}

func DefaultSettings() Settings {
	return Settings{
		GitBranch:        "main",
		GitAuthType:      "ssh",
		GitCertsDir:      "certs",
		RenewBeforeDays:  30,
		AutoRenewEnabled: true,
		CheckCron:        "0 3 * * *",
		CleanupCron:      "0 4 * * *",
		OIDCDefaultRole:  RoleViewer,
	}
}

type OpLog struct {
	ID        string    `json:"id"`
	Level     string    `json:"level"` // info | error
	Action    string    `json:"action"`
	Message   string    `json:"message"`
	Domain    string    `json:"domain,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Store struct {
	db *bolt.DB
}

func Open(path string) (*Store, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.init(); err != nil {
		db.Close()
		return nil, err
	}
	if err := s.bootstrapAdmin(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) init() error {
	if err := s.db.Update(func(tx *bolt.Tx) error {
		for _, b := range [][]byte{bucketDomains, bucketCertificates, bucketSettings, bucketLogs} {
			if _, err := tx.CreateBucketIfNotExists(b); err != nil {
				return err
			}
		}
		sb := tx.Bucket(bucketSettings)
		if sb.Get(keySettings) == nil {
			return sb.Put(keySettings, mustJSON(DefaultSettings()))
		}
		return nil
	}); err != nil {
		return err
	}
	return s.initAuth()
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) GetSettings() (Settings, error) {
	var settings Settings
	err := s.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bucketSettings).Get(keySettings)
		return json.Unmarshal(v, &settings)
	})
	return settings, err
}

func (s *Store) SaveSettings(settings Settings) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketSettings).Put(keySettings, mustJSON(settings))
	})
}

func (s *Store) ListDomains(includeArchived bool) ([]Domain, error) {
	out := []Domain{}
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketDomains).ForEach(func(_, v []byte) error {
			var d Domain
			if err := json.Unmarshal(v, &d); err != nil {
				return err
			}
			if !includeArchived && d.Archived {
				return nil
			}
			out = append(out, d)
			return nil
		})
	})
	return out, err
}

func (s *Store) GetDomain(id string) (Domain, error) {
	var d Domain
	err := s.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bucketDomains).Get([]byte(id))
		if v == nil {
			return fmt.Errorf("domain not found")
		}
		return json.Unmarshal(v, &d)
	})
	return d, err
}

func (s *Store) SaveDomain(d Domain) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketDomains).Put([]byte(d.ID), mustJSON(d))
	})
}

func (s *Store) DeleteDomain(id string) error {
	d, err := s.GetDomain(id)
	if err != nil {
		return err
	}
	d.Archived = true
	d.Enabled = false
	if err := s.SaveDomain(d); err != nil {
		return err
	}
	_, _ = s.DeleteCertificatesForDomain(id, "")
	return nil
}

func (s *Store) ListCertificates() ([]Certificate, error) {
	out := []Certificate{}
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketCertificates).ForEach(func(_, v []byte) error {
			var c Certificate
			if err := json.Unmarshal(v, &c); err != nil {
				return err
			}
			out = append(out, c)
			return nil
		})
	})
	return out, err
}

func (s *Store) GetActiveCert(domainID string) (Certificate, bool) {
	certs, err := s.ListCertificates()
	if err != nil {
		return Certificate{}, false
	}
	for _, c := range certs {
		if c.DomainID == domainID && c.Active {
			return c, true
		}
	}
	return Certificate{}, false
}

func (s *Store) SaveCertificate(c Certificate) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketCertificates)
		if c.Active {
			var toDelete [][]byte
			_ = b.ForEach(func(k, v []byte) error {
				var old Certificate
				if json.Unmarshal(v, &old) != nil {
					return nil
				}
				if old.DomainID == c.DomainID && string(k) != c.ID {
					toDelete = append(toDelete, append([]byte{}, k...))
				}
				return nil
			})
			for _, k := range toDelete {
				_ = b.Delete(k)
			}
		}
		return b.Put([]byte(c.ID), mustJSON(c))
	})
}

func (s *Store) AddLog(level, action, message, domain string) error {
	entry := OpLog{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Level:     level,
		Action:    action,
		Message:   message,
		Domain:    domain,
		CreatedAt: time.Now(),
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketLogs)
		if err := b.Put([]byte(entry.ID), mustJSON(entry)); err != nil {
			return err
		}
		// ponytail: keep last 50 logs via naive scan
		var keys [][]byte
		_ = b.ForEach(func(k, _ []byte) error {
			keys = append(keys, append([]byte{}, k...))
			return nil
		})
		if len(keys) > 50 {
			for _, k := range keys[:len(keys)-50] {
				_ = b.Delete(k)
			}
		}
		return nil
	})
}

func (s *Store) ListLogs() ([]OpLog, error) {
	out := []OpLog{}
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketLogs).ForEach(func(_, v []byte) error {
			var l OpLog
			if err := json.Unmarshal(v, &l); err != nil {
				return err
			}
			out = append(out, l)
			return nil
		})
	})
	// newest first
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, err
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
