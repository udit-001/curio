package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var searchOpts_ struct {
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

var searchCmd = &cobra.Command{
	Use:   "search \"QUERY\"",
	Short: "Search free-licensed image sources",
	Long: `Search 17 image sources for free-licensed images.

Examples:
  curio search "cats" -s openverse -n 3
  curio search "cats" -d --json
  curio search "mars surface" -s nasa -d -o ./public/hero`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		opts := searchOpts{
			count:       searchOpts_.count,
			source:      searchOpts_.source,
			licenseTier: searchOpts_.licenseTier,
			width:       searchOpts_.width,
			wantFull:    searchOpts_.wantFull,
			download:    searchOpts_.download,
			outDir:      searchOpts_.outDir,
			json:        searchOpts_.json,
			quiet:       searchOpts_.quiet,
		}

		results, errs := search(query, opts)

		// In JSON mode, errors go to stderr as-is
		if opts.json {
			for _, e := range errs {
				fmt.Fprintf(cmd.ErrOrStderr(), "  ! %s\n", e)
			}
		} else {
			// Print errors
			for _, e := range errs {
				fmt.Fprintf(cmd.ErrOrStderr(), "  ! %s\n", e)
			}
		}

		// No results
		if len(results) == 0 {
			if opts.source != "all" && len(errs) > 0 {
				// Explicit source failed — suggest available alternatives
				fmt.Fprintf(cmd.ErrOrStderr(), "\n  Available keyless sources: ")
				fmt.Fprintln(cmd.ErrOrStderr(), availableKeylessSources(opts.source))
				fmt.Fprintf(cmd.ErrOrStderr(), "  Ask the user to run 'curio setup' to configure %s.\n", opts.source)
			}
			os.Exit(1)
		}

		if opts.download {
			_, _, _ = download(results, opts.outDir, opts.quiet)
		} else {
			printResults(results, opts.json)
		}
		return nil
	},
}

func init() {
	searchCmd.Flags().IntVarP(&searchOpts_.count, "count", "n", 5, "number of results")
	searchCmd.Flags().StringVarP(&searchOpts_.source, "source", "s", "openverse", "source name or 'all'")
	searchCmd.Flags().StringVarP(&searchOpts_.licenseTier, "license", "l", "free", "free (no attribution) | any")
	searchCmd.Flags().IntVarP(&searchOpts_.width, "width", "w", 0, "max width px")
	searchCmd.Flags().BoolVar(&searchOpts_.wantFull, "full", false, "full-res original")
	searchCmd.Flags().BoolVarP(&searchOpts_.download, "download", "d", false, "download to scratch dir")
	searchCmd.Flags().StringVarP(&searchOpts_.outDir, "output", "o", "", "output dir (overrides scratch)")
	searchCmd.Flags().BoolVar(&searchOpts_.json, "json", false, "machine-readable output")
	searchCmd.Flags().BoolVar(&searchOpts_.quiet, "quiet", false, "download mode: paths only, no progress")
	rootCmd.AddCommand(searchCmd)
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
				errors = append(errors, fmt.Sprintf("%s: skipped (unavailable)", name))
			} else {
				errors = append(errors, fmt.Sprintf("%s is unavailable (API key not configured)", name))
			}
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
