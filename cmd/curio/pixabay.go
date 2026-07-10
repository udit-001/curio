package main

import (
	"fmt"
	"net/url"
)

// ---- Pixabay Source ----

type PixabaySource struct{}

func (s *PixabaySource) Name() string    { return "pixabay" }
func (s *PixabaySource) NeedsKey() bool  { return true }
func (s *PixabaySource) KeyName() string { return "pixabay.api_key" }

func (s *PixabaySource) Description() string {
	return "Mixed photos, illustrations, and vector graphics — broadest of the stock sites"
}
func (s *PixabaySource) Subjects() []string {
	return []string{"photos", "illustrations", "vectors", "nature", "technology", "backgrounds"}
}
func (s *PixabaySource) Licenses() []string {
	return []string{"Pixabay License"}
}

func (s *PixabaySource) Search(query string, count int, licenseTier string, opts Opts) ([]Result, error) {
	key := configGet(s.KeyName())

	searchURL := "https://pixabay.com/api/?" + url.Values{
		"key":        {key},
		"q":          {query},
		"per_page":   {fmt.Sprintf("%d", count)},
		"image_type": {"all"},
	}.Encode()

	var data struct {
		Hits []struct {
			Tags          string `json:"tags"`
			User          string `json:"user"`
			PageURL       string `json:"pageURL"`
			PreviewURL    string `json:"previewURL"`
			WebformatURL  string `json:"webformatURL"`
			LargeImageURL string `json:"largeImageURL"`
			ImageWidth    int    `json:"imageWidth"`
			ImageHeight   int    `json:"imageHeight"`
		} `json:"hits"`
	}
	if err := httpGetJSON(searchURL, nil, &data); err != nil {
		return nil, fmt.Errorf("pixabay: %w", err)
	}

	var out []Result
	for _, h := range data.Hits {
		var imgURL string
		if opts.WantFull {
			imgURL = h.LargeImageURL
		} else if opts.Width > 0 && opts.Width <= 640 {
			imgURL = h.WebformatURL
		} else {
			imgURL = h.LargeImageURL
		}
		if imgURL == "" {
			imgURL = h.WebformatURL
		}
		if imgURL == "" {
			continue
		}

		out = append(out, Result{
			Source:      "pixabay",
			Title:       h.Tags,
			Creator:     h.User,
			License:     "Pixabay Content License (no attribution required)",
			LicenseURL:  "https://pixabay.com/service/license-summary/",
			Attribution: fmt.Sprintf(`"%s" by %s — Pixabay License`, h.Tags, orDefaultStr(h.User, "unknown")),
			ImageURL:    imgURL,
			LandingURL:  h.PageURL,
			Width:       h.ImageWidth,
			Height:      h.ImageHeight,
		})
		if len(out) >= count {
			break
		}
	}
	return out, nil
}

func init() {
	sources["pixabay"] = &PixabaySource{}
}
