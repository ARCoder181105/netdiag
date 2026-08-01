package cmd

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/ARCoder181105/netdiag/pkg/output"
	"github.com/ARCoder181105/netdiag/pkg/probe"
)

var (
	count       int
	timeout     time.Duration
	interval    time.Duration
	pingWorkers int
)

// pingCmd represents the ping command
var pingCmd = &cobra.Command{
	Use:   "ping <host> [more hosts...]",
	Short: "Send ICMP ECHO_REQUEST to network hosts",
	Long: `Ping sends ICMP ECHO_REQUEST packets to the specified
network hosts and reports the responses.

Uses unprivileged ICMP datagram sockets where available, falling back to raw
sockets, so no sudo is required on most systems.

Examples:
  netdiag ping google.com
  netdiag ping -c 5 -i 2s github.com cloudflare.com`,
	Args: cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		hosts, err := validHosts(args)
		if err != nil {
			failUsage(err.Error())
		}
		if count < 1 {
			failUsage("--count must be at least 1")
		}
		if pingWorkers < 1 {
			// errgroup.SetLimit(0) makes every Go call block forever.
			failUsage("--concurrency must be at least 1")
		}
		requirePositiveDuration("--timeout", timeout)
		requirePositiveDuration("--interval", interval)

		ctx, stop := signalContext()
		defer stop()

		grp, gctx := errgroup.WithContext(ctx)
		grp.SetLimit(pingWorkers)

		var (
			mu       sync.Mutex
			results  []probe.Result
			hardFail bool
		)

		for _, host := range hosts {
			h := host

			grp.Go(func() error {
				prober := &probe.PingProber{
					Host:     h,
					Count:    count,
					Timeout:  timeout,
					Interval: interval,
				}

				result, err := prober.Probe(gctx)

				mu.Lock()
				if err != nil {
					// The probe could not run at all — see exitRuntime.
					result = probe.ErrorResult("ping", h, err)
					hardFail = true
				}
				results = append(results, result)
				mu.Unlock()

				return nil
			})
		}

		_ = grp.Wait()

		failCode := exitProbe
		if hardFail {
			failCode = exitRuntime
		}

		// Restore the caller's argument order; goroutines finish out of order.
		sortResultsBy(results, hosts)

		for _, r := range results {
			logResult(r, pingLogAttrs)
		}

		if jsonOutput {
			output.PrintJSON(results)
			exitForAll(results, failCode)
			return
		}

		renderPing(results)
		exitForAll(results, failCode)
	},
}

func pingLogAttrs(r probe.Result) []any {
	if r.PingData == nil {
		return nil
	}
	return []any{
		"resolved_ip", r.PingData.ResolvedIP,
		"loss_pct", r.PingData.PacketLoss,
	}
}

// validHosts trims and rejects empty host arguments.
func validHosts(args []string) ([]string, error) {
	hosts := make([]string, 0, len(args))
	for _, a := range args {
		h := strings.TrimSpace(a)
		if h == "" {
			return nil, fmt.Errorf("empty host argument")
		}
		hosts = append(hosts, h)
	}
	return hosts, nil
}

// sortResultsBy reorders results to match the order of hosts.
func sortResultsBy(results []probe.Result, hosts []string) {
	index := make(map[string]int, len(hosts))
	for i, h := range hosts {
		index[h] = i
	}

	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && index[results[j].Target] < index[results[j-1].Target]; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
}

func renderPing(results []probe.Result) {
	headers := []string{
		"Host", "IP", "Sent", "Received", "Loss",
		"Min RTT", "Avg RTT", "Max RTT", "StdDev RTT", "Status",
	}
	rows := make([][]string, 0, len(results))

	for _, result := range results {
		ip, sent, recv := "-", "-", "-"
		loss := "100.00%"
		minRTT, avgRTT, maxRTT, stddev := "-", "-", "-", "-"

		if result.PingData != nil {
			d := result.PingData
			ip = d.ResolvedIP
			sent = fmt.Sprintf("%d", d.PacketsSent)
			recv = fmt.Sprintf("%d", d.PacketsRecv)
			loss = fmt.Sprintf("%.2f%%", d.PacketLoss)
			minRTT = d.MinRTT.Round(time.Microsecond).String()
			avgRTT = d.AvgRTT.Round(time.Microsecond).String()
			maxRTT = d.MaxRTT.Round(time.Microsecond).String()
			stddev = d.StdDevRTT.Round(time.Microsecond).String()
		}

		rows = append(rows, []string{
			result.Target, ip, sent, recv, loss,
			minRTT, avgRTT, maxRTT, stddev,
			result.Severity.String(),
		})
	}

	fmt.Println()
	output.PrintTable(headers, rows)

	for _, result := range results {
		output.PrintBySeverity(result.Severity, fmt.Sprintf("%s: %s", result.Target, result.Message))
	}
}

func init() {
	rootCmd.AddCommand(pingCmd)

	pingCmd.Flags().IntVarP(&count, "count", "c", 3, "Number of ICMP packets to send")
	pingCmd.Flags().DurationVarP(&timeout, "timeout", "t", 1*time.Second, "Timeout per host (e.g. 1s, 500ms)")
	pingCmd.Flags().DurationVarP(&interval, "interval", "i", 1*time.Second, "Time to wait between packets (e.g. 1s, 500ms)")
	pingCmd.Flags().IntVar(&pingWorkers, "concurrency", 20, "Number of hosts to ping concurrently")
}
