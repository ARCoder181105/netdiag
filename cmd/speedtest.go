/*
Copyright © 2026 ARCoder181105 <EMAIL ADDRESS>
*/

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

var (
	serverID string
	noUpload bool
)

var speedtestCmd = &cobra.Command{
	Use:   "speedtest",
	Short: "Run internet speed test (Download/Upload)",
	Long: `Test your internet connection speed including download and upload speeds.

This command will automatically select the closest server and measure:
- Ping latency
- Download speed
- Upload speed (can be skipped with --no-upload)

Examples:
  netdiag speedtest
  netdiag speedtest --no-upload
  netdiag speedtest --server 12345`,
	Run: func(_ *cobra.Command, _ []string) {

		prober := &probe.SpeedTestProber{
			ServerID: serverID,
			NoUpload: noUpload,
		}

		output.PrintInfo("🌐 Running Internet Speed Test (this may take a moment)...")

		result, err := prober.Probe(context.Background())

		// ── Hard failure (returned as error, not Result) ──────────────────────
		if err != nil {
			result = probe.Result{
				Target:    "internet",
				ProbeType: "speedtest",
				Success:   false,
				Severity:  probe.SeverityError,
				Message:   err.Error(),
				TimeStamp: time.Now(),
			}
			logger.Log.Error("speedtest failed", "error", err)

			if jsonOutput {
				output.PrintJSON(result)
				return
			}
			output.PrintError(err.Error())
			return
		}
		// ─────────────────────────────────────────────────────────────────────

		// Log outcome
		if result.Success {
			logger.Log.Info("speedtest completed",
				"target", result.Target,
				"download_mbps", result.SpeedTestData.DownloadMbps,
				"upload_mbps", result.SpeedTestData.UploadMbps,
				"ping_ms", result.SpeedTestData.PingMs,
			)
		} else {
			logger.Log.Error("speedtest failed",
				"target", result.Target,
				"error", result.Message,
			)
		}

		// JSON mode
		if jsonOutput {
			output.PrintJSON(result)
			return
		}

		// Graceful failure (probe-level failure, not a Go error)
		if result.SpeedTestData == nil {
			output.PrintError(result.Message)
			return
		}

		data := result.SpeedTestData

		headers := []string{"Metric", "Value"}
		rows := [][]string{
			{"ISP", data.ISP},
			{"Public IP", data.PublicIP},
			{"Server", fmt.Sprintf("%s (%s)", data.ServerName, data.Country)},
			{"Sponsor", data.Sponsor},
			{"Distance", fmt.Sprintf("%.2f km", data.DistanceKm)},
			{"Ping", fmt.Sprintf("%.2f ms", data.PingMs)},
			{"Download", fmt.Sprintf("%.2f Mbps", data.DownloadMbps)},
		}

		if !noUpload {
			rows = append(rows, []string{
				"Upload", fmt.Sprintf("%.2f Mbps", data.UploadMbps),
			})
		}

		fmt.Println()
		output.PrintTable(headers, rows)
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
	rootCmd.AddCommand(speedtestCmd)

	speedtestCmd.Flags().BoolVarP(
		&noUpload,
		"no-upload",
		"u",
		false,
		"Skip upload test",
	)

	speedtestCmd.Flags().StringVarP(
		&serverID,
		"server",
		"s",
		"",
		"Specify server ID",
	)
}
