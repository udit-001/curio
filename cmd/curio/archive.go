package main

import (
	"fmt"
	"net/url"
	"strings"
)

// ---- Internet Archive Source ----

type ArchiveSource struct{}

func (s *ArchiveSource) Name() string    { return "archive" }
func (s *ArchiveSource) NeedsKey() bool  { return false }
func (s *ArchiveSource) KeyName() string { return "" }

func (s *ArchiveSource) Description() string {
	return "Internet Archive — historical book scans, engravings, old photographs, manuscripts. Massive scope"
}
func (s *ArchiveSource) Subjects() []string {
	return []string{"history", "book scans", "engravings", "photographs", "manuscripts"}
}
func (s *ArchiveSource) Licenses() []string {
	return []string{"Public Domain", "Various"}
}

func (s *ArchiveSource) Search(query string, count int, licenseTier string, opts Opts) ([]Result, error) {
	searchURL := "https://archive.org/advancedsearch.php?q=" + url.QueryEscape(query+" AND mediatype:image") + "&fl[]=identifier&fl[]=title&fl[]=date&fl[]=mediatype&fl[]=creator&rows=" + fmt.Sprintf("%d", count) + "&output=json"

	var data struct {
		Response struct {
			Docs []struct {
				Identifier string      `json:"identifier"`
				Title      interface{} `json:"title"`
				Creator    interface{} `json:"creator"`
				Date       interface{} `json:"date"`
			} `json:"docs"`
			NumFound int `json:"numFound"`
		} `json:"response"`
	}
	if err := httpGetJSON(searchURL, nil, &data); err != nil {
		return nil, fmt.Errorf("archive: %w", err)
	}

	var out []Result
	for _, doc := range data.Response.Docs {
		title := toString(doc.Title)
		creator := toString(doc.Creator)
		dateStr := toString(doc.Date)

		imgURL := fmt.Sprintf("https://archive.org/services/img/%s", doc.Identifier)
		if opts.WantFull {
			imgURL = fmt.Sprintf("https://archive.org/download/%s", doc.Identifier)
		}

		landingURL := fmt.Sprintf("https://archive.org/details/%s", doc.Identifier)

		thumbnailURL := fmt.Sprintf("https://archive.org/services/img/%s?width=200", doc.Identifier)

		meta := map[string]any{}
		if dateStr != "" {
			meta["date"] = dateStr
		}
		meta["category"] = "historical document"

		out = append(out, Result{
			Source:       "archive",
			Title:        title,
			Creator:      creator,
			License:      "Public domain (Internet Archive)",
			LicenseURL:   "https://archive.org/about/terms.php",
			Attribution:  fmt.Sprintf(`"%s" — Internet Archive`, title),
			ImageURL:     imgURL,
			ThumbnailURL: thumbnailURL,
			LandingURL:   landingURL,
			Meta:         meta,
		})
		if len(out) >= count {
			break
		}
	}
	return out, nil
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case []interface{}:
		if len(val) > 0 {
			if s, ok := val[0].(string); ok {
				return s
			}
		}
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

func init() {
	sources["archive"] = &ArchiveSource{}
}
