/*
Copyright © 2026 ARCoder181105 <EMAIL ADDRESS>
*/

// Package cmd implements the CLI commands.
package cmd

import (
	"fmt"
	"os"

	"github.com/ARCoder181105/netdiag/pkg/config"
	"github.com/ARCoder181105/netdiag/pkg/logger"
	"github.com/spf13/cobra"
)

// Variables to store flag values
var (
	jsonOutput  bool
	logFilePath string
	logFormat   string
	showVersion bool
)

// Version info variables
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// SetVersionInfo sets the version information from main
func SetVersionInfo(v, c, d string) {
	version = v
	commit = c
	date = d
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "netdiag",
	Short: "Network diagnostics and monitoring CLI tool",
	Long: `netdiag is a developer-friendly CLI tool used for
network diagnostics, monitoring, and debugging.`,
	// Wire Viper and Logger into PersistentPreRun
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := logger.Init(logFilePath, logFormat); err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}
		if err := config.Load(); err != nil {
			logger.Log.Warn("Failed to load config file, using defaults", "error", err)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, _ []string) {
		if showVersion {
			fmt.Printf("netdiag version %s (%s) built on %s\n", version, commit, date)
			return
		}
		// If no subcommands are provided, show help
		// Fix errcheck
		_ = cmd.Help()
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
	rootCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "Output JSON format")
	rootCmd.PersistentFlags().StringVarP(&logFilePath, "log-file", "l", "", "Path to the log file")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "Log format (text or json)")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")
}
