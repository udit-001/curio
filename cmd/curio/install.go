package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// detectAgents returns the names of detected agent providers.
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

// installTargets returns the directories to write skill files to.
// Universal standard: .agents/skills (all agents) + .claude/skills (Claude-specific).
func installTargets(detected []string, agentsOnly, claudeOnly bool) []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	var targets []string
	if !claudeOnly {
		targets = append(targets, filepath.Join(home, ".agents/skills/curio"))
	}
	if !agentsOnly {
		for _, d := range detected {
			if d == "claude-code" {
				targets = append(targets, filepath.Join(home, ".claude/skills/curio"))
				break
			}
		}
	}
	return targets
}

// ---- Skill install ----

// installSkillFiles writes embedded SKILL.md and SOURCES.md to the target dir,
// plus a manifest for change detection.
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

// runInstall is the `curio skills install` command.
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

	// Auto-detect agents
	detected := detectAgents()
	if len(detected) == 0 {
		fmt.Printf("  %s⚠ No AI agents detected.%s Install manually:\n", termYellow(), termReset())
		fmt.Printf("    curio skills install --dir ~/.agents/skills/curio/\n")
		return nil
	}

	fmt.Printf("  Detected agents: %v\n", detected)

	targets := installTargets(detected, agentsOnly, claudeOnly)
	for _, target := range targets {
		changed := skillChanged(target, skillFileData)
		if changed {
			if err := installSkillFiles(target); err != nil {
				fmt.Fprintf(os.Stderr, "  ! failed to write to %s: %v\n", target, err)
			} else {
				fmt.Printf("  %s✓%s %s\n", termGreen(), termReset(), target)
			}
		} else {
			fmt.Printf("  %s✓%s %s (already current)\n", termGreen(), termReset(), target)
		}
	}

	if project {
		cwd, _ := os.Getwd()
		projectDir := filepath.Join(cwd, ".agents/skills/curio")
		if err := installSkillFiles(projectDir); err != nil {
			fmt.Fprintf(os.Stderr, "  ! failed to write to %s: %v\n", projectDir, err)
		} else {
			fmt.Printf("  %s✓%s %s (project-level)\n", termGreen(), termReset(), projectDir)
		}
	}

	return nil
}
