package main

import "embed"

//go:embed SKILL.md
var skillFiles embed.FS

// embeddedSkillFiles is the list of files embedded in the binary.
var embeddedSkillFiles = []string{"SKILL.md"}

// readEmbeddedSkillFiles returns the embedded skill file data.
func readEmbeddedSkillFiles() map[string][]byte {
	files := map[string][]byte{}
	for _, name := range embeddedSkillFiles {
		data, err := skillFiles.ReadFile(name)
		if err != nil {
			return nil
		}
		files[name] = data
	}
	return files
}
