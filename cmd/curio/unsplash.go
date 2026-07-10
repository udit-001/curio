package main

import (
	"fmt"
	"net/url"
)

// ---- Unsplash Source ----

type UnsplashSource struct{}

func (s *UnsplashSource) Name() string    { return "unsplash" }
func (s *UnsplashSource) NeedsKey() bool  { return true }
func (s *UnsplashSource) KeyName() string { return "unsplash.access_key" }

func (s *UnsplashSource) Description() string {
	return "High-quality editorial photography — landscapes, architecture, objects, textures"
}
func (s *UnsplashSource) Subjects() []string {
	return []string{"photography", "landscapes", "architecture", "textures", "editorial"}
}
func (s *UnsplashSource) Licenses() []string {
	return []string{"Unsplash License"}
}

func (s *UnsplashSource) Search(query string, count int, licenseTier string, opts Opts) ([]Result, error) {
	key := configGet(s.KeyName())
	if key == "" {
		return nil, fmt.Errorf("Unsplash requires a free API key — register an app at https://unsplash.com/oauth/applications and run 'curio setup'")
	}

	searchURL := "https://api.unsplash.com/search/photos?" + url.Values{
		"query":    {query},
		"per_page": {fmt.Sprintf("%d", count)},
	}.Encode()

	headers := map[string]string{
		"Authorization": "Client-ID " + key,
	}

	var data struct {
		Results []struct {
			AltDescription string `json:"alt_description"`
			URLs           struct {
				Raw     string `json:"raw"`
				Full    string `json:"full"`
				Regular string `json:"regular"`
				Small   string `json:"small"`
				Thumb   string `json:"thumb"`
			} `json:"urls"`
			Width  int `json:"width"`
			Height int `json:"height"`
			User   struct {
				Name  string `json:"name"`
				Links struct {
					HTML string `json:"html"`
				} `json:"links"`
			} `json:"user"`
			Links struct {
				HTML string `json:"html"`
			} `json:"links"`
		} `json:"results"`
	}
	if err := httpGetJSON(searchURL, headers, &data); err != nil {
		return nil, fmt.Errorf("unsplash: %w", err)
	}

	var out []Result
	for _, r := range data.Results {
		var imgURL string
		if opts.WantFull {
			imgURL = r.URLs.Full
		} else if opts.Width > 0 && opts.Width <= 400 {
			imgURL = r.URLs.Small
		} else {
			imgURL = r.URLs.Regular
		}
		if imgURL == "" {
			continue
		}

		title := r.AltDescription
		if title == "" {
			title = "Untitled"
		}

		out = append(out, Result{
			Source:      "unsplash",
			Title:       title,
			Creator:     r.User.Name,
			CreatorURL:  r.User.Links.HTML,
			License:     "Unsplash License (no attribution required)",
			LicenseURL:  "https://unsplash.com/license",
			Attribution: fmt.Sprintf(`"%s" by %s — Unsplash License`, title, orDefaultStr(r.User.Name, "unknown")),
			ImageURL:    imgURL,
			LandingURL:  r.Links.HTML,
			Width:       r.Width,
			Height:      r.Height,
		})
		if len(out) >= count {
			break
		}
	}
	return out, nil
}

func init() {
	sources["unsplash"] = &UnsplashSource{}
}
