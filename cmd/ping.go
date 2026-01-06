/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ARCoder181105/netdiag/pkg/output"
	probing "github.com/prometheus-community/pro-bing"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	count    int
	timeout  int
	interval int
)

type PingResult struct {
	Host    string
	IP      string
	Loss    float64
	Latency time.Duration
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
	Run: func(cmd *cobra.Command, args []string) {

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
					return err
				}

				pinger.Count = count
				pinger.Interval = time.Duration(interval) * time.Second
				pinger.Timeout = time.Duration(timeout) * time.Second * time.Duration(count)

				pinger.OnFinish = func(stats *probing.Statistics) {
					lock.Lock()
					defer lock.Unlock()

					result := PingResult{
						Host:    h,
						IP:      stats.IPAddr.String(), 
						Loss:    stats.PacketLoss,
						Latency: stats.AvgRtt,
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

		headers := []string{"Host", "IP", "Packet Loss", "Avg Latency"}
		rows := [][]string{}

		for _, result := range results {
			row := []string{
				result.Host,
				result.IP,
				fmt.Sprintf("%.2f%%", result.Loss),
				result.Latency.String(),
			}
			rows = append(rows, row)
		}

		// Print the table
		output.PrintTable(headers, rows)

	},
}

func init() {
	rootCmd.AddCommand(pingCmd)

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// pingCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run
	// when this command is called directly.
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
