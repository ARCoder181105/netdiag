// Package cmd implements the CLI commands.
package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ARCoder181105/netdiag/pkg/output"
	"github.com/ARCoder181105/netdiag/pkg/probe"
)

var (
	maxHops      int
	traceTimeout time.Duration
)

var traceCmd = &cobra.Command{
	Use:   "trace <host>",
	Short: "Perform a traceroute to a destination host",
	Long: `Trace the network path to a destination host by sending ICMP packets
with increasing TTL values. Shows each hop (router) along the path.

Requires raw socket access: run as root, or grant the binary CAP_NET_RAW
(sudo setcap cap_net_raw+ep /usr/local/bin/netdiag).

Examples:
  netdiag trace google.com
  netdiag trace 8.8.8.8 -m 20`,
	Args: cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		host := strings.TrimSpace(args[0])
		if host == "" {
			failUsage("no host given")
		}
		if maxHops < 1 || maxHops > 255 {
			failUsage("--max-hops must be between 1 and 255")
		}

		prober := &probe.TraceProber{
			Host:    host,
			MaxHops: maxHops,
			Timeout: traceTimeout,
		}

		runProbe(prober, host, probeOpts{
			LogAttrs: func(r probe.Result) []any {
				if r.TraceData == nil {
					return nil
				}
				return []any{"hops", len(r.TraceData.Hops)}
			},
			Render: renderTrace,
		})
	},
}

func renderTrace(result probe.Result) {
	data := result.TraceData
	if data == nil || len(data.Hops) == 0 {
		return
	}

	headers := []string{"Hop", "IP Address", "Hostname", "RTT (ms)"}
	rows := make([][]string, 0, len(data.Hops))

	for _, hop := range data.Hops {
		rtt, ip, hostname := "*", "*", "*"

		if !hop.Timeout {
			rtt = fmt.Sprintf("%.2f", float64(hop.RTT.Microseconds())/1000.0)
			ip = hop.IP
			if hop.HostName != "" {
				hostname = hop.HostName
			}
		}

		rows = append(rows, []string{
			fmt.Sprintf("%d", hop.HopNumber),
			ip,
			hostname,
			rtt,
		})
	}

	fmt.Println()
	output.PrintTable(headers, rows)
	fmt.Println()
}

func init() {
	rootCmd.AddCommand(traceCmd)
	traceCmd.Flags().IntVarP(&maxHops, "max-hops", "m", 30, "Maximum number of hops")
	traceCmd.Flags().DurationVarP(
		&traceTimeout, "timeout", "t", 2*time.Second,
		"Timeout per hop (e.g. 2s, 500ms)",
	)
}
