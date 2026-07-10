# AGENTS.md — curio

A Go binary that searches free-licensed image sources and downloads results. Designed as a CLI skill — an AI agent calls `curio "query" -s source` to find images for a project.

## Build & verify

```bash
make build
make check
```

## Architecture

The `Source` interface in `source.go` is the single seam. Every source implements it and registers itself in the `sources` map via `init()`. The dispatch in `main.go` looks up the source by name and calls `Search()`.

```go
type Source interface {
    Name() string
    Description() string
    Subjects() []string
    Licenses() []string
    Search(query string, count int, licenseTier string, opts Opts) ([]Result, error)
    NeedsKey() bool
    KeyName() string
}
```

Key files: `source.go` (interface + registry), `config.go` (TOML config read/write at OS-standard dirs), `http.go` (retry/backoff + `httpGetJSON` + `httpPostForm`), `download.go` (download + attribution.json), `main.go` (CLI parsing, dispatch, list/download modes), `embed.go` (`//go:embed` of SKILL.md), `terminal.go` (terminal color, prompt, browser helpers), `setup.go` (interactive key wizard), `install.go` (agent detection + `curio skills install`), `manifest.go` (SHA256 change detection), `sources_cmd.go` (`curio sources` command with metadata), `helpers.go` (shared utilities: `orDefaultStr`, `stripHTML`, `needsCredit`, `licenseFromURL`, `isCC0orPD`, `base64Encode`).

## Adding a source

1. **Create `{source}.go`** — implement the `Source` interface (see any existing source file for the pattern). Register in `init()`: `sources["name"] = &MySource{}`. The `Description()`, `Subjects()`, and `Licenses()` methods carry metadata that `curio sources` exposes — no separate file to update.
   - Done when: the file compiles and `curio "test" -s name` returns results.
2. **Handle keys** — if the source needs a key, return `true` from `NeedsKey()` and the key name from `KeyName()`. The dispatch auto-skips key-required sources in `-s all` and hard-fails with a setup hint for explicit selection.
   - Done when: `-s name` without key gives a clear error; `-s all` without key skips with a stderr note.
3. **Add to setup wizard** — if key-required, add a stage in `setup.go` that opens the signup URL, prompts for the key, and tests it immediately.
   - Done when: `curio setup` includes the new source's stage.
4. **Rebuild and test** — `make build && ./bin/curio "query" -s name -n 2`
   - Done when: results return, download works, license filter works.

## Config

TOML at OS-standard config directory (`~/.config/curio/` on Linux, `~/Library/Application Support/curio/` on macOS, `%AppData%\curio\` on Windows). See `config.go` for the loading logic. `curio setup` writes keys interactively with live testing. Run `curio sources` to see which sources need keys.

## CLI

Run `curio --help` for the full CLI reference. Key commands: `curio "query" [opts]`, `curio sources [--json]`, `curio setup`, `curio skills install`, `curio version`.

## Sources

17 sources implemented. Run `curio sources` (or `curio sources --json`) for the full list with descriptions, subjects, licenses, and key status.

## Dependencies

- `github.com/pelletier/go-toml/v2` (compile-time only)
- Go 1.22+

## Issue tracker

Project IMGF on Lific. PRD: IMGF-DOC-1.
