package certsvc

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kevin/kkcert/internal/acme"
	"github.com/kevin/kkcert/internal/gitsync"
	"github.com/kevin/kkcert/internal/store"
)

// ponytail: in-memory debounce map; upgrade to bbolt if multi-instance
var lastRenew = struct {
	sync.Mutex
	m map[string]time.Time
}{m: make(map[string]time.Time)}

var renewMu sync.Map // domainID -> *sync.Mutex

func lockDomainRenew(id string) func() {
	v, _ := renewMu.LoadOrStore(id, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

type Service struct {
	store   *store.Store
	git     *gitsync.Syncer
	dataDir string
}

func New(st *store.Store, dataDir string) *Service {
	return &Service{store: st, git: gitsync.New(dataDir), dataDir: dataDir}
}

func (s *Service) RenewDomain(domainID string) error {
	d, err := s.store.GetDomain(domainID)
	if err != nil {
		return err
	}
	return s.renew(d)
}

func (s *Service) RenewIfNeeded(d store.Domain, settings store.Settings) error {
	if !d.Enabled || d.Archived {
		return nil
	}
	cert, ok := s.store.GetActiveCert(d.ID)
	if !ok {
		slog.Info("no cert, issuing", "domain", d.Domain)
		return s.renew(d)
	}

	daysLeft := int(time.Until(cert.ExpiresAt).Hours() / 24)
	if daysLeft > settings.RenewBeforeDays {
		return nil
	}

	lastRenew.Lock()
	if t, ok := lastRenew.m[d.ID]; ok && time.Since(t) < 24*time.Hour {
		lastRenew.Unlock()
		slog.Info("skip renew debounce", "domain", d.Domain)
		return nil
	}
	lastRenew.m[d.ID] = time.Now()
	lastRenew.Unlock()

	slog.Info("auto renew", "domain", d.Domain, "days_left", daysLeft)
	return s.renew(d)
}

func (s *Service) renew(d store.Domain) error {
	unlock := lockDomainRenew(d.ID)
	defer unlock()

	settings, err := s.store.GetSettings()
	if err != nil {
		return err
	}

	_ = s.store.AddLog("info", "renew", "started", d.Domain)

	issuer := acme.NewIssuer(s.store, settings)
	result, err := issuer.Issue(d.Domain, d.Wildcard)
	if err != nil {
		_ = s.store.AddLog("error", "renew", err.Error(), d.Domain)
		return fmt.Errorf("issue %s: %w", d.Domain, err)
	}

	cert := store.Certificate{
		ID:        uuid.New().String(),
		DomainID:  d.ID,
		Domain:    d.Domain,
		CertPEM:   result.CertPEM,
		KeyPEM:    result.KeyPEM,
		ExpiresAt: result.ExpiresAt,
		IssuedAt:  time.Now(),
		Active:    true,
	}
	if err := s.store.SaveCertificate(cert); err != nil {
		return err
	}

	if err := s.git.Sync(settings, cert); err != nil {
		_ = s.store.AddLog("error", "git_sync", err.Error(), d.Domain)
		return fmt.Errorf("git sync %s: %w", d.Domain, err)
	}

	_ = s.store.AddLog("info", "renew", fmt.Sprintf("expires %s", result.ExpiresAt.Format("2006-01-02")), d.Domain)
	return nil
}

func (s *Service) SyncAllCertsToGit() (int, error) {
	settings, err := s.store.GetSettings()
	if err != nil {
		return 0, err
	}
	if settings.GitRepoURL == "" {
		return 0, fmt.Errorf("git repo url not configured")
	}

	certs, err := s.store.ListCertificates()
	if err != nil {
		return 0, err
	}
	active := make([]store.Certificate, 0, len(certs))
	for _, c := range certs {
		if c.Active {
			active = append(active, c)
		}
	}
	if len(active) == 0 {
		return 0, fmt.Errorf("no active certificates to sync")
	}

	if err := s.git.SyncAll(settings, active); err != nil {
		_ = s.store.AddLog("error", "git_sync", err.Error(), "")
		return 0, err
	}
	_ = s.store.AddLog("info", "git_sync", fmt.Sprintf("pushed %d certificates", len(active)), "")
	return len(active), nil
}

func CertStatus(expiresAt time.Time) string {
	days := int(time.Until(expiresAt).Hours() / 24)
	switch {
	case days < 0:
		return "expired"
	case days <= 30:
		return "warning"
	default:
		return "ok"
	}
}

func DaysLeft(expiresAt time.Time) int {
	return int(time.Until(expiresAt).Hours() / 24)
}
