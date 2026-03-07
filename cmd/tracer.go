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
	maxHops      int
	traceTimeout time.Duration
)

var traceCmd = &cobra.Command{
	Use:   "trace [host]",
	Short: "Perform a traceroute to a destination host",
	Long: `Trace the network path to a destination host by sending ICMP packets
with increasing TTL values. Shows each hop (router) along the path.

Example:
  netdiag trace google.com
  netdiag trace 8.8.8.8`,
	Args: cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {

		prober := &probe.TraceProber{
			Host:    args[0],
			MaxHops: maxHops,
			Timeout: traceTimeout,
		}

		result, err := prober.Probe(context.Background())
		if err != nil {
			logger.Log.Error("trace failed", "target", args[0], "error", err)
			output.PrintError(err.Error())
			return
		}

		// ── Structured logging ────────────────────────────────────────────────
		if result.Success && result.TraceData != nil {
			logger.Log.Info("trace completed",
				"target", result.Target,
				"hops", len(result.TraceData.Hops),
				"latency_ms", result.Latency.Milliseconds(),
			)
		} else {
			logger.Log.Error("trace failed",
				"target", result.Target,
				"error", result.Message,
			)
		}
		// ─────────────────────────────────────────────────────────────────────

		if jsonOutput {
			output.PrintJSON(result)
			return
		}

		if !result.Success || result.TraceData == nil {
			output.PrintError(result.Message)
			return
		}

		headers := []string{"Hop", "IP Address", "Hostname", "RTT (ms)"}
		var rows [][]string

		for _, hop := range result.TraceData.Hops {
			rttMs := "*"
			if !hop.Timeout {
				rttMs = fmt.Sprintf("%.2f",
					float64(hop.RTT.Microseconds())/1000.0)
			}

			ip := hop.IP
			if hop.Timeout {
				ip = "*"
			}

			hostname := hop.HostName
			if hop.Timeout || hostname == "" {
				hostname = "*"
			}

			rows = append(rows, []string{
				fmt.Sprintf("%d", hop.HopNumber),
				ip,
				hostname,
				rttMs,
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
	rootCmd.AddCommand(traceCmd)
	traceCmd.Flags().IntVarP(&maxHops, "max-hops", "m", 30, "Maximum number of hops")
	traceCmd.Flags().DurationVarP(
		&traceTimeout,
		"timeout",
		"t",
		2*time.Second,
		"Timeout per hop (e.g., 2s, 500ms)",
	)
}
