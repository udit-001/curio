package main

import (
	"fmt"
	"net/url"
	"strings"
)

// ---- Library of Congress Source ----

type LocSource struct{}

func (s *LocSource) Name() string    { return "loc" }
func (s *LocSource) NeedsKey() bool  { return false }
func (s *LocSource) KeyName() string { return "" }

func (s *LocSource) Description() string {
	return "Library of Congress — historical photos, manuscripts, maps, prints, newspapers, film stills. ~1M items"
}
func (s *LocSource) Subjects() []string {
	return []string{"history", "photos", "maps", "manuscripts", "prints", "newspapers"}
}
func (s *LocSource) Licenses() []string {
	return []string{"Public Domain", "Various"}
}

func (s *LocSource) Search(query string, count int, licenseTier string, opts Opts) ([]Result, error) {
	searchURL := "https://www.loc.gov/photos/?" + url.Values{
		"q":  {query},
		"fo": {"json"},
		"c":  {fmt.Sprintf("%d", count)},
	}.Encode()

	var data struct {
		Results []struct {
			ID               string   `json:"id"`
			Title            string   `json:"title"`
			Description      string   `json:"description"`
			AccessRestricted bool     `json:"access_restricted"`
			Date             string   `json:"date"`
			Subject          []string `json:"subject"`
			Format           []string `json:"format"`
			ImageURL         []string `json:"image_url"`
			Item             struct {
				RightsAdvisory string `json:"rights_advisory"`
				Creators       []struct {
					Title string `json:"title"`
				} `json:"creators"`
			} `json:"item"`
		} `json:"results"`
	}
	if err := httpGetJSON(searchURL, nil, &data); err != nil {
		return nil, fmt.Errorf("loc: %w", err)
	}

	var out []Result
	for _, r := range data.Results {
		if len(r.ImageURL) == 0 {
			continue
		}

		rights := r.Item.RightsAdvisory

		imgURL := s.pickImageURL(r.ImageURL, opts)
		if opts.WantFull {
			fullURL := s.fetchFullRes(r.ID, r.ImageURL)
			if fullURL != "" {
				imgURL = fullURL
			}
		}
		if imgURL == "" {
			continue
		}

		license := rights
		if strings.Contains(strings.ToLower(rights), "no known restrictions") {
			license = "Public domain"
		}

		creator := ""
		if len(r.Item.Creators) > 0 {
			creator = r.Item.Creators[0].Title
		}

		thumbnailURL := ""
		if len(r.ImageURL) > 0 {
			thumbnailURL = stripFragment(r.ImageURL[0])
		}

		meta := map[string]any{}
		if r.Description != "" {
			meta["description"] = r.Description
		}
		if len(r.Subject) > 0 {
			meta["tags"] = r.Subject
		}
		if r.Date != "" {
			meta["date"] = r.Date
		}
		if len(r.Format) > 0 {
			meta["category"] = r.Format[0]
		}

		out = append(out, Result{
			Source:       "loc",
			Title:        r.Title,
			Creator:      creator,
			License:      license,
			Attribution:  fmt.Sprintf(`"%s" — %s (Library of Congress)`, r.Title, license),
			ImageURL:     imgURL,
			ThumbnailURL: thumbnailURL,
			LandingURL:   r.ID,
			Meta:         meta,
		})
		if len(out) >= count {
			break
		}
	}
	return out, nil
}

func (s *LocSource) pickImageURL(urls []string, opts Opts) string {
	if len(urls) == 0 {
		return ""
	}

	if opts.WantFull {
		return ""
	}

	var idx int
	if opts.Width > 0 && opts.Width <= 640 {
		idx = 1
	} else {
		idx = 2
	}
	if idx >= len(urls) {
		idx = len(urls) - 1
	}

	return stripFragment(urls[idx])
}

func (s *LocSource) fetchFullRes(itemID string, imageURLs []string) string {
	if len(imageURLs) == 0 {
		return ""
	}

	if itemID == "" {
		return stripFragment(imageURLs[len(imageURLs)-1])
	}

	detailURL := itemID + "?fo=json&at=resources"
	var detail struct {
		Resources []struct {
			Files [][]struct {
				Mimetype string `json:"mimetype"`
				URL      string `json:"url"`
			} `json:"files"`
		} `json:"resources"`
	}
	if err := httpGetJSON(detailURL, nil, &detail); err != nil {
		return stripFragment(imageURLs[len(imageURLs)-1])
	}

	for _, res := range detail.Resources {
		for _, fileGroup := range res.Files {
			for _, f := range fileGroup {
				if f.Mimetype == "image/tiff" {
					return f.URL
				}
			}
		}
	}

	return stripFragment(imageURLs[len(imageURLs)-1])
}

func stripFragment(s string) string {
	if idx := strings.Index(s, "#"); idx >= 0 {
		return s[:idx]
	}
	return s
}

func init() {
	sources["loc"] = &LocSource{}
}
