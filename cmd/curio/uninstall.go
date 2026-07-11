package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func runUninstall(all bool) {
	// Check all family locations
	var installed []string
	for _, f := range families {
		dir := familyGlobalDir(f)
		if isSkillInstalled(dir) {
			installed = append(installed, dir)
		}
	}

	// Also check project-level
	cwd, _ := os.Getwd()
	for _, f := range families {
		dir := filepath.Join(cwd, f.subdir)
		if isSkillInstalled(dir) {
			installed = append(installed, dir)
		}
	}

	if len(installed) == 0 {
		fmt.Println()
		fmt.Println("  No curio skill installs found.")
		fmt.Println()
		return
	}

	fmt.Println()
	fmt.Printf("  Found %d skill install(s):\n\n", len(installed))
	for i, dir := range installed {
		scope := "global"
		if strings.HasPrefix(dir, cwd) {
			scope = "project"
		}
		fmt.Printf("    %d. %s — %s\n", i+1, dir, scope)
	}

	if all {
		fmt.Printf("\n  Remove all %d? [y/N] ", len(installed))
		if !confirm("") {
			fmt.Println("  Cancelled.")
			return
		}
		for _, dir := range installed {
			removeSkillDir(dir)
		}
		fmt.Println()
		return
	}

	fmt.Print("\n  Remove which? (comma-separated numbers, 'all', or 0 to cancel)\n  > ")
	input := ask("")

	if input == "" || input == "0" {
		fmt.Println("  Cancelled.")
		return
	}
	if input == "all" {
		for _, dir := range installed {
			removeSkillDir(dir)
		}
		fmt.Println()
		return
	}

	for _, part := range strings.Split(input, ",") {
		part = strings.TrimSpace(part)
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 || n > len(installed) {
			fmt.Printf("  Ignoring invalid input: %s\n", part)
			continue
		}
		removeSkillDir(installed[n-1])
	}
	fmt.Println()
}

func removeSkillDir(dir string) {
	// Read manifest for file list, fall back to embedded file list
	var files []string
	manifest, err := readManifest(dir)
	if err == nil {
		files = manifest.Files
	} else {
		files = []string{"SKILL.md"}
	}

	for _, f := range files {
		p := filepath.Join(dir, f)
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "  ! could not delete %s: %v\n", p, err)
		}
	}

	// Remove manifest + skill dir
	os.Remove(filepath.Join(dir, manifestFileName))
	os.Remove(dir)

	// Clean up empty parent dirs (skills/, curio/)
	for parent := filepath.Dir(dir); parent != filepath.Dir(filepath.Dir(dir)); parent = filepath.Dir(parent) {
		entries, err := os.ReadDir(parent)
		if err != nil || len(entries) > 0 {
			break
		}
		os.Remove(parent)
	}

	fmt.Printf("  %s✓%s Removed %s\n", termGreen(), termReset(), dir)
}
