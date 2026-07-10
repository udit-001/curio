package main

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
)

// ---- Wellcome Collection Source ----

type WellcomeSource struct{}

func (s *WellcomeSource) Name() string    { return "wellcome" }
func (s *WellcomeSource) NeedsKey() bool  { return false }
func (s *WellcomeSource) KeyName() string { return "" }

func (s *WellcomeSource) Description() string {
	return "Wellcome Collection — medical history, scientific history, anatomy, natural history illustrations"
}
func (s *WellcomeSource) Subjects() []string {
	return []string{"medical history", "scientific history", "anatomy", "natural history", "illustrations"}
}
func (s *WellcomeSource) Licenses() []string {
	return []string{"Public Domain Mark", "CC0", "CC-BY", "CC-BY-NC"}
}

func (s *WellcomeSource) Search(query string, count int, licenseTier string, opts Opts) ([]Result, error) {
	if licenseTier == "cc0,pd" {
		return s.searchDualLicense(query, count, opts)
	}
	return s.searchSingle(query, count, "", opts)
}

func (s *WellcomeSource) searchDualLicense(query string, count int, opts Opts) ([]Result, error) {
	var pdmResults, cc0Results []Result
	var pdmErr, cc0Err error
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		pdmResults, pdmErr = s.searchSingle(query, count, "pdm", opts)
	}()
	go func() {
		defer wg.Done()
		cc0Results, cc0Err = s.searchSingle(query, count, "cc-0", opts)
	}()
	wg.Wait()

	var out []Result
	out = append(out, pdmResults...)
	out = append(out, cc0Results...)
	if len(out) > count {
		out = out[:count]
	}

	if pdmErr != nil && cc0Err != nil {
		return nil, fmt.Errorf("wellcome: both license queries failed: %w; %v", pdmErr, cc0Err)
	}
	return out, nil
}

func (s *WellcomeSource) searchSingle(query string, count int, license string, opts Opts) ([]Result, error) {
	params := url.Values{
		"query":    {query},
		"pageSize": {fmt.Sprintf("%d", count)},
	}
	if license != "" {
		params.Set("locations.license", license)
	}

	searchURL := "https://api.wellcomecollection.org/catalogue/v2/images?" + params.Encode()

	var data struct {
		Results []struct {
			ID     string `json:"id"`
			Source struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"source"`
			Locations []struct {
				License struct {
					ID    string `json:"id"`
					Label string `json:"label"`
					URL   string `json:"url"`
				} `json:"license"`
				Credit string `json:"credit"`
			} `json:"locations"`
			Thumbnail struct {
				URL string `json:"url"`
			} `json:"thumbnail"`
		} `json:"results"`
	}
	if err := httpGetJSON(searchURL, nil, &data); err != nil {
		return nil, fmt.Errorf("wellcome: %w", err)
	}

	var out []Result
	for _, r := range data.Results {
		if len(r.Locations) == 0 || r.Thumbnail.URL == "" {
			continue
		}

		loc := r.Locations[0]
		imgURL := s.iiifURL(r.Thumbnail.URL, opts)
		if imgURL == "" {
			continue
		}

		licenseLabel := s.licenseLabel(loc.License.ID, loc.License.Label)

		out = append(out, Result{
			Source:      "wellcome",
			Title:       r.Source.Title,
			Creator:     loc.Credit,
			License:     licenseLabel,
			LicenseURL:  loc.License.URL,
			Attribution: fmt.Sprintf(`"%s" — %s (%s)`, r.Source.Title, licenseLabel, orDefaultStr(loc.Credit, "Wellcome Collection")),
			ImageURL:    imgURL,
			LandingURL:  fmt.Sprintf("https://wellcomecollection.org/works/%s", r.Source.ID),
		})
		if len(out) >= count {
			break
		}
	}
	return out, nil
}

func (s *WellcomeSource) iiifURL(infoJSONURL string, opts Opts) string {
	base := strings.TrimSuffix(infoJSONURL, "/info.json")
	if base == infoJSONURL {
		return ""
	}

	if opts.WantFull {
		return base + "/full/full/0/default.jpg"
	}
	if opts.Width > 0 {
		return fmt.Sprintf("%s/full/%d,/0/default.jpg", base, opts.Width)
	}
	return base + "/full/!1280,1280/0/default.jpg"
}

func (s *WellcomeSource) licenseLabel(id, label string) string {
	if label != "" {
		return label
	}
	switch id {
	case "cc-0":
		return "CC0"
	case "pdm":
		return "Public Domain Mark"
	case "cc-by":
		return "CC-BY 4.0"
	case "cc-by-sa":
		return "CC-BY-SA 4.0"
	case "cc-by-nc":
		return "CC-BY-NC 4.0"
	case "cc-by-nd":
		return "CC-BY-ND 4.0"
	case "cc-by-nc-sa":
		return "CC-BY-NC-SA 4.0"
	case "cc-by-nc-nd":
		return "CC-BY-NC-ND 4.0"
	case "inc":
		return "In copyright"
	default:
		return id
	}
}

func init() {
	sources["wellcome"] = &WellcomeSource{}
}
