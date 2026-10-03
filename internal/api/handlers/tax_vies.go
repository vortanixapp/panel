package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/regions"
)

const viesEndpoint = "https://ec.europa.eu/taxation_customs/vies/rest-api/check-vat-number"

var viesClient = &http.Client{Timeout: 10 * time.Second}

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9]`)

func viesCountryCode(country string) string {
	if country == "GR" {
		return "EL"
	}
	return country
}

func normalizeVATNumber(country, raw string) string {
	number := strings.ToUpper(nonAlnum.ReplaceAllString(raw, ""))
	for _, prefix := range []string{viesCountryCode(country), country} {
		if prefix != "" && strings.HasPrefix(number, prefix) {
			return strings.TrimPrefix(number, prefix)
		}
	}
	return number
}

func verifyVIES(ctx context.Context, country, taxID string) (valid, checked bool) {
	if !regions.IsEU(country) {
		return false, false
	}
	number := normalizeVATNumber(country, taxID)
	if number == "" {
		return false, false
	}
	payload, _ := json.Marshal(map[string]string{
		"countryCode": viesCountryCode(country),
		"vatNumber":   number,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, viesEndpoint, bytes.NewReader(payload))
	if err != nil {
		return false, false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := viesClient.Do(req)
	if err != nil {
		return false, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, false
	}
	var out struct {
		Valid     bool   `json:"valid"`
		UserError string `json:"userError"`
	}
	if json.NewDecoder(resp.Body).Decode(&out) != nil {
		return false, false
	}
	if out.UserError != "" && out.UserError != "VALID" && out.UserError != "INVALID" {
		return false, false
	}
	return out.Valid, true
}
