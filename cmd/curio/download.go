package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// manifestEntry embeds Result so the manifest can't drift from the Result struct.
// Adding a field to Result automatically appears in attribution.json.
type manifestEntry struct {
	Result
	Filename string `json:"filename"`
	Bytes    int    `json:"bytes"`
}

// download fetches all result images to outDir and writes attribution.json.
// If outDir is empty, creates a unique temp dir so parallel calls don't clobber each other.
// When quiet is true, suppresses progress output and prints only the dir and manifest paths.
// Returns the manifest and the actual output directory used.
func download(results []Result, outDir string, quiet bool) (manifest []map[string]any, actualDir string, err error) {
	if outDir == "" {
		base := filepath.Join(os.TempDir(), "curio")
		_ = os.MkdirAll(base, 0755)
		tmp, err := os.MkdirTemp(base, "")
		if err != nil {
			return nil, "", fmt.Errorf("create temp dir: %w", err)
		}
		outDir = tmp
	} else {
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return nil, "", fmt.Errorf("create output dir: %w", err)
		}
	}

	if !quiet {
		fmt.Printf("Downloading %d image(s) to %s/\n", len(results), outDir)
	}

	var entries []manifestEntry
	for i, r := range results {
		if r.ImageURL == "" {
			continue
		}
		fname, size, ok := saveImage(r, outDir, i+1)
		if !ok {
			continue
		}
		entries = append(entries, manifestEntry{Result: r, Filename: fname, Bytes: size})

		if !quiet {
			fmt.Printf("  + %s  (%d KB)  [%s]\n", fname, size/1024, r.License)
			if needsCredit(r.License) {
				fmt.Println("    attribution required — see attribution.json")
			}
		}
	}

	manifestPath := filepath.Join(outDir, "attribution.json")
	mdata, _ := json.MarshalIndent(entries, "", "  ")
	_ = os.WriteFile(manifestPath, mdata, 0644)

	if quiet {
		fmt.Println(outDir)
		fmt.Println(manifestPath)
	} else {
		fmt.Printf("\nSCRATCH: %s\n", outDir)
		fmt.Printf("attribution: %s\n", manifestPath)
	}

	// Convert to []map[string]any for the return value (callers may expect this type)
	for _, e := range entries {
		raw, _ := json.Marshal(e)
		var m map[string]any
		json.Unmarshal(raw, &m)
		manifest = append(manifest, m)
	}
	return manifest, outDir, nil
}

// saveImage fetches a single result's image and writes it to dir.
// Returns the filename, byte count, and success flag.
func saveImage(r Result, dir string, idx int) (filename string, size int, ok bool) {
	resp, err := httpGet(r.ImageURL, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ! download failed %s: %v\n", r.ImageURL, err)
		return "", 0, false
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ! read failed %s: %v\n", r.ImageURL, err)
		return "", 0, false
	}

	ext := extFor(r.ImageURL, resp.Header.Get("Content-Type"))
	filename = fmt.Sprintf("%02d_%s.%s", idx, slugify(r.Title), ext)
	outPath := filepath.Join(dir, filename)
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "  ! write failed %s: %v\n", outPath, err)
		return "", 0, false
	}
	return filename, len(data), true
}

var extRe = regexp.MustCompile(`\.(jpe?g|png|gif|webp|svg|tiff?)$`)

func extFor(urlStr string, contentType string) string {
	if contentType != "" {
		ct := strings.Split(contentType, ";")[0]
		switch strings.ToLower(strings.TrimSpace(ct)) {
		case "image/jpeg":
			return "jpg"
		case "image/png":
			return "png"
		case "image/gif":
			return "gif"
		case "image/webp":
			return "webp"
		case "image/svg+xml":
			return "svg"
		case "image/tiff":
			return "tif"
		}
	}
	lower := strings.ToLower(urlStr)
	if m := extRe.FindStringSubmatch(lower); m != nil {
		ext := m[1]
		if ext == "jpeg" {
			return "jpg"
		}
		if ext == "tiff" {
			return "tif"
		}
		return ext
	}
	return "jpg"
}

var slugRe = regexp.MustCompile(`[^\w\s-]`)
var slugRe2 = regexp.MustCompile(`[\s_-]+`)

func slugify(s string) string {
	s = stripHTML(s)
	if s == "" {
		return "image"
	}
	s = extRe.ReplaceAllString(s, "")
	s = slugRe.ReplaceAllString(s, "")
	s = strings.TrimSpace(strings.ToLower(s))
	s = slugRe2.ReplaceAllString(s, "-")
	if len(s) > 40 {
		s = s[:40]
	}
	s = strings.Trim(s, "-")
	if s == "" {
		return "image"
	}
	return s
}
