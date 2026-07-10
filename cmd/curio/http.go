package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	userAgent  = "curio/2.0 (https://github.com/udit-001/curio; opencode skill)"
	timeout    = 30 * time.Second
	maxRetries = 3
)

var retryDelays = []time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second}

// httpClient is the shared HTTP client with retry/backoff.
var httpClient = &http.Client{Timeout: timeout}

// httpGet performs a GET request with retry on 429 and returns the response body.
func httpGet(url string, headers map[string]string) (*http.Response, error) {
	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == 429 {
			resp.Body.Close()
			delay := retryDelays[attempt]
			fmt.Fprintf(os.Stderr, "  ! rate limited (429) — backing off %v (attempt %d/%d)\n", delay, attempt+1, maxRetries)
			time.Sleep(delay)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded for %s", url)
}

// httpGetJSON fetches a URL, decodes the JSON response into target, and closes the body.
// Replaces the repeated httpGet → defer Close → json.Decode idiom.
func httpGetJSON(url string, headers map[string]string, target interface{}) error {
	resp, err := httpGet(url, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode error: %w", err)
	}
	return nil
}

func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
