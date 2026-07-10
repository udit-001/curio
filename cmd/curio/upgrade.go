package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const modulePath = "github.com/udit-001/curio/cmd/curio"

func runUpgrade(force, noSkills bool) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		fmt.Println()
		fmt.Println("  Go is not installed on your PATH.")
		fmt.Printf("  Install manually with:\n    go install %s@latest\n", modulePath)
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

	target := fmt.Sprintf("%s@%s", modulePath, latest)
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
	cmd := exec.Command(goPath, "list", "-m", "-versions", "github.com/udit-001/curio")
	output, err := cmd.Output()
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
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	skillDir := fmt.Sprintf("%s/.agents/skills/curio", home)
	skillFileData := map[string][]byte{}
	for _, name := range []string{"SKILL.md"} {
		data, err := skillFiles.ReadFile(name)
		if err != nil {
			return
		}
		skillFileData[name] = data
	}
	if skillChanged(skillDir, skillFileData) {
		fmt.Printf("  Skill files are outdated. Run 'curio skills install' to update.\n")
	} else {
		fmt.Printf("  Skill files are current.\n")
	}
}

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
