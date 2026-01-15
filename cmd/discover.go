/*
Copyright © 2026 ARCoder181105 <EMAIL ADDRESS>
*/
// Package cmd implements the CLI commands.
package cmd

import (
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/spf13/cobra"

	"github.com/ARCoder181105/netdiag/pkg/output"
)

var discoverTimeout int

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Scan local network for devices",
	Long: `Discover active devices on your local network by performing a ping sweep.
This command automatically detects your local IP range and scans all 254 addresses.

Note: This usually requires Administrator/Sudo privileges to send ICMP packets.`,
	Run: func(_ *cobra.Command, _ []string) {
		output.PrintInfo("🔍 Detecting local network...")

		myIP, prefix, err := getLocalIPPrefix()
		if err != nil {
			output.PrintError(fmt.Sprintf("Failed to detect local network: %v", err))
			return
		}

		output.PrintSuccess(fmt.Sprintf("Local IP: %s", myIP))
		output.PrintInfo(fmt.Sprintf("Scanning range: %s.1 - %s.254 ...\n", prefix, prefix))

		var wg sync.WaitGroup
		var mu sync.Mutex

		type Device struct {
			IP       string
			Hostname string
			Latency  time.Duration
		}
		var foundDevices []Device

		sem := make(chan struct{}, 50)

		for i := 1; i < 255; i++ {
			targetIP := fmt.Sprintf("%s.%d", prefix, i)

			if targetIP == myIP {
				continue
			}

			wg.Add(1)
			go func(ip string) {
				defer wg.Done()

				sem <- struct{}{}
				defer func() { <-sem }()

				pinger, err := probing.NewPinger(ip)
				if err != nil {
					return
				}

				pinger.Count = 1
				pinger.Timeout = time.Duration(discoverTimeout) * time.Millisecond
				pinger.SetPrivileged(true)

				err = pinger.Run()
				if err != nil {
					return
				}

				stats := pinger.Statistics()
				if stats.PacketsRecv > 0 {
					hostname := resolveHostname(ip)

					mu.Lock()
					foundDevices = append(foundDevices, Device{
						IP:       ip,
						Hostname: hostname,
						Latency:  stats.AvgRtt,
					})
					mu.Unlock()

					fmt.Print(".")
				}
			}(targetIP)
		}

		wg.Wait()
		fmt.Println()

		if len(foundDevices) == 0 {
			output.PrintWarning("No devices found (Is your firewall blocking pings?)")
			return
		}

		sort.Slice(foundDevices, func(i, j int) bool {
			return foundDevices[i].IP < foundDevices[j].IP
		})

		headers := []string{"IP Address", "Hostname", "Latency"}
		var rows [][]string

		for _, dev := range foundDevices {
			rows = append(rows, []string{
				dev.IP,
				dev.Hostname,
				fmt.Sprintf("%v", dev.Latency),
			})
		}

		output.PrintTable(headers, rows)
		output.PrintSuccess(fmt.Sprintf("Scan complete. Found %d devices.", len(foundDevices)))
	},
}

func getLocalIPPrefix() (string, string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", "", err
	}

	var candidates []string

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip := ipnet.IP.String()

				if strings.HasPrefix(ip, "169.254") {
					continue
				}

				if strings.HasPrefix(ip, "192.168") {
					return extractPrefix(ip)
				}

				if strings.HasPrefix(ip, "10.") {
					candidates = append([]string{ip}, candidates...)
					continue
				}

				candidates = append(candidates, ip)
			}
		}
	}

	if len(candidates) > 0 {
		return extractPrefix(candidates[0])
	}

	return "", "", fmt.Errorf("no active local IP found")
}

func extractPrefix(ip string) (string, string, error) {
	lastDot := strings.LastIndex(ip, ".")
	if lastDot != -1 {
		return ip, ip[:lastDot], nil
	}
	return "", "", fmt.Errorf("invalid IP format")
}

func resolveHostname(ip string) string {
	names, err := net.LookupAddr(ip)
	if err == nil && len(names) > 0 {
		return strings.TrimSuffix(names[0], ".")
	}
	return "(Unknown)"
}

func init() {
	rootCmd.AddCommand(discoverCmd)
	discoverCmd.Flags().IntVarP(&discoverTimeout, "timeout", "t", 500, "Ping timeout in milliseconds")
}
