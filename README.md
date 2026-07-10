# curio

Search 17 free-licensed image sources from the command line. Designed as a CLI skill for AI agents.

## Install

```bash
go install github.com/udit-001/curio@latest
```

Or build from source:

```bash
git clone https://github.com/udit-001/curio.git
cd curio && make build
```

## Usage

```bash
curio "cats" -n 3              # search (list results)
curio "cats" -n 3 -d           # download to scratch dir
curio "mars surface" -s nasa -d
curio "modern office" --json   # machine-readable output
curio sources --json           # list all sources + key status
curio setup                    # interactive API key wizard
```

### Options

| Flag | Description |
|------|-------------|
| `-n N` | results (default 5) |
| `-s SOURCE` | source name or `all` |
| `-l LICENSE` | `cc0,pd` (default) \| `any` |
| `-d` | download to scratch dir |
| `-o DIR` | output dir (overrides scratch) |
| `-w N` | max width px |
| `--full` | full-res original |
| `--json` | machine-readable output |
| `--quiet` | download mode: paths only, no progress |

## Sources

17 sources — 11 keyless (Openverse, NASA, Wikimedia, Met, LoC, Wellcome, PhyloPic, Archive.org, GBIF, V&A, Wikipedia), 6 key-required (Smithsonian, Europeana, Pexels, Pixabay, Unsplash, BHL). Run `curio sources` for the full list.

Key-required sources need `curio setup` to configure API keys. Config is TOML at OS-standard dirs (`~/.config/curio/config.toml` on Linux).

## For AI agents

```bash
curio skills install    # writes SKILL.md to detected agent directories
```

Downloads create a unique temp dir per run with an `attribution.json` sidecar containing full metadata (title, creator, license, dimensions). Use `--quiet` to get just the paths on stdout.

## Development

```bash
make build
make check
```

See [AGENTS.md](AGENTS.md) for architecture and adding new sources. Requires Go 1.22+.
