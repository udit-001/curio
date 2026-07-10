---
name: curio
description: Fetch a free-licensed image and drop it into the project. Use when the user wants a real photo, picture, artwork, or diagram found and added to the project ("find me an image of", "need a hero photo", "add a picture of", "fetch images for this page", "find a science diagram", "need an organism silhouette", "find a historical photo").
---

Drop a real, free-licensed image into the project. Downloads land in a scratch dir by default; you copy the winner into the project.

## The engine

```
SKILL_DIR/curio "QUERY" [opts]
```

`SKILL_DIR` is the folder this `SKILL.md` lives in — substitute the real absolute path.

Options:
- `-n N` — results (default 5)
- `-s` — source (run `curio sources` to see all 17 sources + availability)
- `-l` — license: `cc0,pd` (default, no attribution needed) | `any` (CC-BY etc.)
- `-d` — download to scratch (unique temp dir per run); without `-d`, just list
- `-o DIR` — output dir (overrides scratch — use to place directly into the project)
- `-w N` — max width px (sources that support server-side resize)
- `--full` — full-res original
- `--json` — machine-readable output (recommended for agents)
- `--quiet` — download mode: print only paths, no progress (for scripting)

Subcommands:
- `curio sources [--json]` — list all sources with subjects, licenses, and key status
- `curio setup` — interactive API key wizard (8 key-required sources)
- `curio skills install` — install skill files to detected agent directories
- `curio version` — print version info

## Picking a source

Run `curio sources --json` to see all sources, what they cover, and which are available. Sources marked `needs_key: true` with `key_configured: false` require setup — run `curio setup`. Default `openverse` (broadest); use `all` for maximum breadth.

**For agents:** use `--json` for machine-readable output when previewing results. When downloading with `-d`, read `attribution.json` in the scratch dir for structured metadata (filename, title, license, creator, dimensions).

## Workflow

1. **Parse intent** — subject, how many, where it goes in the project, and the source.
2. **Fetch to scratch** — `SKILL_DIR/curio "q" -s <source> -d --json`. Downloads the top N to a unique scratch dir and prints the path. Your structured source for picking is `scratch/attribution.json` (filename ↔ title ↔ license ↔ creator ↔ dimensions).
   - Done when: the scratch path holds the image files and `attribution.json`.
3. **Inspect** — read `scratch/attribution.json` to pick by title, license, or creator; or open the image files to judge visually. Choose the winner.
   - Done when: you've settled which file to place.
4. **Place the winner** — `cp` the chosen file from scratch into the target project path.
   - Done when: the chosen image exists at the project target path.
5. **Attribution** — if the placed image is non-CC0, grab the credit line from `scratch/attribution.json` and drop it near the image (a code comment, a `CREDITS` file). If the license is CC0 or public domain, attribution is optional. This step is nice-to-have — surface the license to the user and move on.

## License default

`-l cc0,pd` returns only no-attribution-needed images — safe to ship as-is. `-l any` widens to CC-BY etc.; the binary records the credit in `attribution.json` — surface it if easy.

## When results are thin

Broaden the query, switch source (run `curio sources` to see alternatives), or try `-l any`.

## Examples

```bash
# See available sources
SKILL_DIR/curio sources --json

# Preview without downloading (JSON output for agent parsing)
SKILL_DIR/curio "modern office" --json

# Fetch 5 to scratch, then copy the winner into the project
SKILL_DIR/curio "modern office" -d
cp /tmp/curio/.../02_open-plan-desk.jpg ./public/hero.jpg

# Wikipedia — curated image for any subject
SKILL_DIR/curio "crocodile" -s wikipedia -d

# NASA, place directly into the project (skip scratch)
SKILL_DIR/curio "mars surface" -s nasa -d -o ./public/hero

# Smithsonian (requires key — run setup first)
SKILL_DIR/curio "earhart" -s smithsonian -d

# PhyloPic, SVG silhouettes for biology diagrams
SKILL_DIR/curio "dinosauria" -s phylopic -d

# Wellcome, medical/scientific history
SKILL_DIR/curio "anatomy" -s wellcome -d

# Wikimedia, allow CC-BY, full-res
SKILL_DIR/curio "eiffel tower" -s wikimedia -l any --full -d
```
