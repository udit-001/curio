package main

// Result is the uniform output type returned by every Source.
type Result struct {
	Source       string         `json:"source"`
	Title        string         `json:"title"`
	Creator      string         `json:"creator"`
	CreatorURL   string         `json:"creator_url"`
	License      string         `json:"license"`
	LicenseURL   string         `json:"license_url"`
	Attribution  string         `json:"attribution"`
	ImageURL     string         `json:"image_url"`
	ThumbnailURL string         `json:"thumbnail_url,omitempty"`
	LandingURL   string         `json:"landing_url"`
	Width        int            `json:"width"`
	Height       int            `json:"height"`
	Meta         map[string]any `json:"meta,omitempty"`
}

// Meta keys are normalized across all sources. Each source maps its
// API-specific fields to these shared keys so the agent has a consistent
// vocabulary regardless of which source returned the result.
//
// Standard meta keys:
//   tags        - []string  - subject keywords
//   description - string    - caption or description text
//   category    - string    - image type (photograph, illustration, diagram, etc.)
//   date        - string    - creation or publish date (ISO 8601)
//   location    - string    - where the image was taken or relates to

// Opts carries source-agnostic options that each Source interprets as it can.
type Opts struct {
	Width    int  // max width in pixels (for sources that support server-side resize)
	WantFull bool // request full-resolution original instead of a thumbnail
}

// Source is the single seam in the codebase. Every image source implements this
// interface and is registered in the sources map. Tests call Search() and verify
// results — no other seam is needed.
type Source interface {
	// Name returns the source identifier used in -s flag and output.
	Name() string

	// Description is a one-line summary of what makes this source unique.
	Description() string

	// Subjects are tags for what the source covers (e.g. "space", "art", "history").
	Subjects() []string

	// Licenses are the license types this source can return (e.g. "CC0", "CC-BY").
	Licenses() []string

	// Search queries the source and returns up to count results.
	// licenseTier is "free" (no attribution) or "any" (include CC-BY etc.).
	// opts carries width/full-res options; sources that don't support them ignore them.
	Search(query string, count int, licenseTier string, opts Opts) ([]Result, error)

	// NeedsKey returns true if the source requires an API key to function at all.
	// Key-optional sources (Openverse, Wikimedia) return false — they work keyless.
	NeedsKey() bool

	// KeyName returns the config key name for the required API key, or "" if keyless.
	KeyName() string
}

// sources is the registry. Adding a source = implement the interface + add one entry.
var sources = map[string]Source{}
