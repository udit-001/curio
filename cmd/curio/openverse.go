package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ---- Openverse Source ----

type OpenverseSource struct{}

func (s *OpenverseSource) Name() string    { return "openverse" }
func (s *OpenverseSource) NeedsKey() bool  { return false }
func (s *OpenverseSource) KeyName() string { return "" }

func (s *OpenverseSource) Description() string {
	return "Aggregates CC0/CC-BY images from Flickr, Wikimedia, and more — broadest general photo coverage"
}
func (s *OpenverseSource) Subjects() []string {
	return []string{"photos", "people", "places", "nature", "tech", "everyday"}
}
func (s *OpenverseSource) Licenses() []string {
	return []string{"CC0", "Public Domain Mark", "CC-BY", "CC-BY-SA", "CC-BY-NC"}
}

func (s *OpenverseSource) Search(query string, count int, licenseTier string, opts Opts) ([]Result, error) {
	params := url.Values{
		"q":           {query},
		"page_size":   {fmt.Sprintf("%d", count)},
		"filter_dead": {"true"},
	}
	if licenseTier == "free" {
		params.Set("license", "cc0,pdm")
	}

	searchURL := "https://api.openverse.org/v1/images/?" + params.Encode()
	headers := s.authHeaders()

	var data struct {
		Results []struct {
			Title             string `json:"title"`
			Creator           string `json:"creator"`
			CreatorURL        string `json:"creator_url"`
			License           string `json:"license"`
			LicenseVersion    string `json:"license_version"`
			LicenseURL        string `json:"license_url"`
			Attribution       string `json:"attribution"`
			URL               string `json:"url"`
			ForeignLandingURL string `json:"foreign_landing_url"`
			Width             int    `json:"width"`
			Height            int    `json:"height"`
		} `json:"results"`
	}
	if err := httpGetJSON(searchURL, headers, &data); err != nil {
		return nil, fmt.Errorf("openverse: %w", err)
	}

	var out []Result
	for _, r := range data.Results {
		lic := strings.ToUpper(r.License)
		if r.LicenseVersion != "" {
			lic = lic + " " + r.LicenseVersion
		}
		out = append(out, Result{
			Source:      "openverse",
			Title:       r.Title,
			Creator:     r.Creator,
			CreatorURL:  r.CreatorURL,
			License:     strings.TrimSpace(lic),
			LicenseURL:  r.LicenseURL,
			Attribution: r.Attribution,
			ImageURL:    r.URL,
			LandingURL:  r.ForeignLandingURL,
			Width:       r.Width,
			Height:      r.Height,
		})
	}
	return out, nil
}

// authHeaders returns OAuth bearer headers if credentials are configured.
func (s *OpenverseSource) authHeaders() map[string]string {
	token := s.getToken()
	if token != "" {
		return map[string]string{"Authorization": "Bearer " + token}
	}
	return nil
}

// getToken returns a valid Openverse bearer token, refreshing from cached credentials.
func (s *OpenverseSource) getToken() string {
	cid := configGet("openverse.client_id")
	csec := configGet("openverse.client_secret")
	if cid == "" || csec == "" {
		return ""
	}

	cachePath := filepath.Join(configDirPath, "token_cache.json")
	if cached := readTokenCache(cachePath); cached != "" {
		return cached
	}

	resp, err := httpPostForm("https://api.openverse.org/v1/auth_tokens/token/", url.Values{
		"client_id":     {cid},
		"client_secret": {csec},
		"grant_type":    {"client_credentials"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ! openverse token refresh failed: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	var d struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil || d.AccessToken == "" {
		return ""
	}

	expires := time.Now().Add(time.Duration(d.ExpiresIn) * time.Second).Unix()
	writeTokenCache(cachePath, d.AccessToken, expires)
	return d.AccessToken
}

func readTokenCache(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var cached struct {
		AccessToken string `json:"access_token"`
		ExpiresAt   int64  `json:"expires_at"`
	}
	if err := json.Unmarshal(data, &cached); err != nil {
		return ""
	}
	if cached.ExpiresAt > time.Now().Unix()+60 {
		return cached.AccessToken
	}
	return ""
}

func writeTokenCache(path, token string, expiresAt int64) {
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	data, _ := json.Marshal(map[string]any{
		"access_token": token,
		"expires_at":   expiresAt,
	})
	_ = os.WriteFile(path, data, 0600)
}

func init() {
	sources["openverse"] = &OpenverseSource{}
}
