// Package cmd implements the CLI commands.
package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ARCoder181105/netdiag/pkg/config"
	"github.com/ARCoder181105/netdiag/pkg/logger"
	"github.com/ARCoder181105/netdiag/pkg/output"
	"github.com/ARCoder181105/netdiag/pkg/probe"
)

var (
	ports       string
	scanTimeout time.Duration
	concurrency int
)

var scanCmd = &cobra.Command{
	Use:   "scan <host>",
	Short: "Scan for open TCP ports",
	Long: `Scan a target host for open TCP ports using a high-concurrency worker pool.
You can specify a single port, a list, or a range.

Only scan hosts you own or have explicit permission to test.

Examples:
  netdiag scan google.com
  netdiag scan 192.168.1.1 --ports 80,443,8000-8100
  netdiag scan localhost -p 22 -t 2s`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		host := strings.TrimSpace(args[0])
		if host == "" {
			failUsage("no host given")
		}

		portList, err := probe.ParsePortRange(ports)
		if err != nil {
			failUsage(fmt.Sprintf("--ports: %v", err))
		}

		if concurrency < 1 {
			failUsage("--concurrency must be at least 1")
		}

		effectiveTimeout := scanTimeoutValue(cmd)
		requirePositiveDuration("--timeout", effectiveTimeout)

		scanner := &probe.ConnectScanner{
			Host:        host,
			Ports:       portList,
			Timeout:     effectiveTimeout,
			Concurrency: concurrency,
		}

		runProbe(scanner, host, probeOpts{
			LogAttrs: func(r probe.Result) []any {
				if r.ScanData == nil {
					return nil
				}
				return []any{
					"total_ports", r.ScanData.TotalPorts,
					"open_ports", r.ScanData.OpenPorts,
					"scan_method", r.ScanData.ScanMethod,
				}
			},
			Render: renderScan,
		})
	},
}

// scanTimeoutValue prefers an explicit --timeout, falling back to
// scan.default_timeout from ~/.netdiag.yaml.
func scanTimeoutValue(cmd *cobra.Command) time.Duration {
	if cmd.Flags().Changed("timeout") {
		return scanTimeout
	}
	configured := config.AppConfig.Scan.DefaultTimeout
	d, err := time.ParseDuration(configured)
	switch {
	case err != nil:
		logger.Log.Warn("Ignoring unparseable scan.default_timeout",
			"value", configured, "error", err, "using", scanTimeout)
	case d <= 0:
		logger.Log.Warn("Ignoring non-positive scan.default_timeout",
			"value", configured, "using", scanTimeout)
	default:
		return d
	}
	return scanTimeout
}

func renderScan(result probe.Result) {
	data := result.ScanData
	if data == nil {
		return
	}

	if len(data.OpenPorts) > 0 {
		headers := []string{"Port", "Protocol", "Status"}
		rows := make([][]string, 0, len(data.OpenPorts))

		for _, p := range data.OpenPorts {
			rows = append(rows, []string{fmt.Sprintf("%d", p), "TCP", "Open"})
		}

		fmt.Println()
		output.PrintTable(headers, rows)
	}

	output.PrintInfo(fmt.Sprintf(
		"Scanned %d ports in %s (%.1f ports/sec) using the %s method.",
		data.TotalPorts,
		result.Latency.Round(time.Millisecond),
		data.PortsPerSec,
		data.ScanMethod,
	))
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().DurationVarP(&scanTimeout, "timeout", "t", time.Second, "Connection timeout per port (e.g. 1s, 500ms)")
	scanCmd.Flags().StringVarP(&ports, "ports", "p", "1-1024", "Ports to scan: a list, a range, or both (e.g. 22,80,8000-8100)")
	scanCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 100, "Number of ports to probe concurrently")
}
