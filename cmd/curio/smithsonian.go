package main

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// ---- Smithsonian Open Access Source ----

type SmithsonianSource struct{}

func (s *SmithsonianSource) Name() string    { return "smithsonian" }
func (s *SmithsonianSource) NeedsKey() bool  { return true }
func (s *SmithsonianSource) KeyName() string { return "smithsonian.api_key" }

func (s *SmithsonianSource) Description() string {
	return "5.1M CC0 items across 21 museums — natural history, air & space, American history, cultural artifacts"
}
func (s *SmithsonianSource) Subjects() []string {
	return []string{"science", "natural history", "air & space", "american history", "cultural artifacts"}
}
func (s *SmithsonianSource) Licenses() []string {
	return []string{"CC0"}
}

func (s *SmithsonianSource) Search(query string, count int, licenseTier string, opts Opts) ([]Result, error) {
	key := configGet(s.KeyName())

	fqs, _ := json.Marshal([]string{"online_media_type:Images"})

	params := url.Values{
		"q":       {query},
		"rows":    {fmt.Sprintf("%d", count)},
		"api_key": {key},
		"fqs":     {string(fqs)},
	}
	searchURL := "https://api.si.edu/openaccess/api/v1.0/search?" + params.Encode()

	var data struct {
		ResponseCode int `json:"responseCode"`
		Response     struct {
			RowCount int `json:"rowCount"`
			Rows     []struct {
				ID      string `json:"id"`
				Title   string `json:"title"`
				URL     string `json:"url"`
				Content struct {
					Freetext map[string][]struct {
						Label   string `json:"label"`
						Content string `json:"content"`
					} `json:"freetext"`
					DescriptiveNonRepeating struct {
						RecordLink  string `json:"record_link"`
						OnlineMedia *struct {
							Media []struct {
								IdsID string `json:"idsId"`
								Usage struct {
									Access string `json:"access"`
								} `json:"usage"`
								Resources []struct {
									Label  string `json:"label"`
									URL    string `json:"url"`
									Width  int    `json:"width"`
									Height int    `json:"height"`
								} `json:"resources"`
							} `json:"media"`
						} `json:"online_media"`
					} `json:"descriptiveNonRepeating"`
				} `json:"content"`
			} `json:"rows"`
		} `json:"response"`
	}
	if err := httpGetJSON(searchURL, nil, &data); err != nil {
		return nil, fmt.Errorf("smithsonian: %w", err)
	}
	if data.ResponseCode != 1 {
		return nil, nil
	}

	var out []Result
	for _, row := range data.Response.Rows {
		om := row.Content.DescriptiveNonRepeating.OnlineMedia
		if om == nil || len(om.Media) == 0 {
			continue
		}

		media := om.Media[0]

		var imgURL string
		var width, height int
		for _, res := range media.Resources {
			if res.Label == "Screen Image" {
				imgURL = res.URL
				width = res.Width
				height = res.Height
				break
			}
		}
		if imgURL == "" {
			for _, res := range media.Resources {
				if res.Label == "High-resolution JPEG" {
					imgURL = res.URL
					width = res.Width
					height = res.Height
					break
				}
			}
		}
		if imgURL == "" {
			continue
		}

		creator := ""
		if freetext, ok := row.Content.Freetext["maker"]; ok && len(freetext) > 0 {
			creator = freetext[0].Content
		}
		if creator == "" {
			if freetext, ok := row.Content.Freetext["creator"]; ok && len(freetext) > 0 {
				creator = freetext[0].Content
			}
		}

		license := "CC0"
		if media.Usage.Access != "" {
			license = media.Usage.Access
		}

		out = append(out, Result{
			Source:      "smithsonian",
			Title:       row.Title,
			Creator:     creator,
			License:     license,
			LicenseURL:  "https://creativecommons.org/publicdomain/zero/1.0/",
			Attribution: fmt.Sprintf(`"%s" — CC0 (Smithsonian Open Access)`, row.Title),
			ImageURL:    imgURL,
			LandingURL:  row.Content.DescriptiveNonRepeating.RecordLink,
			Width:       width,
			Height:      height,
		})
		if len(out) >= count {
			break
		}
	}
	return out, nil
}

func init() {
	sources["smithsonian"] = &SmithsonianSource{}
}
