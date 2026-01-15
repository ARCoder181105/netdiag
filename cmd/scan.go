/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/ARCoder181105/netdiag/pkg/output"
)

var (
	ports       string
	scanTimeout int
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan <host>",
	Short: "Scan for open TCP ports",
	Long: `Scan a target host for open TCP ports using a high-concurrency worker pool.
You can specify a single port, a list, or a range.

Examples:
  netdiag scan google.com
  netdiag scan 192.168.1.1 --ports 80,443,8000-8100
  netdiag scan localhost -p 22 -t 2`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		host := args[0]

		// Parse the ports
		portList := parsePortRange(ports)

		if len(portList) == 0 {
			output.PrintError("No valid ports parsed. Please check your --ports flag.")
			return
		}

		output.PrintInfo(fmt.Sprintf("Starting scan against %s (%d ports)...\n", host, len(portList)))

		// Setup Concurrency
		results := make(chan int)
		var wg sync.WaitGroup
		// Semaphore
		sem := make(chan struct{}, 100)

		for _, p := range portList {
			wg.Add(1)
			go func(port int) {
				defer wg.Done()

				// Acquire token (blocks if 100 scans are running)
				sem <- struct{}{}
				defer func() { <-sem }() // Release token

				address := net.JoinHostPort(host, strconv.Itoa(port))
				conn, err := net.DialTimeout("tcp", address, time.Duration(scanTimeout)*time.Second)
				if err == nil {
					conn.Close()
					results <- port // Send open port to results channel
				}
			}(p)
		}

		// Waits for all workers to finish, then closes the channel so the reader knows to stop.
		go func() {
			wg.Wait()
			close(results)
		}()

		// This loop blocks until the channel is closed.
		var openPorts []int
		for p := range results {
			openPorts = append(openPorts, p)
		}

		// Output Processing
		if len(openPorts) == 0 {
			output.PrintWarning("No open ports found.")
			return
		}

		sort.Ints(openPorts)

		headers := []string{"Port", "Protocol", "Status"}
		var rows [][]string

		for _, p := range openPorts {
			rows = append(rows, []string{
				fmt.Sprintf("%d", p),
				"TCP",
				"Open",
			})
		}

		fmt.Println()
		output.PrintTable(headers, rows)
		output.PrintSuccess(fmt.Sprintf("Scan complete. Found %d open ports.", len(openPorts)))
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().IntVarP(&scanTimeout, "timeout", "t", 1, "Timeout in seconds")
	scanCmd.Flags().StringVarP(&ports, "ports", "p", "1-1024", "The range to scan")
}

// parsePortRange converts strings like "80,443,1000-1005" into a slice of integers
func parsePortRange(portStr string) []int {
	var result []int
	parts := strings.Split(portStr, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.Contains(part, "-") {
			// Handle range (e.g., "80-443")
			rangeParts := strings.Split(part, "-")

			if len(rangeParts) != 2 {
				fmt.Printf("Invalid range format: %s (skipping)\n", part)
				continue
			}

			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				fmt.Printf("Invalid start port '%s': %v (skipping)\n", rangeParts[0], err)
				continue
			}

			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				fmt.Printf("Invalid end port '%s': %v (skipping)\n", rangeParts[1], err)
				continue
			}

			// Swap if range is reversed (e.g., "443-80")
			if start > end {
				start, end = end, start
			}

			// Validate port range (TCP ports: 1-65535)
			if start < 1 || start > 65535 {
				fmt.Printf("Start port %d out of valid range (1-65535) (skipping)\n", start)
				continue
			}
			if end < 1 || end > 65535 {
				fmt.Printf("End port %d out of valid range (1-65535) (skipping)\n", end)
				continue
			}

			for i := start; i <= end; i++ {
				result = append(result, i)
			}
		} else {
			// Handle single port (e.g., "80")
			num, err := strconv.Atoi(part)
			if err != nil {
				fmt.Printf("Invalid port '%s': %v (skipping)\n", part, err)
				continue
			}

			if num < 1 || num > 65535 {
				fmt.Printf("Port %d out of valid range (1-65535) (skipping)\n", num)
				continue
			}

			result = append(result, num)
		}
	}

	return result
}
