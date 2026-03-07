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
	ports       string
	scanTimeout int
	concurrency int
)

var scanCmd = &cobra.Command{
	Use:   "scan <host>",
	Short: "Scan for open TCP ports",
	Long: `Scan a target host for open TCP ports using a high-concurrency worker pool.
You can specify a single port, a list, or a range.

Examples:
  netdiag scan google.com
  netdiag scan 192.168.1.1 --ports 80,443,8000-8100
  netdiag scan localhost -p 22 -t 2`,
	Args: cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		host := args[0]

		portList := probe.ParsePortRange(ports)
		if len(portList) == 0 {
			output.PrintError("No valid ports parsed. Please check your --ports flag.")
			return
		}

		scanner := &probe.ConnectScanner{
			Host:        host,
			Ports:       portList,
			Timeout:     time.Duration(scanTimeout) * time.Second,
			Concurrency: concurrency,
		}

		result, err := scanner.Probe(context.Background())

		if err != nil {
			result = probe.Result{
				Target:    host,
				ProbeType: "scan",
				Success:   false,
				Severity:  probe.SeverityError,
				Message:   err.Error(),
				TimeStamp: time.Now(),
			}
		}

		// ── Structured logging ────────────────────────────────────────────────
		if result.Success && result.ScanData != nil {
			logger.Log.Info("scan completed",
				"target", result.Target,
				"total_ports", result.ScanData.TotalPorts,
				"open_ports", result.ScanData.OpenPorts,
				"scan_method", result.ScanData.ScanMethod,
			)
		} else {
			logger.Log.Error("scan failed",
				"target", result.Target,
				"error", result.Message,
			)
		}
		// ─────────────────────────────────────────────────────────────────────

		if jsonOutput {
			output.PrintJSON(result)
			return
		}

		if result.ScanData == nil || len(result.ScanData.OpenPorts) == 0 {
			output.PrintWarning(result.Message)
			return
		}

		headers := []string{"Port", "Protocol", "Status"}
		var rows [][]string

		for _, p := range result.ScanData.OpenPorts {
			rows = append(rows, []string{
				fmt.Sprintf("%d", p),
				"TCP",
				"Open",
			})
		}

		fmt.Println()
		output.PrintTable(headers, rows)

		output.PrintInfo(fmt.Sprintf(
			"Scanned %d ports (%d ports/ms) using %s method.",
			result.ScanData.TotalPorts,
			result.ScanData.ScanRateMs,
			result.ScanData.ScanMethod,
		))

		output.PrintSuccess(result.Message)
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().IntVarP(&scanTimeout, "timeout", "t", 1, "Timeout in seconds")
	scanCmd.Flags().StringVarP(&ports, "ports", "p", "1-1024", "The range to scan")
	scanCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 100, "Number of concurrent ports to scan")
}
