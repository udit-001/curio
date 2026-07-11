package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

var rootCmd = &cobra.Command{
	Use:   "curio",
	Short: "Search & download free-licensed images",
	Long: `Curio searches 17 free-licensed image sources and downloads results.
Designed as a CLI skill for AI agents.

Run 'curio <command> --help' for command-specific help.`,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printVersion() {
	fmt.Printf("curio %s", version)
	if commit != "" {
		fmt.Printf(" (commit: %s, built: %s)", commit, date)
	}
	fmt.Println()
}
