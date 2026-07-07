package scheduler

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kevin/kkcert/internal/certsvc"
	"github.com/kevin/kkcert/internal/store"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	store   *store.Store
	certs   *certsvc.Service
	dataDir string
	cron    *cron.Cron
	mu      sync.Mutex
	running bool
}

func New(st *store.Store, dataDir string) *Scheduler {
	return &Scheduler{
		store:   st,
		certs:   certsvc.New(st, dataDir),
		dataDir: dataDir,
		cron:    cron.New(),
	}
}

func (s *Scheduler) Start() error {
	settings, err := s.store.GetSettings()
	if err != nil {
		return err
	}
	cronExpr := settings.CheckCron
	if cronExpr == "" {
		cronExpr = "0 3 * * *"
	}
	_, err = s.cron.AddFunc(cronExpr, func() {
		slog.Info("scheduled cert check started")
		if err := s.RunCheck(); err != nil {
			slog.Error("scheduled check failed", "err", err)
		}
	})
	if err != nil {
		return err
	}

	cleanupCron := settings.CleanupCron
	if cleanupCron == "" {
		cleanupCron = "0 4 * * *"
	}
	_, err = s.cron.AddFunc(cleanupCron, func() {
		slog.Info("scheduled data cleanup started")
		if err := s.RunCleanup(); err != nil {
			slog.Error("scheduled cleanup failed", "err", err)
		}
	})
	if err != nil {
		return err
	}

	s.cron.Start()
	return nil
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}

func (s *Scheduler) RunCheck() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	settings, err := s.store.GetSettings()
	if err != nil {
		return err
	}

	domains, err := s.store.ListDomains(false)
	if err != nil {
		return err
	}

	for _, d := range domains {
		if !d.Enabled {
			continue
		}
		if settings.AutoRenewEnabled {
			if err := s.certs.RenewIfNeeded(d, settings); err != nil {
				slog.Error("renew failed", "domain", d.Domain, "err", err)
			}
		}
		time.Sleep(10 * time.Second) // ponytail: godaddy rate limit spacing
	}
	_ = s.store.AddLog("info", "check", "daily check completed", "")
	return nil
}

func (s *Scheduler) RenewDomain(id string) error {
	return s.certs.RenewDomain(id)
}

func (s *Scheduler) SyncAllCertsToGit() (int, error) {
	return s.certs.SyncAllCertsToGit()
}

func (s *Scheduler) RunCleanup() error {
	certs, sessions, err := s.store.CleanupStaleData()
	if err != nil {
		return err
	}
	apiTokens, _ := s.store.PurgeExpiredAPITokens()
	if certs > 0 || sessions > 0 || apiTokens > 0 {
		msg := fmt.Sprintf("removed %d certs, %d sessions, %d api tokens", certs, sessions, apiTokens)
		slog.Info("data cleanup", "certs", certs, "sessions", sessions, "api_tokens", apiTokens)
		_ = s.store.AddLog("info", "cleanup", msg, "")
	}
	return nil
}
