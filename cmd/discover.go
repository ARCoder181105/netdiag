// Package cmd implements the CLI commands.
package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/ARCoder181105/netdiag/pkg/output"
	"github.com/ARCoder181105/netdiag/pkg/probe"
)

var discoverTimeout int

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Scan local network for devices",
	Long: `Sweep the local IPv4 network for reachable devices.

The network is derived from the first active non-loopback interface, including
its netmask, so non-/24 networks are handled correctly.

Examples:
  netdiag discover
  netdiag discover -t 1000`,
	Args: cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		prober := &probe.DiscoverProber{
			Timeout: time.Duration(discoverTimeout) * time.Millisecond,
		}

		runProbe(prober, "local-network", probeOpts{
			Banner: "🔍 Scanning local network (this may take a moment)...",
			LogAttrs: func(r probe.Result) []any {
				if r.DiscoverData == nil {
					return nil
				}
				return []any{
					"prefix", r.DiscoverData.Prefix,
					"devices_found", len(r.DiscoverData.Devices),
				}
			},
			Render: renderDiscover,
		})
	},
}

func renderDiscover(result probe.Result) {
	data := result.DiscoverData
	if data == nil || len(data.Devices) == 0 {
		return
	}

	headers := []string{"IP Address", "Hostname", "Latency"}
	rows := make([][]string, 0, len(data.Devices))

	for _, dev := range data.Devices {
		rows = append(rows, []string{
			dev.IP,
			dev.HostName,
			dev.Latency.Round(time.Millisecond).String(),
		})
	}

	fmt.Println()
	output.PrintTable(headers, rows)
}

func init() {
	rootCmd.AddCommand(discoverCmd)
	discoverCmd.Flags().IntVarP(&discoverTimeout, "timeout", "t", 500, "Ping timeout in milliseconds")
}
