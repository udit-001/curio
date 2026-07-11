package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ---- Agent detection ----

var agentProviders = []string{"opencode", "codex", "pi.dev", "claude-code"}

func hasBinary(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func hasDir(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func detectAgents() []string {
	var detected []string
	home, err := os.UserHomeDir()
	if err != nil {
		return detected
	}

	for _, name := range agentProviders {
		found := hasBinary(name)
		if !found && hasDir(filepath.Join(home, ".agents")) {
			found = true
		}
		if found {
			detected = append(detected, name)
		}
	}
	return detected
}

// ---- Install families ----

type installFamily struct {
	name    string
	subdir  string
	readers []string
}

var families = []installFamily{
	{
		name:    "agents",
		subdir:  ".agents/skills/curio",
		readers: []string{"opencode", "codex", "pi.dev"},
	},
	{
		name:    "claude",
		subdir:  ".claude/skills/curio",
		readers: []string{"claude-code"},
	},
}

func familyGlobalDir(f installFamily) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return f.subdir
	}
	return filepath.Join(home, f.subdir)
}

func isSkillInstalled(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "SKILL.md"))
	return err == nil
}

// ---- Skill install ----

func installSkillFiles(skillDir string) error {
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("create skill dir: %w", err)
	}

	files := map[string][]byte{}
	for _, name := range []string{"SKILL.md"} {
		data, err := skillFiles.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", name, err)
		}
		files[name] = data
		if err := os.WriteFile(filepath.Join(skillDir, name), data, 0644); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}

	return writeManifest(skillDir, files)
}

func runInstall(dir string, project bool, agentsOnly, claudeOnly bool) error {
	// Read embedded skill files for hash comparison
	skillFileData := map[string][]byte{}
	for _, name := range []string{"SKILL.md"} {
		data, err := skillFiles.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", name, err)
		}
		skillFileData[name] = data
	}

	// Explicit dir — install there directly
	if dir != "" {
		changed := skillChanged(dir, skillFileData)
		if !changed {
			fmt.Printf("  %s✓%s Skill files already current at %s\n", termGreen(), termReset(), dir)
			return nil
		}
		if err := installSkillFiles(dir); err != nil {
			return err
		}
		fmt.Printf("  %s✓%s Skill files written to %s\n", termGreen(), termReset(), dir)
		return nil
	}

	// Non-interactive flags
	if agentsOnly || claudeOnly {
		var selected []installFamily
		if agentsOnly {
			selected = []installFamily{families[0]}
		} else {
			selected = []installFamily{families[1]}
		}
		return installFamilies(selected, project, skillFileData)
	}

	// Auto-detect agents
	detected := detectAgents()
	if len(detected) == 0 {
		fmt.Printf("  %s⚠ No AI agents detected.%s Install manually:\n", termYellow(), termReset())
		fmt.Printf("    curio skills install --dir ~/.agents/skills/curio/\n")
		return nil
	}

	fmt.Printf("  Detected: %s\n", strings.Join(detected, ", "))

	// Determine available families based on detected agents
	avail := availableFamilies(detected)
	if len(avail) == 0 {
		fmt.Println("  No installable families for detected agents.")
		return nil
	}

	// Interactive selection if more than one family
	var selected []installFamily
	if len(avail) <= 1 {
		selected = avail
	} else {
		selected = promptFamilySelect(avail)
		if selected == nil {
			fmt.Println("  Cancelled.")
			return nil
		}
	}

	// Prompt scope (global vs project)
	if !project {
		var cancelled bool
		project, cancelled = promptInstallScope(selected)
		if cancelled {
			fmt.Println("  Cancelled.")
			return nil
		}
	}

	return installFamilies(selected, project, skillFileData)
}

func availableFamilies(detected []string) []installFamily {
	var avail []installFamily
	for _, f := range families {
		for _, reader := range f.readers {
			for _, d := range detected {
				if d == reader {
					avail = append(avail, f)
					break
				}
			}
		}
	}
	return avail
}

func promptFamilySelect(avail []installFamily) []installFamily {
	fmt.Println()
	fmt.Println("  Install to:")
	fmt.Printf("    1. Both           — %s, %s\n", familyGlobalDir(families[0]), familyGlobalDir(families[1]))
	fmt.Printf("    2. Standard only  — %s  (%s)\n", familyGlobalDir(families[0]), strings.Join(families[0].readers, ", "))
	fmt.Printf("    3. Claude only    — %s  (%s)\n", familyGlobalDir(families[1]), strings.Join(families[1].readers, ", "))
	fmt.Println("    0. Cancel")
	fmt.Println()
	input := ask("Enter number [1]")

	switch input {
	case "0":
		return nil
	case "2":
		return []installFamily{families[0]}
	case "3":
		return []installFamily{families[1]}
	default:
		return avail
	}
}

func promptInstallScope(selected []installFamily) (bool, bool) {
	globalDirs := make([]string, len(selected))
	projectDirs := make([]string, len(selected))
	for i, f := range selected {
		globalDirs[i] = familyGlobalDir(f)
		projectDirs[i] = "./" + f.subdir
	}
	fmt.Println()
	fmt.Println("  Scope:")
	fmt.Printf("    1. Globally     — %s\n", strings.Join(globalDirs, ", "))
	fmt.Printf("    2. This project — %s\n", strings.Join(projectDirs, ", "))
	fmt.Println("    0. Cancel")
	fmt.Println()
	input := ask("Enter number [1]")

	switch input {
	case "0":
		return false, true
	case "2":
		return true, false
	default:
		return false, false
	}
}

func installFamilies(selected []installFamily, project bool, skillFileData map[string][]byte) error {
	for _, f := range families {
		if !familyInList(f, selected) {
			continue
		}
		var dir string
		if project {
			dir = f.subdir
		} else {
			dir = familyGlobalDir(f)
		}
		changed := skillChanged(dir, skillFileData)
		if !changed {
			fmt.Printf("  %s✓%s %s (already current) — %s\n", termGreen(), termReset(), dir, strings.Join(f.readers, ", "))
			continue
		}
		if err := installSkillFiles(dir); err != nil {
			fmt.Fprintf(os.Stderr, "  ! failed to write to %s: %v\n", dir, err)
		} else {
			fmt.Printf("  %s✓%s %s — %s\n", termGreen(), termReset(), dir, strings.Join(f.readers, ", "))
		}
	}
	fmt.Println()
	return nil
}

func familyInList(f installFamily, list []installFamily) bool {
	for _, x := range list {
		if x.name == f.name {
			return true
		}
	}
	return false
}
