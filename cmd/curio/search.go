package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"

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
	srcOpts := Opts{Width: opts.width, WantFull: opts.wantFull}

	// Determine which sources to query
	var sourceNames []string
	if opts.source == "all" {
		for name := range sources {
			sourceNames = append(sourceNames, name)
		}
		sort.Strings(sourceNames)
	} else {
		sourceNames = []string{opts.source}
	}

	type sourceResult struct {
		results []Result
		errMsg  string
	}

	// Query sources concurrently with a concurrency limit
	results := make([][]Result, len(sourceNames))
	errs := make([]string, len(sourceNames))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // max 5 concurrent source queries

	for i, name := range sourceNames {
		wg.Add(1)
		go func(idx int, srcName string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			src, ok := sources[srcName]
			if !ok {
				return
			}
			if src.NeedsKey() && configGet(src.KeyName()) == "" {
				if opts.source == "all" {
					errs[idx] = fmt.Sprintf("%s: skipped (unavailable)", srcName)
				} else {
					errs[idx] = fmt.Sprintf("%s is unavailable (API key not configured)", srcName)
				}
				return
			}
			r, err := src.Search(query, opts.count, opts.licenseTier, srcOpts)
			if err != nil {
				errs[idx] = fmt.Sprintf("%s: %v", srcName, err)
				return
			}
			results[idx] = r
		}(i, name)
	}
	wg.Wait()

	// Flatten results in source order (stable)
	var allResults []Result
	var allErrors []string
	for i := range sourceNames {
		if errs[i] != "" {
			allErrors = append(allErrors, errs[i])
		}
		allResults = append(allResults, results[i]...)
	}

	// Dedup by ImageURL
	seen := map[string]bool{}
	var deduped []Result
	for _, r := range allResults {
		if r.ImageURL != "" && !seen[r.ImageURL] {
			seen[r.ImageURL] = true
			deduped = append(deduped, r)
		}
	}
	return deduped, allErrors
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
