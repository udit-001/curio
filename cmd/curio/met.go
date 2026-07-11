package main

import (
	"fmt"
	"net/url"
)

// ---- Met Museum Source ----

type MetSource struct{}

func (s *MetSource) Name() string    { return "met" }
func (s *MetSource) NeedsKey() bool  { return false }
func (s *MetSource) KeyName() string { return "" }

func (s *MetSource) Description() string {
	return "Met Museum collection — art, paintings, sculptures, decorative arts. CC0 subset (public domain objects)"
}
func (s *MetSource) Subjects() []string {
	return []string{"art", "paintings", "sculptures", "historical artifacts", "decorative arts"}
}
func (s *MetSource) Licenses() []string {
	return []string{"CC0", "Restricted"}
}

func (s *MetSource) Search(query string, count int, licenseTier string, opts Opts) ([]Result, error) {
	searchURL := "https://collectionapi.metmuseum.org/public/collection/v1/search?" + url.Values{
		"q":         {query},
		"hasImages": {"true"},
	}.Encode()

	var searchData struct {
		Total     int   `json:"total"`
		ObjectIDs []int `json:"objectIDs"`
	}
	if err := httpGetJSON(searchURL, nil, &searchData); err != nil {
		return nil, fmt.Errorf("met: %w", err)
	}

	if len(searchData.ObjectIDs) == 0 {
		return nil, nil
	}

	maxFetch := count * 2
	if licenseTier == "any" {
		maxFetch = count
	}
	if maxFetch > len(searchData.ObjectIDs) {
		maxFetch = len(searchData.ObjectIDs)
	}

	type objResult struct {
		index int
		obj   *metObject
		err   error
	}

	results := make(chan objResult, maxFetch)
	for i := 0; i < maxFetch; i++ {
		go func(idx int, objectID int) {
			obj, err := s.fetchObject(objectID)
			results <- objResult{idx, obj, err}
		}(i, searchData.ObjectIDs[i])
	}

	objs := make([]*metObject, maxFetch)
	for i := 0; i < maxFetch; i++ {
		r := <-results
		if r.err != nil || r.obj == nil {
			objs[r.index] = nil
		} else {
			objs[r.index] = r.obj
		}
	}
	close(results)

	var out []Result
	for _, obj := range objs {
		if obj == nil {
			continue
		}
		if licenseTier == "free" && !obj.IsPublicDomain {
			continue
		}

		imgURL := obj.PrimaryImageSmall
		if opts.WantFull || imgURL == "" {
			imgURL = obj.PrimaryImage
		}
		if imgURL == "" {
			continue
		}

		license := "Restricted"
		if obj.IsPublicDomain {
			license = "CC0"
		}

		meta := map[string]any{}
		if obj.Medium != "" {
			meta["description"] = fmt.Sprintf("%s. %s", obj.Classification, obj.Medium)
		}
		if obj.ObjectDate != "" {
			meta["date"] = obj.ObjectDate
		}
		if obj.Department != "" {
			meta["tags"] = []string{obj.Department, obj.Classification}
		}
		if obj.Classification != "" {
			meta["category"] = obj.Classification
		}

		out = append(out, Result{
			Source:       "met",
			Title:        obj.Title,
			Creator:      obj.ArtistDisplayName,
			License:      license,
			LicenseURL:   obj.ObjectURL,
			Attribution:  fmt.Sprintf(`"%s" by %s — %s (Met Museum)`, obj.Title, orDefaultStr(obj.ArtistDisplayName, "unknown"), license),
			ImageURL:     imgURL,
			ThumbnailURL: obj.PrimaryImageSmall,
			LandingURL:   obj.ObjectURL,
			Meta:         meta,
		})
		if len(out) >= count {
			break
		}
	}
	return out, nil
}

type metObject struct {
	Title             string `json:"title"`
	ArtistDisplayName string `json:"artistDisplayName"`
	ArtistAlphaSort   string `json:"artistAlphaSort"`
	ArtistBeginDate   string `json:"artistBeginDate"`
	ArtistEndDate     string `json:"artistEndDate"`
	ObjectDate        string `json:"objectDate"`
	ObjectBeginDate   string `json:"objectBeginDate"`
	Classification    string `json:"classification"`
	Department        string `json:"department"`
	Medium            string `json:"medium"`
	Dimensions        string `json:"dimensions"`
	ObjectName        string `json:"objectName"`
	IsPublicDomain    bool   `json:"isPublicDomain"`
	PrimaryImage      string `json:"primaryImage"`
	PrimaryImageSmall string `json:"primaryImageSmall"`
	ObjectURL         string `json:"objectURL"`
	Tags              []struct {
		Term string `json:"term"`
	} `json:"tags"`
}

func (s *MetSource) fetchObject(objectID int) (*metObject, error) {
	detailURL := fmt.Sprintf("https://collectionapi.metmuseum.org/public/collection/v1/objects/%d", objectID)
	var obj metObject
	if err := httpGetJSON(detailURL, nil, &obj); err != nil {
		return nil, fmt.Errorf("met: %w", err)
	}
	return &obj, nil
}

func init() {
	sources["met"] = &MetSource{}
}
