package main

import (
	"encoding/json"
	"fmt"
	"sort"
)

// ---- Sources command ----

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
		fmt.Printf("    subjects: %s\n", joinTags(info.Subjects))
		fmt.Printf("    licenses: %s\n", joinTags(info.Licenses))
	}
}

func joinTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	result := tags[0]
	for i := 1; i < len(tags); i++ {
		result += ", " + tags[i]
	}
	return result
}
