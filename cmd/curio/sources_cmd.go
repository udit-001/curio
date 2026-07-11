package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// ---- Sources command ----

var sourcesCmd = &cobra.Command{
	Use:   "sources",
	Short: "List all image sources",
	Long:  "List all 17 image sources with description and availability.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		asJSON, _ := cmd.Flags().GetBool("json")
		runSources(asJSON)
	},
}

func init() {
	sourcesCmd.Flags().Bool("json", false, "machine-readable output")
	rootCmd.AddCommand(sourcesCmd)
}

type sourceInfo struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Subjects      []string `json:"subjects"`
	Licenses      []string `json:"licenses"`
	NeedsKey      bool     `json:"needs_key"`
	KeyConfigured bool     `json:"key_configured"`
}

func runSources(asJSON bool) {
	var infos []sourceInfo
	for name, src := range sources {
		info := sourceInfo{
			Name:        name,
			Description: src.Description(),
			Subjects:    src.Subjects(),
			Licenses:    src.Licenses(),
			NeedsKey:    src.NeedsKey(),
		}
		if info.NeedsKey {
			info.KeyConfigured = configGet(src.KeyName()) != ""
		}
		infos = append(infos, info)
	}

	// Available first (alphabetical), then unavailable (alphabetical)
	sort.Slice(infos, func(i, j int) bool {
		iAvail := !infos[i].NeedsKey || infos[i].KeyConfigured
		jAvail := !infos[j].NeedsKey || infos[j].KeyConfigured
		if iAvail != jAvail {
			return iAvail
		}
		return infos[i].Name < infos[j].Name
	})

	if asJSON {
		data, _ := json.MarshalIndent(infos, "", "  ")
		fmt.Println(string(data))
		return
	}

	for _, info := range infos {
		mark := termGreen() + "✓" + termReset()
		if info.NeedsKey && !info.KeyConfigured {
			mark = termYellow() + "✗" + termReset()
		}
		fmt.Printf("  %s %-14s %s\n", mark, info.Name, info.Description)
	}
}

// availableKeylessSources returns a comma-separated list of available
// keyless sources, excluding the given source name.
func availableKeylessSources(exclude string) string {
	var names []string
	for name, src := range sources {
		if name == exclude {
			continue
		}
		if !src.NeedsKey() || configGet(src.KeyName()) != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
