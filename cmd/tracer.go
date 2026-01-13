/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"net"
	"time"

	"github.com/ARCoder181105/netdiag/pkg/output"
	"github.com/spf13/cobra"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

var maxHops int

// traceCmd represents the trace command
var traceCmd = &cobra.Command{
	Use:   "trace [host]",
	Short: "Perform a traceroute to a destination host",
	Long: `Trace the network path to a destination host by sending ICMP packets
with increasing TTL values. Shows each hop (router) along the path.

Example:
  netdiag trace google.com
  netdiag trace 8.8.8.8`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		host := args[0]
		if err := runTrace(host, maxHops); err != nil {
			output.PrintError(fmt.Sprintf("Error: %v", err))
		}
	},
}

func init() {
	rootCmd.AddCommand(traceCmd)
	traceCmd.Flags().IntVarP(&maxHops, "max-hops", "m", 30, "Maximum number of hops")
}

// getHostname attempts to resolve an IP address to a hostname
func getHostname(ip string) string {
	names, err := net.LookupAddr(ip)
	if err == nil && len(names) > 0 {
		return names[0]
	}
	return ip
}

func runTrace(host string, maxHops int) error {
	// Step 1: Resolve Destination
	destAddr, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		return fmt.Errorf("failed to resolve host: %w", err)
	}

	output.PrintInfo(fmt.Sprintf("Traceroute to %s (%s), %d hops max\n", host, destAddr.String(), maxHops))

	// Step 2: Open raw IP connection for sending
	conn, err := net.ListenPacket("ip4:1", "0.0.0.0") // Protocol 1 = ICMP
	if err != nil {
		return fmt.Errorf("failed to open connection (try running with Administrator privileges): %w", err)
	}
	defer conn.Close()

	// Step 3: Wrap for IPv4 Control
	p := ipv4.NewPacketConn(conn)
	defer p.Close()

	// Open ICMP listener for receiving replies
	icmpConn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return fmt.Errorf("failed to open ICMP listener: %w", err)
	}
	defer icmpConn.Close()

	// Prepare table data
	headers := []string{"Hop", "IP Address", "Hostname", "RTT (ms)", "Status"}
	var rows [][]string

	// Step 4: The Main Loop (Discovery)
	reachedDestination := false
	for i := 1; i <= maxHops; i++ {
		// Step 5: Set TTL
		if err := p.SetTTL(i); err != nil {
			return fmt.Errorf("failed to set TTL: %w", err)
		}

		// Step 6: Send the Probe
		msg := icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   1234,
				Seq:  i,
				Data: []byte("TRACEROUTE"),
			},
		}

		msgBytes, err := msg.Marshal(nil)
		if err != nil {
			return fmt.Errorf("failed to marshal ICMP message: %w", err)
		}

		startTime := time.Now()

		// Send through the packet connection with TTL control
		if _, err := p.WriteTo(msgBytes, nil, destAddr); err != nil {
			return fmt.Errorf("failed to send probe: %w", err)
		}

		// Step 7: Wait for the Router's Reply
		reply := make([]byte, 1500)
		err = icmpConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if err != nil {
			return fmt.Errorf("failed to set read deadline: %w", err)
		}

		n, peer, err := icmpConn.ReadFrom(reply)
		rtt := time.Since(startTime)

		// Step 8: Analyze the Reply
		if err != nil {
			// Timeout - router ignored us
			rows = append(rows, []string{
				fmt.Sprintf("%d", i),
				"*",
				"*",
				"*",
				"Timeout",
			})
			continue
		}

		// Parse the ICMP message
		parsedMsg, err := icmp.ParseMessage(1, reply[:n])
		if err != nil {
			rows = append(rows, []string{
				fmt.Sprintf("%d", i),
				peer.String(),
				"*",
				"*",
				"Parse Error",
			})
			continue
		}

		// Get IP address and hostname
		ipAddr := peer.String()
		hostname := getHostname(ipAddr)
		rttMs := fmt.Sprintf("%.2f", float64(rtt.Microseconds())/1000.0)

		switch parsedMsg.Type {
		case ipv4.ICMPTypeTimeExceeded:
			// Case A: We hit a router!
			rows = append(rows, []string{
				fmt.Sprintf("%d", i),
				ipAddr,
				hostname,
				rttMs,
				"Router",
			})

		case ipv4.ICMPTypeEchoReply:
			// Case B: We hit the destination!
			rows = append(rows, []string{
				fmt.Sprintf("%d", i),
				ipAddr,
				hostname,
				rttMs,
				"Destination",
			})
			reachedDestination = true

			// Print the table
			fmt.Println()
			output.PrintTable(headers, rows)
			fmt.Println()
			output.PrintSuccess("Trace complete!")
			return nil

		default:
			rows = append(rows, []string{
				fmt.Sprintf("%d", i),
				ipAddr,
				hostname,
				rttMs,
				fmt.Sprintf("Type: %v", parsedMsg.Type),
			})
		}
	}

	// Print the table even if we didn't reach destination
	fmt.Println()
	output.PrintTable(headers, rows)
	fmt.Println()

	if !reachedDestination {
		output.PrintWarning("Max hops reached without reaching destination")
	}

	return nil
}
