---
name: curio
description: Fetch a free-licensed image and drop it into the project. Use when the user wants a real photo, picture, artwork, or diagram found and added to the project ("find me an image of", "need a hero photo", "add a picture of", "fetch images for this page", "find a science diagram", "need an organism silhouette", "find a historical photo").
---

Drop a real, free-licensed image into the project. Downloads land in a scratch dir by default; you copy the winner into the project.

## The engine

```bash
# Search and download (the main thing)
curio "cats" -s openverse -d

# Preview without downloading
curio "cats" -s openverse --json

# See available sources
curio sources --json
```

Most defaults are fine. Override when needed:

- `-n N` — more results (default 5)
- `-l any` — allow attribution licenses (default `free` = no attribution needed)
- `-o DIR` — place directly into the project, skip scratch
- `--quiet` — print only paths, no progress (for scripting)

Run `curio --help` for all flags (`-w`, `--full`, subcommands).

## Picking a source

Run `curio sources --json` to see all sources, what they cover, and which are available. Sources marked `needs_key: true` with `key_configured: false` require setup — run `curio setup`. Default `openverse` (broadest); use `all` for maximum breadth.

When downloading with `-d`, read `attribution.json` in the scratch dir for structured metadata (filename, title, license, creator, dimensions).

## Workflow

1. **Parse intent** — subject, how many, where it goes in the project, and the source.
   - Done when: you have a query string, a count, a target path, and a source (or `all`).
2. **Fetch to scratch** — `curio "q" -s <source> -d --json`. Downloads the top N to a unique scratch dir and prints the path. Your structured source for picking is `scratch/attribution.json` (filename ↔ title ↔ license ↔ creator ↔ dimensions).
   - Done when: the scratch path holds the image files and `attribution.json`.
3. **Inspect** — read `scratch/attribution.json` to pick by title, license, or creator; or open the image files to judge visually. Choose the winner.
   - Done when: you've settled which file to place.
4. **Place the winner** — `cp` the chosen file from scratch into the target project path.
   - Done when: the chosen image exists at the project target path.
5. **Attribution** — if the placed image is non-CC0, grab the credit line from `scratch/attribution.json` and drop it near the image (a code comment, a `CREDITS` file). If the license is CC0 or public domain, attribution is optional.
   - Done when: the license is surfaced to the user (stated or dropped near the image).

## License default

`-l free` returns only no-attribution-needed images — safe to ship as-is. `-l any` widens to CC-BY etc.; the binary records the credit in `attribution.json` — surface it if easy.

## When results are thin

Broaden the query, switch source (run `curio sources` to see alternatives), or try `-l any`.

## Examples

```bash
# See available sources
curio sources --json

# Preview without downloading (JSON output for agent parsing)
curio "modern office" --json

# Fetch 5 to scratch, then copy the winner into the project
curio "modern office" -d
cp /tmp/curio/.../02_open-plan-desk.jpg ./public/hero.jpg

# Wikipedia — curated image for any subject
curio "crocodile" -s wikipedia -d

# NASA, place directly into the project (skip scratch)
curio "mars surface" -s nasa -d -o ./public/hero

# Smithsonian (requires key — run setup first)
curio "earhart" -s smithsonian -d

# PhyloPic, SVG silhouettes for biology diagrams
curio "dinosauria" -s phylopic -d

# Wellcome, medical/scientific history
curio "anatomy" -s wellcome -d

# Wikimedia, allow CC-BY, full-res
curio "eiffel tower" -s wikimedia -l any --full -d
```
