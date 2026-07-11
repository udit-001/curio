package main

import (
	"fmt"
	"net/url"
)

// ---- NASA Source ----

type NasaSource struct{}

func (s *NasaSource) Name() string    { return "nasa" }
func (s *NasaSource) NeedsKey() bool  { return false }
func (s *NasaSource) KeyName() string { return "" }

func (s *NasaSource) Description() string {
	return "NASA Image Library — space, astronomy, aeronautics, earth-from-orbit. All public domain"
}
func (s *NasaSource) Subjects() []string {
	return []string{"space", "astronomy", "science", "aeronautics", "earth"}
}
func (s *NasaSource) Licenses() []string {
	return []string{"Public Domain"}
}

func (s *NasaSource) Search(query string, count int, licenseTier string, opts Opts) ([]Result, error) {
	searchURL := "https://images-api.nasa.gov/search?" + url.Values{
		"q":          {query},
		"media_type": {"image"},
	}.Encode()

	var data struct {
		Collection struct {
			Items []struct {
				Href string `json:"href"`
				Data []struct {
					Title            string   `json:"title"`
					SecondaryCreator string   `json:"secondary_creator"`
					Description      string   `json:"description"`
					Keywords         []string `json:"keywords"`
					DateCreated      string   `json:"date_created"`
					Location         string   `json:"location"`
					Center           string   `json:"center"`
				} `json:"data"`
				Links []struct {
					Href   string `json:"href"`
					Rel    string `json:"rel"`
					Render string `json:"render"`
					Width  int    `json:"width"`
					Height int    `json:"height"`
				} `json:"links"`
			} `json:"items"`
		} `json:"collection"`
	}
	if err := httpGetJSON(searchURL, nil, &data); err != nil {
		return nil, fmt.Errorf("nasa: %w", err)
	}

	var out []Result
	for _, item := range data.Collection.Items {
		if len(item.Data) == 0 {
			continue
		}
		d := item.Data[0]
		title := d.Title
		creator := d.SecondaryCreator
		if creator == "" {
			creator = "NASA"
		}

		var canonical, preview string
		var cw, ch int
		for _, link := range item.Links {
			if link.Rel == "canonical" && link.Render == "image" {
				canonical = link.Href
				cw = link.Width
				ch = link.Height
			} else if link.Rel == "preview" {
				preview = link.Href
			}
		}

		imgURL := canonical
		if !opts.WantFull && preview != "" {
			imgURL = preview
		}
		if imgURL == "" {
			continue
		}

		var tags []string
		tags = append(tags, d.Keywords...)
		if d.Center != "" {
			tags = append(tags, d.Center)
		}
		if d.Location != "" {
			tags = append(tags, d.Location)
		}

		meta := map[string]any{}
		if len(tags) > 0 {
			meta["tags"] = tags
		}
		if d.Description != "" {
			meta["description"] = d.Description
		}
		if d.DateCreated != "" {
			meta["date"] = d.DateCreated
		}
		if d.Location != "" {
			meta["location"] = d.Location
		}

		out = append(out, Result{
			Source:       "nasa",
			Title:        title,
			Creator:      creator,
			License:      "Public domain (NASA)",
			LicenseURL:   "https://www.nasa.gov/about/about_nasa.html",
			Attribution:  fmt.Sprintf(`"%s" by NASA is in the public domain.`, title),
			ImageURL:     imgURL,
			ThumbnailURL: preview,
			LandingURL:   item.Href,
			Width:        cw,
			Height:       ch,
			Meta:         meta,
		})
		if len(out) >= count {
			break
		}
	}
	return out, nil
}

func init() {
	sources["nasa"] = &NasaSource{}
}
