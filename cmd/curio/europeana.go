package main

import (
	"fmt"
	"net/url"
)

// ---- Europeana Source ----

type EuropeanaSource struct{}

func (s *EuropeanaSource) Name() string    { return "europeana" }
func (s *EuropeanaSource) NeedsKey() bool  { return true }
func (s *EuropeanaSource) KeyName() string { return "europeana.api_key" }

func (s *EuropeanaSource) Description() string {
	return "EU aggregator — 4000+ institutions (British Library, Rijksmuseum, Louvre). Art, historical documents, manuscripts, maps"
}
func (s *EuropeanaSource) Subjects() []string {
	return []string{"european heritage", "art", "history", "manuscripts", "maps", "photographs"}
}
func (s *EuropeanaSource) Licenses() []string {
	return []string{"CC0", "Public Domain Mark", "CC-BY", "CC-BY-SA"}
}

func (s *EuropeanaSource) Search(query string, count int, licenseTier string, opts Opts) ([]Result, error) {
	key := configGet(s.KeyName())
	if key == "" {
		return nil, fmt.Errorf("Europeana requires a free API key — get one at https://pro.europeana.eu/pages/get-api and run 'curio setup'")
	}

	params := url.Values{
		"query":     {query},
		"rows":      {fmt.Sprintf("%d", count)},
		"wskey":     {key},
		"profile":   {"rich"},
		"qf":        {"TYPE:IMAGE"},
		"media":     {"true"},
		"thumbnail": {"true"},
	}

	params.Set("reusability", "open")

	searchURL := "https://api.europeana.eu/record/v2/search.json?" + params.Encode()

	var data struct {
		Success bool `json:"success"`
		Items   []struct {
			ID           string   `json:"id"`
			Title        []string `json:"title"`
			DataProvider []string `json:"dataProvider"`
			Rights       []string `json:"rights"`
			EdmIsShownBy []string `json:"edmIsShownBy"`
			EdmPreview   []string `json:"edmPreview"`
			GUID         string   `json:"guid"`
			Link         string   `json:"link"`
		} `json:"items"`
	}
	if err := httpGetJSON(searchURL, nil, &data); err != nil {
		return nil, fmt.Errorf("europeana: %w", err)
	}

	var out []Result
	for _, item := range data.Items {
		license := "Unknown"
		licenseURL := ""
		if len(item.Rights) > 0 {
			licenseURL = item.Rights[0]
			license = licenseFromURL(licenseURL)
		}

		if licenseTier == "cc0,pd" && !isCC0orPD(license, licenseURL) {
			continue
		}

		var imgURL string
		if opts.WantFull && len(item.EdmIsShownBy) > 0 {
			imgURL = item.EdmIsShownBy[0]
		} else if len(item.EdmPreview) > 0 {
			imgURL = item.EdmPreview[0]
		} else if len(item.EdmIsShownBy) > 0 {
			imgURL = item.EdmIsShownBy[0]
		}
		if imgURL == "" {
			continue
		}

		title := ""
		if len(item.Title) > 0 {
			title = item.Title[0]
		}

		creator := ""
		if len(item.DataProvider) > 0 {
			creator = item.DataProvider[0]
		}

		landingURL := item.GUID
		if landingURL == "" {
			landingURL = item.Link
		}

		out = append(out, Result{
			Source:      "europeana",
			Title:       title,
			Creator:     creator,
			License:     license,
			LicenseURL:  licenseURL,
			Attribution: fmt.Sprintf(`"%s" — %s (%s, Europeana)`, title, license, orDefaultStr(creator, "unknown")),
			ImageURL:    imgURL,
			LandingURL:  landingURL,
		})
		if len(out) >= count {
			break
		}
	}
	return out, nil
}

func init() {
	sources["europeana"] = &EuropeanaSource{}
}
