// Package cmd implements the CLI commands.
package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ARCoder181105/netdiag/pkg/config"
	"github.com/ARCoder181105/netdiag/pkg/logger"
	"github.com/ARCoder181105/netdiag/pkg/output"
	"github.com/ARCoder181105/netdiag/pkg/probe"
)

var (
	ports         string
	scanTimeout   time.Duration
	concurrency   int
	fastScan      bool
	benchmarkScan bool
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
  netdiag scan localhost -p 22 -t 2s
  netdiag scan 192.168.1.1 -p 1-65535 --fast`,
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

		connect := &probe.ConnectScanner{
			Host:        host,
			Ports:       portList,
			Timeout:     effectiveTimeout,
			Concurrency: concurrency,
		}

		syn := &probe.SYNScanner{
			Host:        host,
			Ports:       portList,
			Timeout:     effectiveTimeout,
			Concurrency: concurrency,
			Fallback:    connect,
			// Diagnostics go to stderr so --json stdout stays parseable.
			Notify: output.PrintErrorLine,
		}

		if benchmarkScan {
			runScanBenchmark(host, connect, syn)
			return
		}

		var scanner probe.Prober = connect
		if fastScan {
			scanner = syn
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

// runScanBenchmark runs both scan methods against the same target and prints a
// comparison.
//
// It does not go through runProbe: runProbe runs one probe and exits, which is
// the right shape for every other command. Rather than teach it about a second
// result, the benchmark drives the two Probers directly — they are the same
// production types the normal path uses.
func runScanBenchmark(host string, connect *probe.ConnectScanner, syn *probe.SYNScanner) {
	ctx, stop := signalContext()
	defer stop()

	if !jsonOutput {
		output.PrintInfo(fmt.Sprintf("Benchmarking both scan methods against %s...", host))
	}

	results := make([]probe.Result, 0, 2)
	for _, scanner := range []probe.Prober{connect, syn} {
		result, err := scanner.Probe(ctx)
		if err != nil {
			output.PrintErrorLine(fmt.Sprintf("benchmark aborted: %v", err))
			os.Exit(exitRuntime)
		}
		results = append(results, result)
		logResult(result, nil)
	}

	if jsonOutput {
		output.PrintJSON(results)
		exitForAll(results, exitProbe)
	}

	renderScanBenchmark(results)
	exitForAll(results, exitProbe)
}

// renderScanBenchmark prints the comparison table. Speedup is relative to the
// first row, which is always the connect scan.
func renderScanBenchmark(results []probe.Result) {
	headers := []string{"Method", "Duration", "Ports/sec", "Open ports", "Speedup"}
	rows := make([][]string, 0, len(results))

	baseline := results[0].Latency

	for i, r := range results {
		speedup := "1.0x (baseline)"
		if i > 0 && r.Latency > 0 {
			speedup = fmt.Sprintf("%.1fx", float64(baseline)/float64(r.Latency))
		}

		method := "unknown"
		portsPerSec, openPorts := 0.0, 0
		if r.ScanData != nil {
			method = r.ScanData.ScanMethod
			portsPerSec = r.ScanData.PortsPerSec
			openPorts = len(r.ScanData.OpenPorts)
		}

		rows = append(rows, []string{
			method,
			r.Latency.Round(time.Millisecond).String(),
			fmt.Sprintf("%.0f", portsPerSec),
			fmt.Sprintf("%d", openPorts),
			speedup,
		})
	}

	fmt.Println()
	output.PrintTable(headers, rows)

	// A SYN scan that fell back to the connect scan would otherwise look like a
	// suspiciously fair fight.
	if len(results) == 2 && results[1].ScanData != nil && results[1].ScanData.ScanMethod != "syn" {
		output.PrintErrorLine("The SYN run fell back to the connect scan, so this is not a comparison of two methods.")
	}
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
	scanCmd.Flags().BoolVar(&fastScan, "fast", false, "Use a half-open SYN scan (needs CAP_NET_RAW; falls back to the connect scan)")
	scanCmd.Flags().BoolVar(&benchmarkScan, "benchmark", false, "Run both scan methods against the target and compare them")
}
