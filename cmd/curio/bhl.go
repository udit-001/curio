package main

import (
	"fmt"
	"net/url"
)

// ---- Biodiversity Heritage Library Source ----

type BhlSource struct{}

func (s *BhlSource) Name() string    { return "bhl" }
func (s *BhlSource) NeedsKey() bool  { return true }
func (s *BhlSource) KeyName() string { return "bhl.api_key" }

func (s *BhlSource) Description() string {
	return "Vintage scientific illustrations from natural history books — botanical plates, zoological drawings, anatomical illustrations"
}
func (s *BhlSource) Subjects() []string {
	return []string{"biology", "botany", "zoology", "anatomy", "scientific illustrations", "natural history"}
}
func (s *BhlSource) Licenses() []string {
	return []string{"Public Domain", "CC0"}
}

func (s *BhlSource) Search(query string, count int, licenseTier string, opts Opts) ([]Result, error) {
	key := configGet(s.KeyName())

	searchURL := "https://www.biodiversitylibrary.org/api3?" + url.Values{
		"action": {"BookSearch"},
		"query":  {query},
		"format": {"json"},
		"apikey": {key},
		"count":  {fmt.Sprintf("%d", count)},
	}.Encode()

	var data struct {
		Result []struct {
			TitleID         int    `json:"TitleID"`
			ShortTitle      string `json:"ShortTitle"`
			PublicationDate string `json:"PublicationDate"`
			Authors         []struct {
				Name string `json:"Name"`
			} `json:"Authors"`
			Items []struct {
				ItemID int `json:"ItemID"`
			} `json:"Items"`
		} `json:"Result"`
	}
	if err := httpGetJSON(searchURL, nil, &data); err != nil {
		return nil, fmt.Errorf("bhl: %w", err)
	}

	var out []Result
	for _, book := range data.Result {
		if len(book.Items) == 0 {
			continue
		}
		itemID := book.Items[0].ItemID

		pageURL := "https://www.biodiversitylibrary.org/api3?" + url.Values{
			"action": {"GetPageMetadata"},
			"pages":  {fmt.Sprintf("%d", itemID)},
			"format": {"json"},
			"apikey": {key},
		}.Encode()

		var pageData struct {
			Result []struct {
				PageID       int    `json:"PageID"`
				PageURL      string `json:"PageURL"`
				ThumbnailURL string `json:"ThumbnailURL"`
				FullImageURL string `json:"FullImageURL"`
			} `json:"Result"`
		}
		if err := httpGetJSON(pageURL, nil, &pageData); err != nil {
			continue
		}

		if len(pageData.Result) == 0 {
			continue
		}
		page := pageData.Result[0]

		imgURL := page.FullImageURL
		if !opts.WantFull && page.ThumbnailURL != "" {
			imgURL = page.ThumbnailURL
		}
		if imgURL == "" {
			continue
		}

		creator := ""
		if len(book.Authors) > 0 {
			creator = book.Authors[0].Name
		}

		out = append(out, Result{
			Source:      "bhl",
			Title:       book.ShortTitle,
			Creator:     creator,
			License:     "Public domain (BHL)",
			LicenseURL:  "https://creativecommons.org/publicdomain/mark/1.0/",
			Attribution: fmt.Sprintf(`"%s" (%s) — Public domain (BHL)`, book.ShortTitle, book.PublicationDate),
			ImageURL:    imgURL,
			LandingURL:  page.PageURL,
		})
		if len(out) >= count {
			break
		}
	}
	return out, nil
}

func init() {
	sources["bhl"] = &BhlSource{}
}
