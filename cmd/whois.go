/*
Copyright © 2026 ARCoder181105 <EMAIL ADDRESS>
*/

// Package cmd implements the CLI commands.
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ARCoder181105/netdiag/pkg/logger"
	"github.com/ARCoder181105/netdiag/pkg/output"
	"github.com/ARCoder181105/netdiag/pkg/probe"
)

var whoisCmd = &cobra.Command{
	Use:   "whois <domain>",
	Short: "Retrieve domain registration information",
	Long: `Query the WHOIS database to find information about a domain name,
including the registrar, creation date, and expiration date.`,
	Args: cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {

		prober := &probe.WhoisProber{
			Domain: args[0],
		}

		output.PrintInfo(fmt.Sprintf("Querying WHOIS for %s...", args[0]))

		result, err := prober.Probe(context.Background())
		if err != nil {
			logger.Log.Error("whois failed", "target", args[0], "error", err)
			output.PrintError(err.Error())
			return
		}

		// ── Structured logging ────────────────────────────────────────────────
		if result.Success {
			logger.Log.Info("whois completed",
				"target", result.Target,
				"latency_ms", result.Latency.Milliseconds(),
			)
		} else {
			logger.Log.Error("whois failed",
				"target", result.Target,
				"error", result.Message,
			)
		}
		// ─────────────────────────────────────────────────────────────────────

		if jsonOutput {
			output.PrintJSON(result)
			return
		}

		if !result.Success || result.WhoisData == nil {
			output.PrintError(result.Message)
			return
		}

		fmt.Println()
		fmt.Println(result.WhoisData.Raw)
		fmt.Println()

		switch result.Severity {
		case probe.SeverityOK:
			output.PrintSuccess(result.Message)
		case probe.SeverityWarning:
			output.PrintWarning(result.Message)
		case probe.SeverityError:
			output.PrintError(result.Message)
		default:
			output.PrintInfo(result.Message)
		}
	},
}

func init() {
	rootCmd.AddCommand(whoisCmd)
}
