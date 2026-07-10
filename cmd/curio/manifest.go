package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ---- Skill manifest for change detection ----

const manifestFileName = "curio.skill.json"

type skillManifest struct {
	Hash  string   `json:"hash"`
	Files []string `json:"files"`
}

// computeManifestHash calculates a SHA-256 hash over sorted file paths + contents.
func computeManifestHash(files map[string][]byte) string {
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	h := sha256.New()
	for _, p := range paths {
		h.Write([]byte(p))
		h.Write([]byte{0})
		h.Write(files[p])
		h.Write([]byte{0})
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// writeManifest writes the skill manifest to the skill directory.
func writeManifest(skillDir string, files map[string][]byte) error {
	manifest := skillManifest{
		Hash:  computeManifestHash(files),
		Files: make([]string, 0, len(files)),
	}
	for p := range files {
		manifest.Files = append(manifest.Files, p)
	}
	sort.Strings(manifest.Files)

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(skillDir, manifestFileName), data, 0644)
}

// readManifest reads the skill manifest from the skill directory.
func readManifest(skillDir string) (*skillManifest, error) {
	data, err := os.ReadFile(filepath.Join(skillDir, manifestFileName))
	if err != nil {
		return nil, err
	}
	var m skillManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// skillChanged returns true if the manifest hash doesn't match or any file is missing.
func skillChanged(skillDir string, files map[string][]byte) bool {
	manifest, err := readManifest(skillDir)
	if err != nil {
		return true
	}
	if manifest.Hash != computeManifestHash(files) {
		return true
	}
	for _, f := range manifest.Files {
		if _, err := os.Stat(filepath.Join(skillDir, f)); os.IsNotExist(err) {
			return true
		}
	}
	return false
}
