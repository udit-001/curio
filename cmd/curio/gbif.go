package main

import (
	"fmt"
	"net/url"
)

// ---- GBIF Source ----

type GbifSource struct{}

func (s *GbifSource) Name() string    { return "gbif" }
func (s *GbifSource) NeedsKey() bool  { return false }
func (s *GbifSource) KeyName() string { return "" }

func (s *GbifSource) Description() string {
	return "Real organism photos with taxonomic data — complements PhyloPic silhouettes with actual species photos"
}
func (s *GbifSource) Subjects() []string {
	return []string{"biology", "organisms", "species", "taxonomy", "wildlife"}
}
func (s *GbifSource) Licenses() []string {
	return []string{"CC0", "CC-BY", "CC-BY-NC"}
}

func (s *GbifSource) Search(query string, count int, licenseTier string, opts Opts) ([]Result, error) {
	searchURL := "https://api.gbif.org/v1/occurrence/search?" + url.Values{
		"q":         {query},
		"mediaType": {"StillImage"},
		"limit":     {fmt.Sprintf("%d", count)},
	}.Encode()

	var data struct {
		Results []struct {
			Key              int      `json:"key"`
			Species          string   `json:"species"`
			Genus            string   `json:"genus"`
			Family           string   `json:"family"`
			Order            string   `json:"order"`
			Class            string   `json:"class"`
			Phylum           string   `json:"phylum"`
			Kingdom          string   `json:"kingdom"`
			ScientificName   string   `json:"scientificName"`
			VernacularName   string   `json:"vernacularName"`
			RecordedBy       string   `json:"recordedBy"`
			EventDate        string   `json:"eventDate"`
			Country          string   `json:"country"`
			DecimalLatitude  *float64 `json:"decimalLatitude"`
			DecimalLongitude *float64 `json:"decimalLongitude"`
			RightsHolder     string   `json:"rightsHolder"`
			License          string   `json:"license"`
			Media            []struct {
				Identifier   string `json:"identifier"`
				License      string `json:"license"`
				RightsHolder string `json:"rightsHolder"`
				Format       string `json:"format"`
			} `json:"media"`
		} `json:"results"`
	}
	if err := httpGetJSON(searchURL, nil, &data); err != nil {
		return nil, fmt.Errorf("gbif: %w", err)
	}

	var out []Result
	for _, r := range data.Results {
		if len(r.Media) == 0 {
			continue
		}

		media := r.Media[0]
		imgURL := media.Identifier
		if imgURL == "" {
			continue
		}

		license := licenseFromURL(r.License)
		if licenseTier == "free" && !isCC0orPD(license, r.License) {
			continue
		}

		title := r.VernacularName
		if title == "" {
			title = r.ScientificName
		}
		if title == "" {
			title = r.Genus
		}

		creator := r.RecordedBy
		if creator == "" {
			creator = media.RightsHolder
		}

		var tags []string
		for _, t := range []string{r.Kingdom, r.Phylum, r.Class, r.Order, r.Family, r.Genus, r.Species} {
			if t != "" {
				tags = append(tags, t)
			}
		}
		meta := map[string]any{}
		if len(tags) > 0 {
			meta["tags"] = tags
		}
		if r.VernacularName != "" {
			meta["description"] = r.VernacularName
		}
		if r.EventDate != "" {
			meta["date"] = r.EventDate
		}
		if r.Country != "" {
			meta["location"] = r.Country
		}
		meta["category"] = "wildlife photography"

		out = append(out, Result{
			Source:      "gbif",
			Title:       title,
			Creator:     creator,
			License:     license,
			LicenseURL:  r.License,
			Attribution: fmt.Sprintf(`"%s" by %s — %s (GBIF)`, title, orDefaultStr(creator, "unknown"), license),
			ImageURL:    imgURL,
			LandingURL:  fmt.Sprintf("https://www.gbif.org/occurrence/%d", r.Key),
			Meta:        meta,
		})
		if len(out) >= count {
			break
		}
	}
	return out, nil
}

func init() {
	sources["gbif"] = &GbifSource{}
}
