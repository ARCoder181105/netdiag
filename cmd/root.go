/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version information (set by build flags)
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// Variables to store flag values
var jsonOutput bool
var logFilePath string
var showVersion bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "netdiag",
	Short: "Network diagnostics and monitoring CLI tool",
	Long: `netdiag is a developer-friendly CLI tool used for
network diagnostics, monitoring, and debugging.`,
	Run: func(cmd *cobra.Command, args []string) {
		// If --version flag is set, show version and exit
		if showVersion {
			fmt.Printf("netdiag version %s\n", Version)
			fmt.Printf("  commit: %s\n", Commit)
			fmt.Printf("  built:  %s\n", Date)
			return
		}
		// If no subcommands are provided, show help
		cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags (PersistentFlags) apply to this command and all subcommands
	rootCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "Output JSON format")
	rootCmd.PersistentFlags().StringVarP(&logFilePath, "log-file", "l", "", "Path to the log file")

	// Local flags (Flags) apply only to this command
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")
}
