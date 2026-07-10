package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Subcommands
	switch os.Args[1] {
	case "sources":
		asJSON := false
		for _, arg := range os.Args[2:] {
			if arg == "--json" {
				asJSON = true
			}
		}
		runSources(asJSON)
		return
	case "setup":
		runSetup()
		return
	case "install", "skills":
		// Handle: curio install, curio install --dir, curio skills install
		dir := ""
		project := false
		agentsOnly := false
		claudeOnly := false
		rest := os.Args[2:]
		// Skip "install" if called as "curio skills install"
		if len(os.Args) > 2 && os.Args[2] == "install" {
			rest = os.Args[3:]
		}
		for i := 0; i < len(rest); i++ {
			if rest[i] == "--dir" && i+1 < len(rest) {
				dir = rest[i+1]
				i++
			} else if rest[i] == "--project" {
				project = true
			} else if rest[i] == "--agents-only" {
				agentsOnly = true
			} else if rest[i] == "--claude-only" {
				claudeOnly = true
			}
		}
		if err := runInstall(dir, project, agentsOnly, claudeOnly); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	case "version", "--version", "-v":
		printVersion()
		return
	case "help", "--help", "-h":
		printUsage()
		return
	}

	// Search mode — parse flags
	args := os.Args[1:]
	query, opts, err := parseArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if query == "" {
		fmt.Fprintln(os.Stderr, "Error: query is required")
		os.Exit(1)
	}

	results, errs := search(query, opts)
	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "  ! source error — %s\n", e)
	}

	if len(results) == 0 && len(errs) > 0 {
		fmt.Fprintln(os.Stderr, "No results.")
		os.Exit(1)
	}

	if opts.download {
		if len(results) == 0 {
			fmt.Fprintln(os.Stderr, "No results to download.")
			os.Exit(1)
		}
		_, _, _ = download(results, opts.outDir, opts.quiet)
	} else {
		printResults(results, opts.json)
		if len(results) == 0 {
			os.Exit(1)
		}
	}
}

type searchOpts struct {
	count       int
	source      string
	licenseTier string
	width       int
	wantFull    bool
	download    bool
	outDir      string
	json        bool
	quiet       bool
}

func parseArgs(args []string) (string, searchOpts, error) {
	opts := searchOpts{
		count:       5,
		source:      "openverse",
		licenseTier: "free",
		outDir:      "", // empty = create unique temp dir per run
	}

	var query string
	var positional []string
	i := 0
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "-n":
			i++
			if i >= len(args) {
				return "", opts, fmt.Errorf("-n requires a value")
			}
			_, err := fmt.Sscanf(args[i], "%d", &opts.count)
			if err != nil {
				return "", opts, fmt.Errorf("-n: invalid number %q", args[i])
			}
		case "-s":
			i++
			if i >= len(args) {
				return "", opts, fmt.Errorf("-s requires a value")
			}
			opts.source = args[i]
		case "-l":
			i++
			if i >= len(args) {
				return "", opts, fmt.Errorf("-l requires a value")
			}
			opts.licenseTier = args[i]
		case "-w":
			i++
			if i >= len(args) {
				return "", opts, fmt.Errorf("-w requires a value")
			}
			_, err := fmt.Sscanf(args[i], "%d", &opts.width)
			if err != nil {
				return "", opts, fmt.Errorf("-w: invalid number %q", args[i])
			}
		case "--full":
			opts.wantFull = true
		case "-d":
			opts.download = true
		case "-o":
			i++
			if i >= len(args) {
				return "", opts, fmt.Errorf("-o requires a value")
			}
			opts.outDir = args[i]
		case "--json":
			opts.json = true
		case "--quiet":
			opts.quiet = true
		default:
			if !strings.HasPrefix(arg, "-") {
				positional = append(positional, arg)
			} else {
				return "", opts, fmt.Errorf("unknown flag: %s", arg)
			}
		}
		i++
	}

	if len(positional) > 0 {
		query = positional[0]
	}

	// Validate source
	if opts.source != "all" {
		if _, ok := sources[opts.source]; !ok {
			return "", opts, fmt.Errorf("unknown source: %s (available: %s)", opts.source, availableSources())
		}
	}

	return query, opts, nil
}

func search(query string, opts searchOpts) ([]Result, []string) {
	var results []Result
	var errors []string

	srcOpts := Opts{Width: opts.width, WantFull: opts.wantFull}

	runSource := func(name string) {
		src, ok := sources[name]
		if !ok {
			return
		}
		if src.NeedsKey() && configGet(src.KeyName()) == "" {
			if opts.source == "all" {
				errors = append(errors, fmt.Sprintf("%s: skipped (no API key — run 'curio setup')", name))
				return
			}
			errors = append(errors, fmt.Sprintf("%s: requires API key '%s' — run 'curio setup'", name, src.KeyName()))
			return
		}
		r, err := src.Search(query, opts.count, opts.licenseTier, srcOpts)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", name, err))
			return
		}
		results = append(results, r...)
	}

	if opts.source == "all" {
		for name := range sources {
			runSource(name)
		}
	} else {
		runSource(opts.source)
	}

	// Dedup by ImageURL
	seen := map[string]bool{}
	var deduped []Result
	for _, r := range results {
		if r.ImageURL != "" && !seen[r.ImageURL] {
			seen[r.ImageURL] = true
			deduped = append(deduped, r)
		}
	}
	return deduped, errors
}

func printResults(results []Result, asJSON bool) {
	if asJSON {
		data, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(data))
		return
	}
	if len(results) == 0 {
		fmt.Println("No results.")
		return
	}
	for i, r := range results {
		fmt.Printf("\n[%d] %s\n", i+1, r.Title)
		fmt.Printf("    source:    %s\n", r.Source)
		fmt.Printf("    creator:   %s\n", orDefaultStr(r.Creator, "unknown"))
		fmt.Printf("    license:   %s\n", r.License)
		if r.LicenseURL != "" {
			fmt.Printf("    lic url:   %s\n", r.LicenseURL)
		}
		fmt.Printf("    image url: %s\n", r.ImageURL)
	}
}

func availableSources() string {
	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

func printUsage() {
	fmt.Println(`curio — search & download free-licensed images

Usage:
  curio "QUERY" [options]
  curio sources [--json]
  curio setup
  curio skills install [--dir DIR] [--project] [--agents-only] [--claude-only]
  curio version

Options:
  -n N          results (default 5)
  -s SOURCE     source: openverse, nasa, wikimedia, smithsonian, met, loc, wellcome, phylopic, europeana, wikipedia, pexels, pixabay, unsplash, bhl, archive, gbif, va, all
  -l LICENSE    license: free (default, no attribution) | any
  -w N          max width px
  --full        full-res original
  -d            download to scratch
  -o DIR        output dir (default: unique temp dir per run)
  --json        machine-readable output (recommended for agents)
  --quiet       download mode: print only paths, no progress (for scripting)`)
}

func printVersion() {
	fmt.Printf("curio %s", version)
	if commit != "" {
		fmt.Printf(" (commit: %s, built: %s)", commit, date)
	}
	fmt.Println()
}
