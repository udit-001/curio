package main

import (
	"encoding/base64"
	"regexp"
	"strconv"
	"strings"
)

// Shared helpers used across multiple source files.

// ---- Command suggestion ----

var knownCommands = []string{"search", "sources", "setup", "skills", "upgrade", "version", "help"}

func suggestCommand(input string) string {
	best := ""
	bestDist := 3
	for _, cmd := range knownCommands {
		dist := editDistance(input, cmd)
		if dist < bestDist {
			best = cmd
			bestDist = dist
		}
	}
	return best
}

func editDistance(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

// ---- Semver ----

func semverCompare(a, b string) int {
	pa := parseSemver(a)
	pb := parseSemver(b)
	min := len(pa)
	if len(pb) < min {
		min = len(pb)
	}
	for i := 0; i < min; i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	if len(pa) < len(pb) {
		return -1
	}
	if len(pa) > len(pb) {
		return 1
	}
	return 0
}

func parseSemver(v string) []int {
	if idx := strings.IndexAny(v, "-+"); idx >= 0 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	nums := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nums
		}
		nums = append(nums, n)
	}
	return nums
}

// ---- String utilities ----

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

func stripHTML(s string) string {
	if s == "" {
		return ""
	}
	return strings.TrimSpace(htmlTagRe.ReplaceAllString(s, ""))
}

func needsCredit(licenseStr string) bool {
	upper := strings.ToUpper(licenseStr)
	noAttrib := strings.Contains(upper, "CC0") ||
		strings.Contains(upper, "PDM") ||
		strings.Contains(upper, "PUBLIC DOMAIN") ||
		upper == "PD"
	return !noAttrib
}

// isCC0orPD returns true if the license is CC0 or Public Domain.
// Checks both the license name and the license URL for public domain markers.
func isCC0orPD(license, licenseURL string) bool {
	if !needsCredit(license) {
		return true
	}
	lower := strings.ToLower(licenseURL)
	return strings.Contains(lower, "publicdomain")
}

func orDefaultStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func licenseFromURL(licenseURL string) string {
	lower := strings.ToLower(licenseURL)
	if strings.Contains(lower, "publicdomain/zero") {
		return "CC0"
	}
	if strings.Contains(lower, "publicdomain/mark") {
		return "Public Domain Mark"
	}
	if strings.Contains(lower, "licenses/by/") {
		return "CC-BY"
	}
	if strings.Contains(lower, "licenses/by-sa/") {
		return "CC-BY-SA"
	}
	if strings.Contains(lower, "licenses/by-nc/") {
		return "CC-BY-NC"
	}
	return "See license"
}
