/*
Copyright © 2026 ARCoder181105 <EMAIL ADDRESS>
*/

package cmd

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/ARCoder181105/netdiag/pkg/logger"
	"github.com/ARCoder181105/netdiag/pkg/output"
	"github.com/ARCoder181105/netdiag/pkg/probe"
)

var (
	count    int
	timeout  time.Duration
	interval time.Duration
)

// pingCmd represents the ping command
var pingCmd = &cobra.Command{
	Use:   "ping <host> [more hosts...]",
	Short: "Send ICMP ECHO_REQUEST to network hosts",
	Long: `Ping sends ICMP ECHO_REQUEST packets to the specified
network hosts and reports the responses.

Examples:
  netdiag ping google.com
  netdiag ping -c 5 -i 2 github.com cloudflare.com`,
	Args: cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		grp, ctx := errgroup.WithContext(context.Background())
		var mu sync.Mutex
		var results []probe.Result

		for _, host := range args {
			h := host // capture correctly

			grp.Go(func() error {
				prober := &probe.PingProber{
					Host:     h,
					Count:    count,
					Timeout:  timeout,
					Interval: interval,
				}

				result, err := prober.Probe(ctx)

				if err != nil {
					result = probe.Result{
						Target:    h,
						ProbeType: "ping",
						Success:   false,
						Severity:  probe.SeverityError,
						Message:   err.Error(),
						TimeStamp: time.Now(),
					}
				}

				mu.Lock()
				results = append(results, result)
				mu.Unlock()

				return nil
			})
		}

		_ = grp.Wait()

		// --- THIS IS THE FIX: Write to the structured logger ---
		for _, r := range results {
			if r.Success {
				logger.Log.Info("Ping completed",
					"target", r.Target,
					"latency_ms", float64(r.Latency.Microseconds())/1000.0,
					"loss_pct", r.PingData.PacketLoss,
				)
			} else {
				logger.Log.Error("Ping failed",
					"target", r.Target,
					"error", r.Message,
				)
			}
		}
		// -------------------------------------------------------

		if jsonOutput {
			output.PrintJSON(results)
			return
		}

		headers := []string{
			"Host", "IP", "Sent", "Received", "Loss",
			"Min RTT", "Avg RTT", "Max RTT", "StdDev RTT",
			"Success", "Severity", "Message",
		}
		var rows [][]string

		for _, result := range results {
			ip, sent, recv := "-", "-", "-"
			loss := "100.00%"
			min, avg, max, stddev := "-", "-", "-", "-"

			if result.PingData != nil {
				ip = result.PingData.ResolvedIP
				sent = fmt.Sprintf("%d", result.PingData.PacketsSent)
				recv = fmt.Sprintf("%d", result.PingData.PacketsRecv)
				loss = fmt.Sprintf("%.2f%%", result.PingData.PacketLoss)
				min = result.PingData.MinRTT.String()
				avg = result.PingData.AvgRTT.String()
				max = result.PingData.MaxRTT.String()
				stddev = result.PingData.StdDevRTT.String()
			}

			rows = append(rows, []string{
				result.Target, ip, sent, recv, loss,
				min, avg, max, stddev,
				fmt.Sprintf("%t", result.Success),
				result.Severity.String(),
				result.Message,
			})
		}

		fmt.Println()
		output.PrintTable(headers, rows)
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)

	pingCmd.Flags().IntVarP(&count, "count", "c", 3, "Number of ICMP packets to send")
	pingCmd.Flags().DurationVarP(&timeout, "timeout", "t", 1*time.Second, "Timeout per packet (e.g., 1s, 500ms)")
	pingCmd.Flags().DurationVarP(&interval, "interval", "i", 1*time.Second, "Time to wait between packets (e.g., 1s, 500ms)")
}
