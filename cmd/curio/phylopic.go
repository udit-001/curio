package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// ---- PhyloPic Source ----

type phylopicImage struct {
	UUID        string             `json:"uuid"`
	Attribution string             `json:"attribution"`
	Links       phylopicImageLinks `json:"_links"`
}

type phylopicImageLinks struct {
	Self struct {
		Href  string `json:"href"`
		Title string `json:"title"`
	} `json:"self"`
	License struct {
		Href string `json:"href"`
	} `json:"license"`
	VectorFile *struct {
		Href  string `json:"href"`
		Sizes string `json:"sizes"`
		Type  string `json:"type"`
	} `json:"vectorFile"`
	RasterFiles []struct {
		Href  string `json:"href"`
		Sizes string `json:"sizes"`
		Type  string `json:"type"`
	} `json:"rasterFiles"`
}

type phylopicName struct {
	Class string `json:"class"`
	Text  string `json:"text"`
}

type PhyloPicSource struct {
	build   string
	buildMu sync.Mutex
}

func (s *PhyloPicSource) Name() string    { return "phylopic" }
func (s *PhyloPicSource) NeedsKey() bool  { return false }
func (s *PhyloPicSource) KeyName() string { return "" }

func (s *PhyloPicSource) Description() string {
	return "Organism silhouettes as scalable SVG — animals, plants, microbes. Perfect for phylogenetic trees and biology diagrams"
}
func (s *PhyloPicSource) Subjects() []string {
	return []string{"biology", "organisms", "silhouettes", "evolution", "phylogenetics", "svg"}
}
func (s *PhyloPicSource) Licenses() []string {
	return []string{"CC0", "Public Domain Mark", "CC-BY", "CC-BY-SA"}
}

var phylopicNameRe = regexp.MustCompile(`[^a-z\s]`)

func (s *PhyloPicSource) Search(query string, count int, licenseTier string, opts Opts) ([]Result, error) {
	build, err := s.getBuild()
	if err != nil {
		return nil, fmt.Errorf("phylopic: %w", err)
	}

	cleanName := phylopicNameRe.ReplaceAllString(strings.ToLower(query), "")
	cleanName = strings.TrimSpace(cleanName)

	searchURL := "https://api.phylopic.org/nodes?" + url.Values{
		"filter_name":        {cleanName},
		"build":              {build},
		"page":               {"0"},
		"embed_items":        {"true"},
		"embed_primaryImage": {"true"},
	}.Encode()

	headers := map[string]string{
		"Accept": "application/vnd.phylopic.v2+json",
	}

	var data struct {
		Embedded struct {
			Items []struct {
				UUID     string           `json:"uuid"`
				Names    [][]phylopicName `json:"names"`
				Embedded struct {
					PrimaryImage *phylopicImage `json:"primaryImage"`
				} `json:"_embedded"`
			} `json:"items"`
		} `json:"_embedded"`
		TotalItems int `json:"totalItems"`
	}
	if err := httpGetJSON(searchURL, headers, &data); err != nil {
		return nil, fmt.Errorf("phylopic: %w", err)
	}

	var out []Result
	for _, item := range data.Embedded.Items {
		img := item.Embedded.PrimaryImage
		if img == nil {
			continue
		}

		licenseURL := img.Links.License.Href
		if licenseTier == "cc0,pd" && !strings.Contains(licenseURL, "publicdomain") {
			continue
		}

		imgURL := s.pickImageURL(img, opts)
		if imgURL == "" {
			continue
		}

		title := img.Links.Self.Title
		if title == "" {
			title = nodeName(item.Names)
		}

		license := licenseFromURL(licenseURL)

		out = append(out, Result{
			Source:      "phylopic",
			Title:       title,
			Creator:     img.Attribution,
			License:     license,
			LicenseURL:  licenseURL,
			Attribution: fmt.Sprintf(`"%s" by %s — %s (PhyloPic)`, title, orDefaultStr(img.Attribution, "unknown"), license),
			ImageURL:    imgURL,
			LandingURL:  fmt.Sprintf("https://www.phylopic.org/images/%s", img.UUID),
		})
		if len(out) >= count {
			break
		}
	}
	return out, nil
}

func (s *PhyloPicSource) pickImageURL(img *phylopicImage, opts Opts) string {
	if opts.Width > 0 {
		for _, rf := range img.Links.RasterFiles {
			w, _ := strconv.Atoi(strings.SplitN(rf.Sizes, "x", 2)[0])
			if w <= opts.Width {
				return rf.Href
			}
		}
		if len(img.Links.RasterFiles) > 0 {
			return img.Links.RasterFiles[len(img.Links.RasterFiles)-1].Href
		}
	}

	if img.Links.VectorFile != nil {
		return img.Links.VectorFile.Href
	}
	if len(img.Links.RasterFiles) > 0 {
		return img.Links.RasterFiles[0].Href
	}
	return ""
}

func (s *PhyloPicSource) getBuild() (string, error) {
	s.buildMu.Lock()
	defer s.buildMu.Unlock()

	if s.build != "" {
		return s.build, nil
	}

	var data struct {
		Build int `json:"build"`
	}
	if err := httpGetJSON("https://api.phylopic.org/", map[string]string{
		"Accept": "application/vnd.phylopic.v2+json",
	}, &data); err != nil {
		return "", err
	}

	s.build = strconv.Itoa(data.Build)
	return s.build, nil
}

func nodeName(names [][]phylopicName) string {
	for _, nameGroup := range names {
		for _, n := range nameGroup {
			if n.Class == "scientific" || n.Class == "vernacular" {
				return n.Text
			}
		}
	}
	return "unknown"
}

func init() {
	sources["phylopic"] = &PhyloPicSource{}
}
