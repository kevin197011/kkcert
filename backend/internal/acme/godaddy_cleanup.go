package acme

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// ponytail: minimal GoDaddy client for pre/post ACME challenge cleanup only.
func deleteChallengeTXT(apiKey, apiSecret, zone string) error {
	url := fmt.Sprintf("https://api.godaddy.com/v1/domains/%s/records/TXT/_acme-challenge", zone)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("sso-key %s:%s", apiKey, apiSecret))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("godaddy delete txt: status %d: %s", resp.StatusCode, body)
}
