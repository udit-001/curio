package main

import (
	"fmt"
	"net/url"
	"strings"
)

// ---- V&A Museum Source ----

type VaSource struct{}

func (s *VaSource) Name() string    { return "va" }
func (s *VaSource) NeedsKey() bool  { return false }
func (s *VaSource) KeyName() string { return "" }

func (s *VaSource) Description() string {
	return "V&A Museum — decorative arts, design, fashion, textiles, ceramics. The world's leading museum of art and design"
}
func (s *VaSource) Subjects() []string {
	return []string{"decorative arts", "design", "fashion", "textiles", "ceramics"}
}
func (s *VaSource) Licenses() []string {
	return []string{"CC0", "Public Domain", "Various"}
}

func (s *VaSource) Search(query string, count int, licenseTier string, opts Opts) ([]Result, error) {
	searchURL := "https://api.vam.ac.uk/v2/objects/search?" + url.Values{
		"q":         {query},
		"page_size": {fmt.Sprintf("%d", count)},
	}.Encode()

	var data struct {
		Records []struct {
			SystemNumber string `json:"systemNumber"`
			PrimaryTitle string `json:"_primaryTitle"`
			PrimaryMaker struct {
				Name        string `json:"name"`
				Association string `json:"association"`
			} `json:"_primaryMaker"`
			ObjectType     string `json:"_objectType"`
			PrimaryImageID string `json:"_primaryImageId"`
			Images         struct {
				IiifBaseURL string `json:"_iiif_image_base_url"`
			} `json:"_images"`
			Rights string `json:"rights"`
		} `json:"records"`
		Info struct {
			RecordCount int `json:"record_count"`
		} `json:"info"`
	}
	if err := httpGetJSON(searchURL, nil, &data); err != nil {
		return nil, fmt.Errorf("va: %w", err)
	}

	var out []Result
	for _, r := range data.Records {
		if r.PrimaryImageID == "" {
			continue
		}

		iiifBase := r.Images.IiifBaseURL
		if iiifBase == "" {
			iiifBase = "https://framemark.vam.ac.uk/collections/" + r.PrimaryImageID + "/"
		}

		var imgURL string
		width := 1280
		if opts.WantFull {
			imgURL = iiifBase + "full/full/0/default.jpg"
		} else if opts.Width > 0 {
			width = opts.Width
			imgURL = fmt.Sprintf("%s/full/%d,/0/default.jpg", iiifBase, opts.Width)
		} else {
			imgURL = iiifBase + "full/!1280,1280/0/default.jpg"
		}

		license := "See item"
		licenseURL := ""
		if r.Rights != "" {
			license = r.Rights
			low := strings.ToLower(r.Rights)
			if strings.Contains(low, "cc0") || strings.Contains(low, "public domain") {
				license = "CC0"
				licenseURL = "https://creativecommons.org/publicdomain/zero/1.0/"
			}
		}

		if licenseTier == "cc0,pd" && license != "CC0" {
			continue
		}

		title := r.PrimaryTitle
		if title == "" {
			title = r.ObjectType
		}

		maker := r.PrimaryMaker.Name

		out = append(out, Result{
			Source:      "va",
			Title:       title,
			Creator:     maker,
			License:     license,
			LicenseURL:  licenseURL,
			Attribution: fmt.Sprintf(`"%s" by %s — %s (V&A Museum)`, title, orDefaultStr(maker, "unknown"), license),
			ImageURL:    imgURL,
			LandingURL:  fmt.Sprintf("https://collections.vam.ac.uk/item/%s", r.SystemNumber),
			Width:       width,
		})
		if len(out) >= count {
			break
		}
	}
	return out, nil
}

func init() {
	sources["va"] = &VaSource{}
}
