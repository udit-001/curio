package main

import (
	"fmt"
	"net/url"
)

// ---- Pexels Source ----

type PexelsSource struct{}

func (s *PexelsSource) Name() string    { return "pexels" }
func (s *PexelsSource) NeedsKey() bool  { return true }
func (s *PexelsSource) KeyName() string { return "pexels.api_key" }

func (s *PexelsSource) Description() string {
	return "Modern everyday photography — people, business, food, nature, lifestyle. Professional quality stock photos"
}
func (s *PexelsSource) Subjects() []string {
	return []string{"photos", "people", "business", "food", "nature", "lifestyle"}
}
func (s *PexelsSource) Licenses() []string {
	return []string{"Pexels License"}
}

func (s *PexelsSource) Search(query string, count int, licenseTier string, opts Opts) ([]Result, error) {
	key := configGet(s.KeyName())

	searchURL := "https://api.pexels.com/v1/search?" + url.Values{
		"query":       {query},
		"per_page":    {fmt.Sprintf("%d", count)},
		"orientation": {"landscape"},
	}.Encode()

	headers := map[string]string{
		"Authorization": key,
	}

	var data struct {
		Photos []struct {
			ID              int    `json:"id"`
			Alt             string `json:"alt"`
			Photographer    string `json:"photographer"`
			PhotographerURL string `json:"photographer_url"`
			URL             string `json:"url"`
			Src             struct {
				Original string `json:"original"`
				Large2x  string `json:"large2x"`
				Large    string `json:"large"`
				Medium   string `json:"medium"`
				Small    string `json:"small"`
				Portrait string `json:"portrait"`
			} `json:"src"`
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"photos"`
	}
	if err := httpGetJSON(searchURL, headers, &data); err != nil {
		return nil, fmt.Errorf("pexels: %w", err)
	}

	var out []Result
	for _, p := range data.Photos {
		var imgURL string
		if opts.WantFull {
			imgURL = p.Src.Original
		} else if opts.Width > 0 && opts.Width <= 640 {
			imgURL = p.Src.Medium
		} else {
			imgURL = p.Src.Large2x
		}
		if imgURL == "" {
			imgURL = p.Src.Large
		}
		if imgURL == "" {
			continue
		}

		out = append(out, Result{
			Source:      "pexels",
			Title:       p.Alt,
			Creator:     p.Photographer,
			CreatorURL:  p.PhotographerURL,
			License:     "Pexels License (no attribution required)",
			LicenseURL:  "https://www.pexels.com/license/",
			Attribution: fmt.Sprintf(`"%s" by %s — Pexels License`, p.Alt, orDefaultStr(p.Photographer, "unknown")),
			ImageURL:    imgURL,
			LandingURL:  p.URL,
			Width:       p.Width,
			Height:      p.Height,
		})
		if len(out) >= count {
			break
		}
	}
	return out, nil
}

func init() {
	sources["pexels"] = &PexelsSource{}
}
