package main

import (
	"fmt"
	"net/url"
)

// ---- Wikipedia Source ----
// Searches Wikipedia articles and returns each article's primary image (infobox image).
// Different from the "wikimedia" source which searches Wikimedia Commons by filename —
// this finds the article about the subject and returns the curated image editors chose.

type WikipediaSource struct{}

func (s *WikipediaSource) Name() string    { return "wikipedia" }
func (s *WikipediaSource) NeedsKey() bool  { return false }
func (s *WikipediaSource) KeyName() string { return "" }

func (s *WikipediaSource) Description() string {
	return "Curated infobox images from Wikipedia articles — editors' chosen image per subject. Broadest coverage"
}
func (s *WikipediaSource) Subjects() []string {
	return []string{"any", "education", "science", "history", "art", "geography"}
}
func (s *WikipediaSource) Licenses() []string {
	return []string{"Various"}
}

func (s *WikipediaSource) Search(query string, count int, licenseTier string, opts Opts) ([]Result, error) {
	thumbSize := 1280
	if opts.Width > 0 {
		thumbSize = opts.Width
	}

	searchURL := "https://en.wikipedia.org/w/api.php?" + url.Values{
		"action":       {"query"},
		"generator":    {"search"},
		"gsrsearch":    {query},
		"gsrlimit":     {fmt.Sprintf("%d", count)},
		"gsrnamespace": {"0"},
		"prop":         {"pageimages"},
		"piprop":       {"thumbnail|original"},
		"pithumbsize":  {fmt.Sprintf("%d", thumbSize)},
		"format":       {"json"},
		"origin":       {"*"},
	}.Encode()

	var data struct {
		Query struct {
			Pages map[string]struct {
				Title     string `json:"title"`
				Index     int    `json:"index"`
				Thumbnail *struct {
					Source string `json:"source"`
					Width  int    `json:"width"`
					Height int    `json:"height"`
				} `json:"thumbnail"`
				Original *struct {
					Source string `json:"source"`
				} `json:"original"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := httpGetJSON(searchURL, nil, &data); err != nil {
		return nil, fmt.Errorf("wikipedia: %w", err)
	}

	type page struct {
		index  int
		title  string
		thumb  string
		thumbW int
		thumbH int
		orig   string
	}

	var pages []page
	for _, p := range data.Query.Pages {
		pg := page{
			index: p.Index,
			title: p.Title,
		}
		if p.Thumbnail != nil {
			pg.thumb = p.Thumbnail.Source
			pg.thumbW = p.Thumbnail.Width
			pg.thumbH = p.Thumbnail.Height
		}
		if p.Original != nil {
			pg.orig = p.Original.Source
		}
		pages = append(pages, pg)
	}

	for i := 0; i < len(pages); i++ {
		for j := i + 1; j < len(pages); j++ {
			if pages[j].index < pages[i].index {
				pages[i], pages[j] = pages[j], pages[i]
			}
		}
	}

	var out []Result
	for _, p := range pages {
		var imgURL string
		if opts.WantFull && p.orig != "" {
			imgURL = p.orig
		} else if p.thumb != "" {
			imgURL = p.thumb
		} else if p.orig != "" {
			imgURL = p.orig
		}
		if imgURL == "" {
			continue
		}

		landingURL := "https://en.wikipedia.org/wiki/" + url.PathEscape(p.title)

		out = append(out, Result{
			Source:      "wikipedia",
			Title:       p.title,
			Creator:     "Wikipedia contributors",
			License:     "Various (see image source)",
			LicenseURL:  landingURL,
			Attribution: fmt.Sprintf(`"%s" — image from Wikipedia article`, p.title),
			ImageURL:    imgURL,
			LandingURL:  landingURL,
			Width:       p.thumbW,
			Height:      p.thumbH,
		})
		if len(out) >= count {
			break
		}
	}
	return out, nil
}

func init() {
	sources["wikipedia"] = &WikipediaSource{}
}
