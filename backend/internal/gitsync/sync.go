package gitsync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/kevin/kkcert/internal/store"
	gossh "golang.org/x/crypto/ssh"
)

type Syncer struct {
	dataDir string
}

func New(dataDir string) *Syncer {
	return &Syncer{dataDir: dataDir}
}

type metadata struct {
	Domain    string    `json:"domain"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedAt  time.Time `json:"issued_at"`
}

func (s *Syncer) Sync(settings store.Settings, cert store.Certificate, action ...string) error {
	if settings.GitRepoURL == "" {
		return fmt.Errorf("git repo url not configured")
	}

	workDir, branch, certsDir, err := s.paths(settings)
	if err != nil {
		return err
	}

	repo, err := s.openOrClone(settings, workDir, branch)
	if err != nil {
		return err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}

	if err := s.writeCert(workDir, certsDir, wt, cert); err != nil {
		return err
	}

	msg := fmt.Sprintf("kkcert: renew %s", cert.Domain)
	if len(action) > 0 && action[0] != "" {
		msg = fmt.Sprintf("kkcert: %s %s", action[0], cert.Domain)
	}
	return s.commitAndPush(settings, repo, wt, msg)
}

// SyncAll writes every active certificate and pushes in a single commit.
func (s *Syncer) SyncAll(settings store.Settings, certs []store.Certificate) error {
	if settings.GitRepoURL == "" {
		return fmt.Errorf("git repo url not configured")
	}
	if len(certs) == 0 {
		return fmt.Errorf("no active certificates to sync")
	}

	workDir, branch, certsDir, err := s.paths(settings)
	if err != nil {
		return err
	}

	repo, err := s.openOrClone(settings, workDir, branch)
	if err != nil {
		return err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}

	for _, cert := range certs {
		if err := s.writeCert(workDir, certsDir, wt, cert); err != nil {
			return fmt.Errorf("%s: %w", cert.Domain, err)
		}
	}

	msg := fmt.Sprintf("kkcert: sync all (%d domains)", len(certs))
	return s.commitAndPush(settings, repo, wt, msg)
}

func (s *Syncer) paths(settings store.Settings) (workDir, branch, certsDir string, err error) {
	workDir = filepath.Join(s.dataDir, "git-workspace")
	branch = settings.GitBranch
	if branch == "" {
		branch = "main"
	}
	certsDir = settings.GitCertsDir
	if certsDir == "" {
		certsDir = "certs"
	}
	return workDir, branch, certsDir, nil
}

func (s *Syncer) writeCert(workDir, certsDir string, wt *git.Worktree, cert store.Certificate) error {
	domainDir := filepath.Join(workDir, certsDir, cert.Domain)
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(domainDir, "fullchain.pem"), []byte(cert.CertPEM), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(domainDir, "privkey.pem"), []byte(cert.KeyPEM), 0600); err != nil {
		return err
	}
	meta, _ := json.MarshalIndent(metadata{
		Domain:    cert.Domain,
		ExpiresAt: cert.ExpiresAt,
		IssuedAt:  cert.IssuedAt,
	}, "", "  ")
	if err := os.WriteFile(filepath.Join(domainDir, "metadata.json"), meta, 0644); err != nil {
		return err
	}

	for _, name := range []string{"fullchain.pem", "privkey.pem", "metadata.json"} {
		if _, err := wt.Add(filepath.Join(certsDir, cert.Domain, name)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Syncer) commitAndPush(settings store.Settings, repo *git.Repository, wt *git.Worktree, msg string) error {
	_, err := wt.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{Name: "kkcert", Email: "kkcert@local", When: time.Now()},
		All:    true,
	})
	if err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	return repo.Push(&git.PushOptions{RemoteName: "origin", Auth: s.auth(settings)})
}

func (s *Syncer) openOrClone(settings store.Settings, workDir, branch string) (*git.Repository, error) {
	if _, err := os.Stat(filepath.Join(workDir, ".git")); err == nil {
		repo, err := git.PlainOpen(workDir)
		if err != nil {
			return nil, err
		}
		wt, err := repo.Worktree()
		if err != nil {
			return nil, err
		}
		_ = wt.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(branch)})
		_ = wt.Pull(&git.PullOptions{RemoteName: "origin", Auth: s.auth(settings)})
		return repo, nil
	}

	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, err
	}
	return git.PlainClone(workDir, false, &git.CloneOptions{
		URL:           settings.GitRepoURL,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		SingleBranch:  true,
		Auth:          s.auth(settings),
	})
}

func (s *Syncer) auth(settings store.Settings) transport.AuthMethod {
	if settings.GitAuthType == "token" {
		return &http.BasicAuth{Username: "token", Password: settings.GitToken}
	}
	if settings.GitSSHKeyPath == "" {
		return nil
	}
	pub, err := ssh.NewPublicKeysFromFile("git", settings.GitSSHKeyPath, "")
	if err != nil {
		return nil
	}
	pub.HostKeyCallback = gossh.InsecureIgnoreHostKey() // ponytail: use known_hosts in production
	return pub
}
