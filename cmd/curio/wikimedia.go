package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ---- Wikimedia Commons Source ----

type WikimediaSource struct{}

func (s *WikimediaSource) Name() string    { return "wikimedia" }
func (s *WikimediaSource) NeedsKey() bool  { return false }
func (s *WikimediaSource) KeyName() string { return "" }

func (s *WikimediaSource) Description() string {
	return "Wikimedia Commons — landmarks, historical photos, diagrams, SVG illustrations, maps, technical drawings"
}
func (s *WikimediaSource) Subjects() []string {
	return []string{"landmarks", "history", "diagrams", "maps", "svg", "technical drawings"}
}
func (s *WikimediaSource) Licenses() []string {
	return []string{"Public Domain", "CC0", "CC-BY", "CC-BY-SA"}
}

func (s *WikimediaSource) Search(query string, count int, licenseTier string, opts Opts) ([]Result, error) {
	iiWidth := 1280
	if opts.Width > 0 {
		iiWidth = opts.Width
	}

	params := url.Values{
		"action":       {"query"},
		"generator":    {"search"},
		"gsrnamespace": {"6"},
		"gsrsearch":    {query},
		"gsrlimit":     {strconv.Itoa(count * 3)},
		"prop":         {"imageinfo"},
		"iiprop":       {"url|extmetadata|size|mime"},
		"iiurlwidth":   {strconv.Itoa(iiWidth)},
		"format":       {"json"},
		"origin":       {"*"},
		"maxlag":       {"5"},
	}
	searchURL := "https://commons.wikimedia.org/w/api.php?" + params.Encode()

	headers := map[string]string{}
	if wmUser := configGet("wikimedia.bot_user"); wmUser != "" {
		if wmPass := configGet("wikimedia.bot_pass"); wmPass != "" {
			creds := wmUser + ":" + wmPass
			headers["Authorization"] = "Basic " + base64Encode(creds)
		}
	}

	var data struct {
		Query struct {
			Pages map[string]struct {
				Title     string `json:"title"`
				Index     int    `json:"index"`
				Imageinfo []struct {
					URL            string `json:"url"`
					ThumbURL       string `json:"thumburl"`
					ThumbWidth     int    `json:"thumbwidth"`
					ThumbHeight    int    `json:"thumbheight"`
					DescriptionURL string `json:"descriptionurl"`
					Mime           string `json:"mime"`
					Extmetadata    struct {
						LicenseShortName struct {
							Value string `json:"value"`
						} `json:"LicenseShortName"`
						LicenseURL struct {
							Value string `json:"value"`
						} `json:"LicenseUrl"`
						Artist struct {
							Value string `json:"value"`
						} `json:"Artist"`
						ImageDescription struct {
							Value string `json:"value"`
						} `json:"ImageDescription"`
						Categories struct {
							Value string `json:"value"`
						} `json:"Categories"`
						DateTime struct {
							Value string `json:"value"`
						} `json:"DateTime"`
					} `json:"extmetadata"`
				} `json:"imageinfo"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := httpGetJSON(searchURL, headers, &data); err != nil {
		return nil, fmt.Errorf("wikimedia: %w", err)
	}

	type page struct {
		index int
		title string
		ii    struct {
			URL              string
			ThumbURL         string
			ThumbWidth       int
			ThumbHeight      int
			DescriptionURL   string
			LicenseShortName string
			LicenseURL       string
			Artist           string
			Description      string
			Categories       string
			DateTime         string
			Mime             string
		}
	}

	var pages []page
	for _, p := range data.Query.Pages {
		if len(p.Imageinfo) == 0 {
			continue
		}
		ii := p.Imageinfo[0]
		pg := page{
			index: p.Index,
			title: p.Title,
		}
		pg.ii.URL = ii.URL
		pg.ii.ThumbURL = ii.ThumbURL
		pg.ii.ThumbWidth = ii.ThumbWidth
		pg.ii.ThumbHeight = ii.ThumbHeight
		pg.ii.DescriptionURL = ii.DescriptionURL
		pg.ii.LicenseShortName = stripHTML(ii.Extmetadata.LicenseShortName.Value)
		pg.ii.LicenseURL = stripHTML(ii.Extmetadata.LicenseURL.Value)
		pg.ii.Artist = stripHTML(ii.Extmetadata.Artist.Value)
		pg.ii.Description = stripHTML(ii.Extmetadata.ImageDescription.Value)
		pg.ii.Categories = stripHTML(ii.Extmetadata.Categories.Value)
		pg.ii.DateTime = stripHTML(ii.Extmetadata.DateTime.Value)
		pg.ii.Mime = ii.Mime
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
		licName := p.ii.LicenseShortName
		if licenseTier == "free" && !isCC0orPD(licName, "") {
			continue
		}

		var imgURL string
		if opts.WantFull {
			imgURL = p.ii.URL
		} else {
			imgURL = p.ii.ThumbURL
		}
		if imgURL == "" {
			imgURL = p.ii.URL
		}

		title := strings.TrimPrefix(p.title, "File:")
		artist := p.ii.Artist
		if artist == "" {
			artist = "unknown"
		}

		attribution := fmt.Sprintf(`"%s" by %s — %s`, title, artist, licName)
		if p.ii.LicenseURL != "" {
			attribution += fmt.Sprintf(" (%s)", p.ii.LicenseURL)
		}

		meta := map[string]any{}
		if p.ii.Description != "" {
			meta["description"] = p.ii.Description
		}
		if p.ii.Categories != "" {
			cats := strings.Split(p.ii.Categories, "|")
			var tags []string
			for _, c := range cats {
				c = strings.TrimSpace(c)
				if c != "" {
					tags = append(tags, c)
				}
			}
			if len(tags) > 0 {
				meta["tags"] = tags
			}
		}
		if p.ii.DateTime != "" {
			meta["date"] = p.ii.DateTime
		}
		if p.ii.Mime != "" {
			meta["category"] = p.ii.Mime
		}

		out = append(out, Result{
			Source:       "wikimedia",
			Title:        title,
			Creator:      artist,
			License:      orDefaultStr(licName, "See item"),
			LicenseURL:   p.ii.LicenseURL,
			Attribution:  attribution,
			ImageURL:     imgURL,
			ThumbnailURL: p.ii.ThumbURL,
			LandingURL:   p.ii.DescriptionURL,
			Width:        p.ii.ThumbWidth,
			Height:       p.ii.ThumbHeight,
			Meta:         meta,
		})
		if len(out) >= count {
			break
		}
	}
	return out, nil
}

func init() {
	sources["wikimedia"] = &WikimediaSource{}
}
