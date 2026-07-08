package certpack

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/kevin/kkcert/internal/store"
)

type metadata struct {
	Domain    string    `json:"domain"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedAt  time.Time `json:"issued_at"`
}

// WriteDomainZip packs fullchain.pem, privkey.pem, metadata.json for one domain.
func WriteDomainZip(destPath string, cert store.Certificate) error {
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	prefix := cert.Domain + "/"
	files := []struct {
		name string
		body []byte
		mode os.FileMode
	}{
		{"fullchain.pem", []byte(cert.CertPEM), 0644},
		{"privkey.pem", []byte(cert.KeyPEM), 0600},
	}
	for _, file := range files {
		w, err := zw.CreateHeader(&zip.FileHeader{
			Name:   prefix + file.name,
			Method: zip.Deflate,
		})
		if err != nil {
			return err
		}
		if _, err := w.Write(file.body); err != nil {
			return err
		}
	}

	meta, _ := json.MarshalIndent(metadata{
		Domain:    cert.Domain,
		ExpiresAt: cert.ExpiresAt,
		IssuedAt:  cert.IssuedAt,
	}, "", "  ")
	w, err := zw.CreateHeader(&zip.FileHeader{
		Name:   prefix + "metadata.json",
		Method: zip.Deflate,
	})
	if err != nil {
		return err
	}
	if _, err := w.Write(meta); err != nil {
		return err
	}

	return zw.Close()
}

// CreateDomainZip writes a temp zip under dir and returns its path.
func CreateDomainZip(dir string, cert store.Certificate) (string, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, uuid.New().String()+".zip")
	if err := WriteDomainZip(path, cert); err != nil {
		return "", err
	}
	return path, nil
}

// CleanupDir removes all .zip files in dir. ponytail: midnight cron only.
func CleanupDir(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".zip" {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			return n, fmt.Errorf("remove %s: %w", e.Name(), err)
		}
		n++
	}
	return n, nil
}

func DownloadsDir(dataDir string) string {
	return filepath.Join(dataDir, "downloads")
}
