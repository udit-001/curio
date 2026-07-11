package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const modulePath = "github.com/udit-001/curio"
const cmdPath = modulePath + "/cmd/curio"

func runUpgrade(force, noSkills bool) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		fmt.Println()
		fmt.Println("  Go is not installed on your PATH.")
		fmt.Printf("  Install manually with:\n    go install %s@latest\n", cmdPath)
		fmt.Println()
		return
	}

	fmt.Println()
	fmt.Printf("  Checking for upgrades...\n")
	fmt.Printf("  Current version: %s\n", version)

	latest, err := latestVersionFromProxy(goPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ! %v\n", err)
		return
	}
	fmt.Printf("  Latest version:  %s\n", latest)

	current := strings.TrimPrefix(version, "v")
	latestClean := strings.TrimPrefix(latest, "v")
	if !force && current != "" && current != "dev" && semverCompare(current, latestClean) >= 0 {
		fmt.Printf("  Already up to date (%s)\n", version)
		fmt.Println()
		return
	}

	target := fmt.Sprintf("%s@%s", cmdPath, latest)
	fmt.Printf("  Running: go install %s\n", target)

	c := exec.Command(goPath, "install", target)
	output, err := c.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ! go install failed: %v\n%s", err, string(output))
		return
	}

	fmt.Printf("  Upgraded to %s\n", latest)

	if !noSkills {
		offerSkillUpgrade()
	}

	fmt.Println()
}

func latestVersionFromProxy(goPath string) (string, error) {
	c := exec.Command(goPath, "list", "-m", "-versions", modulePath)
	output, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("check for versions failed: %w", err)
	}
	parts := strings.Fields(string(output))
	if len(parts) < 2 {
		return "", fmt.Errorf("no versions found — push a git tag first")
	}
	return parts[len(parts)-1], nil
}

func offerSkillUpgrade() {
	skillFileData := readEmbeddedSkillFiles()
	if skillFileData == nil {
		return
	}
	for _, f := range families {
		dir := familyGlobalDir(f)
		if !isSkillInstalled(dir) {
			continue
		}
		if skillChanged(dir, skillFileData) {
			fmt.Printf("  Skill files outdated at %s. Run 'curio skills install' to update.\n", dir)
		}
	}
}
