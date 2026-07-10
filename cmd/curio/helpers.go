package main

import (
	"regexp"
	"strings"
)

// Shared helpers used across multiple source files.

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

func orDefaultStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
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
