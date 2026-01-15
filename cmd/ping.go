/*
Copyright © 2026 ARCoder181105 <EMAIL ADDRESS>
*/

// Package cmd implements the CLI commands.
package cmd

import (
	"context"
	"fmt"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/ARCoder181105/netdiag/pkg/output"
)

var (
	count    int
	timeout  int
	interval int
)

// PingResult holds the statistics of a ping execution.
type PingResult struct {
	Host       string
	IP         string
	Loss       float64
	AvgLatency time.Duration
	MinLatency time.Duration
	MaxLatency time.Duration
}

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
		group, _ := errgroup.WithContext(context.Background())
		group.SetLimit(20)
		var lock sync.Mutex
		var results []PingResult

		for _, host := range args {
			h := host
			group.Go(func() error {
				pinger, err := probing.NewPinger(h)
				if err != nil {
					return err
				}

				pinger.SetPrivileged(true)

				err = pinger.Resolve()
				if err != nil {
					lock.Lock()
					results = append(results, PingResult{
						Host:       h,
						IP:         "Resolution Failed",
						Loss:       100.0,
						AvgLatency: 0,
						MinLatency: 0,
						MaxLatency: 0,
					})
					lock.Unlock()
					return nil
				}

				pinger.Count = count
				pinger.Interval = time.Duration(interval) * time.Second
				pinger.Timeout = time.Duration(timeout) * time.Second * time.Duration(count)

				pinger.OnFinish = func(stats *probing.Statistics) {
					lock.Lock()
					defer lock.Unlock()

					result := PingResult{
						Host:       h,
						IP:         pinger.IPAddr().String(),
						Loss:       stats.PacketLoss,
						AvgLatency: stats.AvgRtt,
						MinLatency: stats.MinRtt,
						MaxLatency: stats.MaxRtt,
					}
					results = append(results, result)
				}

				err = pinger.Run()
				if err != nil {
					return err
				}

				return nil
			})
		}

		if err := group.Wait(); err != nil {
			fmt.Println("Error:", err)
			return
		}

		headers := []string{"Host", "IP", "Packet Loss", "Avg Latency", "Max Latency", "Min Latency"}
		rows := [][]string{}

		for _, result := range results {
			row := []string{
				result.Host,
				result.IP,
				fmt.Sprintf("%.2f%%", result.Loss),
				result.AvgLatency.String(),
				result.MaxLatency.String(),
				result.MinLatency.String(),
			}
			rows = append(rows, row)
		}

		// Print the table
		output.PrintTable(headers, rows)
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)

	pingCmd.Flags().IntVarP(
		&count,
		"count",
		"c",
		3,
		"Number of ICMP packets to send",
	)

	pingCmd.Flags().IntVarP(
		&timeout,
		"timeout",
		"t",
		1,
		"Timeout per packet (seconds)",
	)

	pingCmd.Flags().IntVarP(
		&interval,
		"interval",
		"i",
		1,
		"Time to wait between packets (seconds)",
	)
}
