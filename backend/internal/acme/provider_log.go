package acme

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"
)

type logDNSProvider struct {
	inner   challenge.Provider
	timeout challenge.ProviderTimeout
	zone    string
	key     string
	secret  string
}

func wrapDNSProvider(inner challenge.Provider, zone, key, secret string) challenge.Provider {
	p := &logDNSProvider{inner: inner, zone: zone, key: key, secret: secret}
	if t, ok := inner.(challenge.ProviderTimeout); ok {
		p.timeout = t
	}
	return p
}

func (p *logDNSProvider) Present(domain, token, keyAuth string) error {
	// ponytail: lego godaddy merges TXT; wipe first so stale/extra records cannot block propagation
	if err := deleteChallengeTXT(p.key, p.secret, p.zone); err != nil {
		return fmt.Errorf("pre-present cleanup: %w", err)
	}
	err := p.inner.Present(domain, token, keyAuth)
	txts, apiErr := listChallengeTXT(p.key, p.secret, p.zone)
	slog.Info("acme dns present",
		"domain", domain,
		"zone", p.zone,
		"err", err,
		"godaddy_txt", txts,
		"godaddy_api_err", apiErr,
	)
	return err
}

func (p *logDNSProvider) CleanUp(domain, token, keyAuth string) error {
	return p.inner.CleanUp(domain, token, keyAuth)
}

func (p *logDNSProvider) Timeout() (timeout, interval time.Duration) {
	if p.timeout != nil {
		return p.timeout.Timeout()
	}
	return 0, 0
}

// ponytail: lego PropagationWait(s,false) sleeps on every poll; we only need once before first check.
func oncePropagationWait(d time.Duration) dns01.ChallengeOption {
	var once sync.Once
	return dns01.WrapPreCheck(func(domain, fqdn, value string, check dns01.PreCheckFunc) (bool, error) {
		once.Do(func() { time.Sleep(d) })
		ok, err := check(fqdn, value)
		if !ok && err != nil {
			slog.Info("acme dns propagation pending", "domain", domain, "fqdn", fqdn, "err", err)
		}
		return ok, err
	})
}
