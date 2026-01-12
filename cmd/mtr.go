// /*
// Copyright © 2026 NAME HERE <EMAIL ADDRESS>
// */

// Currently Not Working Properly So Please Ignore this

// package cmd

// import (
// 	"fmt"
// 	"time"

// 	"github.com/ARCoder181105/netdiag/pkg/output"
// 	go_mtr "github.com/smuzoey/go-mtr"
// 	"github.com/spf13/cobra"
// )

// var (
// 	ttl   int
// 	retry int
// )

// // mtrCmd represents the mtr command
// var mtrCmd = &cobra.Command{
// 	Use:   "mtr <host>",
// 	Short: "Run a real-time MTR trace to a host",
// 	Long: `MTR (My Traceroute) combines the functionality of traceroute and ping
// into a single network diagnostic tool. It shows the route packets take
// to reach a destination and displays statistics for each hop.

// Examples:
//   netdiag mtr google.com
//   netdiag mtr -t 20 -r 3 github.com`,
// 	Args: cobra.MinimumNArgs(1),
// 	Run: func(cmd *cobra.Command, args []string) {
// 		host := args[0]

// 		output.PrintInfo(fmt.Sprintf("Tracing route to %s... (Max TTL: %d, Retry: %d)", host, ttl, retry))

// 		// 1. Create the tracer configuration
// 		// We set MaxUnReply to 8 so it doesn't quit immediately if one hop is silent
// 		tracer, err := go_mtr.NewTrace(go_mtr.Config{
// 			ICMP:        true,
// 			MaxUnReply:  8,
// 			NextHopWait: time.Millisecond * 200, // Speed of the trace
// 		})
// 		if err != nil {
// 			output.PrintError(fmt.Sprintf("Failed to initialize tracer: %v", err))
// 			return
// 		}

// 		// 2. Define the trace target
// 		trace := &go_mtr.Trace{
// 			SrcAddr: go_mtr.GetOutbondIP(),
// 			DstAddr: host,
// 			MaxTTL:  uint8(ttl),
// 			Retry:   retry,
// 		}

// 		// 3. Execute the batch trace
// 		// We send 1 batch of probes
// 		res, err := tracer.BatchTrace([]go_mtr.Trace{*trace}, 1)
// 		if err != nil {
// 			output.PrintError(fmt.Sprintf("Trace failed to execute: %v", err))
// 			return
// 		}

// 		if len(res) == 0 {
// 			output.PrintWarning("No results returned from tracer.")
// 			return
// 		}

// 		// 4. Use the Library's Aggregate function
// 		// This replaces your manual 'hopMap' logic. It automatically averages 
// 		// the latency and packet loss for repeated packets on the same hop.
// 		aggregated := res[0].Aggregate()

// 		// OUTPUT LOGIC
// 		headers := []string{"Hop", "IP", "Reached", "Packet Loss", "Avg Latency"}
// 		rows := [][]string{}

// 		for _, hop := range aggregated.Res {
// 			// Format the data
// 			// hop.SrcTTL is the IP address of the router at this hop
// 			ip := hop.SrcTTL
// 			if ip == "" {
// 				ip = "*"
// 			}

// 			reached := "No"
// 			if hop.Reached {
// 				reached = "Yes"
// 			}

// 			// PacketLoss is returned as a float (0.0 to 1.0 or 0 to 100 depending on implementation)
// 			// Usually go-mtr returns a ratio (0.1 = 10%), so we multiply by 100.
// 			// We format it to 1 decimal place.
// 			loss := hop.PacketLoss * 100

// 			row := []string{
// 				fmt.Sprintf("%d", hop.TTL),
// 				ip,
// 				reached,
// 				fmt.Sprintf("%.1f%%", loss),
// 				hop.Latency.String(),
// 			}
// 			rows = append(rows, row)
// 		}

// 		output.PrintTable(headers, rows)

// 		// 5. Diagnostic Check
// 		// If the very first hop is "*", it means even the local gateway is blocked.
// 		if len(rows) > 0 && rows[0][1] == "*" {
// 			fmt.Println() // Empty line
// 			output.PrintError("CRITICAL: Hop 1 failed to respond.")
// 			output.PrintWarning("Diagnostic Tips:")
// 			output.PrintInfo("1. Are you running with 'sudo'? (Required for ICMP)")
// 			output.PrintInfo("2. Is your Firewall (ufw/iptables) blocking INCOMING 'Time Exceeded' packets?")
// 			output.PrintInfo("3. If using Docker, use '--network host' or '--cap-add=NET_ADMIN'.")
// 		}
// 	},
// }

// func init() {
// 	rootCmd.AddCommand(mtrCmd)
// 	mtrCmd.Flags().IntVarP(&ttl, "ttl", "t", 30, "Max hops to check")
// 	mtrCmd.Flags().IntVarP(&retry, "retry", "r", 2, "How many times to retry each hop")
// }
