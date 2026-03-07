// Package cmd implements the CLI commands.
package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/ARCoder181105/netdiag/pkg/logger"
	"github.com/ARCoder181105/netdiag/pkg/output"
	"github.com/ARCoder181105/netdiag/pkg/probe"
)

var discoverTimeout int

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Scan local network for devices",
	Run: func(_ *cobra.Command, _ []string) {

		prober := &probe.DiscoverProber{
			Timeout: time.Duration(discoverTimeout) * time.Millisecond,
		}

		result, err := prober.Probe(context.Background())

		if err != nil {
			result = probe.Result{
				ProbeType: "discover",
				Success:   false,
				Severity:  probe.SeverityError,
				Message:   err.Error(),
				TimeStamp: time.Now(),
			}
		}

		// ── Structured logging ────────────────────────────────────────────────
		if result.Success && result.DiscoverData != nil {
			logger.Log.Info("network discovery completed",
				"prefix", result.DiscoverData.Prefix,
				"devices_found", len(result.DiscoverData.Devices),
				"latency_ms", result.Latency.Milliseconds(),
			)
		} else {
			logger.Log.Error("network discovery failed",
				"error", result.Message,
			)
		}
		// ─────────────────────────────────────────────────────────────────────

		if jsonOutput {
			output.PrintJSON(result)
			return
		}

		output.PrintInfo("🔍 Scanning local network (this may take a moment)...")

		if !result.Success || result.DiscoverData == nil {
			output.PrintError(result.Message)
			return
		}

		data := result.DiscoverData

		headers := []string{"IP Address", "Hostname", "Latency"}
		var rows [][]string

		for _, dev := range data.Devices {
			rows = append(rows, []string{
				dev.IP,
				dev.HostName,
				dev.Latency.String(),
			})
		}

		fmt.Println()
		output.PrintTable(headers, rows)

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
	rootCmd.AddCommand(discoverCmd)
	discoverCmd.Flags().IntVarP(&discoverTimeout, "timeout", "t", 500, "Ping timeout in milliseconds")
}
