package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
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

// doWithRetry executes a request with retry on 429.
func doWithRetry(req *http.Request) (*http.Response, error) {
	for attempt := 0; attempt < maxRetries; attempt++ {
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

	return nil, fmt.Errorf("max retries exceeded for %s", req.URL.String())
}

// httpGet performs a GET request with retry on 429 and returns the response.
func httpGet(url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return doWithRetry(req)
}

// httpPostForm performs a POST request with form values, retry on 429, and returns the response.
func httpPostForm(url string, values url.Values) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return doWithRetry(req)
}

// httpGetJSON fetches a URL, decodes the JSON response into target, and closes the body.
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
