package main

import (
	"encoding/base64"
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
