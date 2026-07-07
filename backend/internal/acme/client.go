package acme

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns/godaddy"
	"github.com/go-acme/lego/v4/registration"
	"github.com/kevin/kkcert/internal/store"
)

type Issuer struct {
	st       *store.Store
	email    string
	staging  bool
	gdKey    string
	gdSecret string
}

func NewIssuer(st *store.Store, settings store.Settings) *Issuer {
	return &Issuer{
		st:       st,
		email:    settings.AcmeEmail,
		staging:  settings.AcmeStaging,
		gdKey:    settings.GoDaddyAPIKey,
		gdSecret: settings.GoDaddyAPISecret,
	}
}

type user struct {
	email        string
	registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *user) GetEmail() string                        { return u.email }
func (u *user) GetRegistration() *registration.Resource { return u.registration }
func (u *user) GetPrivateKey() crypto.PrivateKey        { return u.key }

type Result struct {
	CertPEM   string
	KeyPEM    string
	ExpiresAt time.Time
}

func (i *Issuer) Issue(domain string, wildcard bool) (*Result, error) {
	if i.email == "" || i.gdKey == "" || i.gdSecret == "" {
		return nil, fmt.Errorf("acme email and godaddy credentials required")
	}

	privateKey, reg, err := i.loadOrCreateAccount()
	if err != nil {
		return nil, err
	}

	u := &user{email: i.email, key: privateKey, registration: reg}
	cfg := lego.NewConfig(u)
	cfg.CADirURL = lego.LEDirectoryProduction
	if i.staging {
		cfg.CADirURL = lego.LEDirectoryStaging
	}
	cfg.Certificate.KeyType = certcrypto.EC256

	client, err := lego.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	gdCfg := godaddy.NewDefaultConfig()
	gdCfg.APIKey = i.gdKey
	gdCfg.APISecret = i.gdSecret
	gdCfg.PropagationTimeout = 10 * time.Minute
	gdCfg.PollingInterval = 3 * time.Second
	gdProvider, err := godaddy.NewDNSProviderConfig(gdCfg)
	if err != nil {
		return nil, err
	}
	provider := wrapDNSProvider(gdProvider, domain, i.gdKey, i.gdSecret)

	propagationNS := []string{"8.8.8.8:53", "1.1.1.1:53"}
	if err := client.Challenge.SetDNS01Provider(provider,
		oncePropagationWait(45*time.Second),
		dns01.AddRecursiveNameservers(propagationNS),
		dns01.AddDNSTimeout(10*time.Second),
	); err != nil {
		return nil, err
	}

	if u.registration == nil {
		regRes, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			return nil, fmt.Errorf("acme register: %w", err)
		}
		u.registration = regRes
		if err := i.persistAccount(privateKey, regRes); err != nil {
			return nil, err
		}
	}

	domains := []string{domain}
	if wildcard {
		domains = append(domains, "*."+domain)
	}

	// ponytail: wipe stale _acme-challenge TXT before obtain; failed runs often leave extras
	if err := deleteChallengeTXT(i.gdKey, i.gdSecret, domain); err != nil {
		return nil, fmt.Errorf("cleanup acme txt: %w", err)
	}

	cert, err := client.Certificate.Obtain(certificate.ObtainRequest{Domains: domains, Bundle: true})
	if err != nil {
		logChallengeDiagnostics(i.gdKey, i.gdSecret, domain)
		_ = deleteChallengeTXT(i.gdKey, i.gdSecret, domain)
		return nil, fmt.Errorf("obtain certificate: %w", err)
	}

	expiresAt, err := parseCertExpiry(cert.Certificate)
	if err != nil {
		expiresAt = time.Now().Add(90 * 24 * time.Hour)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: cert.PrivateKey})
	return &Result{
		CertPEM:   string(cert.Certificate),
		KeyPEM:    string(keyPEM),
		ExpiresAt: expiresAt,
	}, nil
}

func (i *Issuer) loadOrCreateAccount() (crypto.PrivateKey, *registration.Resource, error) {
	if acc, ok := i.st.GetACMEAccount(i.staging); ok && acc.AccountKeyPEM != "" {
		key, err := parseAccountKey(acc.AccountKeyPEM)
		if err != nil {
			return nil, nil, err
		}
		var reg registration.Resource
		if len(acc.RegistrationJSON) > 0 {
			_ = json.Unmarshal(acc.RegistrationJSON, &reg)
			return key, &reg, nil
		}
		return key, nil, nil
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return key, nil, nil
}

func (i *Issuer) persistAccount(key crypto.PrivateKey, reg *registration.Resource) error {
	pemBytes, err := marshalAccountKey(key)
	if err != nil {
		return err
	}
	regJSON, err := json.Marshal(reg)
	if err != nil {
		return err
	}
	return i.st.SaveACMEAccount(store.ACMEAccount{
		Email:            i.email,
		Staging:          i.staging,
		AccountKeyPEM:    string(pemBytes),
		RegistrationJSON: regJSON,
	})
}

func marshalAccountKey(key crypto.PrivateKey) ([]byte, error) {
	ec, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("unsupported key type")
	}
	b, err := x509.MarshalECPrivateKey(ec)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b}), nil
}

func parseAccountKey(pemStr string) (crypto.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("invalid account key pem")
	}
	ec, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return ec, nil
}

func parseCertExpiry(certPEM []byte) (time.Time, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return time.Time{}, fmt.Errorf("invalid pem")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, err
	}
	return cert.NotAfter, nil
}
