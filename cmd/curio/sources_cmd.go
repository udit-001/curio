package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
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
	}
}

func availableSources() string {
	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}
