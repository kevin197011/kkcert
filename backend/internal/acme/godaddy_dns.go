package acme

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type godaddyTXT struct {
	Data string `json:"data"`
	Name string `json:"name"`
}

func listChallengeTXT(apiKey, apiSecret, zone string) ([]string, error) {
	u := fmt.Sprintf("https://api.godaddy.com/v1/domains/%s/records/TXT/_acme-challenge", zone)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("sso-key %s:%s", apiKey, apiSecret))

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("godaddy list txt: status %d: %s", resp.StatusCode, body)
	}

	var records []godaddyTXT
	if err := json.Unmarshal(body, &records); err != nil {
		return nil, err
	}

	out := make([]string, 0, len(records))
	for _, r := range records {
		if r.Data != "" {
			out = append(out, r.Data)
		}
	}
	return out, nil
}

func logChallengeDiagnostics(apiKey, apiSecret, zone string) {
	fqdn := "_acme-challenge." + zone
	txts, err := listChallengeTXT(apiKey, apiSecret, zone)
	slog.Warn("acme dns diagnostics",
		"zone", zone,
		"fqdn", fqdn,
		"godaddy_api_txt", txts,
		"godaddy_api_err", err,
	)
}
