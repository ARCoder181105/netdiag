/*
Copyright © 2026 ARCoder181105 <EMAIL ADDRESS>
*/
// Package cmd implements the CLI commands.
package cmd

import (
	"fmt"
	"strconv"

	"github.com/showwin/speedtest-go/speedtest"
	"github.com/spf13/cobra"

	"github.com/ARCoder181105/netdiag/pkg/output"
)

var (
	serverID string
	noUpload bool
)

var speedtestCmd = &cobra.Command{
	Use:   "speedtest",
	Short: "Run internet speed test (Download/Upload)",
	Long: `Test your internet connection speed including download and upload speeds.

This command will automatically select the closest server and measure:
- Ping latency
- Download speed
- Upload speed (can be skipped with --no-upload)

Examples:
  netdiag speedtest
  netdiag speedtest --no-upload
  netdiag speedtest --server 12345`,
	Run: func(_ *cobra.Command, _ []string) {
		output.PrintInfo("🌐 Starting Internet Speed Test...\n")

		output.PrintInfo("Fetching user information...")
		user, err := speedtest.FetchUserInfo()
		if err != nil {
			output.PrintError(fmt.Sprintf("Error fetching user info: %v", err))
			return
		}

		output.PrintSuccess(fmt.Sprintf("ISP: %s", user.String()))
		output.PrintSuccess(fmt.Sprintf("Public IP: %s\n", user.IP))

		output.PrintInfo("Fetching server list...")
		serverList, err := speedtest.FetchServers()
		if err != nil {
			output.PrintError(fmt.Sprintf("Error fetching servers: %v", err))
			return
		}

		var targets speedtest.Servers

		if serverID != "" {
			output.PrintInfo(fmt.Sprintf("Looking for server ID: %s", serverID))
			serverIDInt, err := strconv.Atoi(serverID)
			if err != nil {
				output.PrintError("Invalid server ID format")
				return
			}

			targets, err = serverList.FindServer([]int{serverIDInt})
			if err != nil || len(targets) == 0 {
				output.PrintError("Server not found")
				return
			}
		} else {
			output.PrintInfo("Finding closest server...")
			targets, err = serverList.FindServer([]int{})
			if err != nil || len(targets) == 0 {
				output.PrintError("No servers found")
				return
			}
		}

		target := targets[0]
		output.PrintSuccess(fmt.Sprintf(
			"Testing against: %s (%s) - %s\n",
			target.Name,
			target.Country,
			target.Sponsor,
		))

		output.PrintInfo("🏓 Running Ping Test...")
		if err := target.PingTest(nil); err != nil {
			output.PrintWarning(fmt.Sprintf("Ping failed: %v", err))
		} else {
			output.PrintSuccess(fmt.Sprintf(
				"Ping: %.2f ms",
				float64(target.Latency.Milliseconds()),
			))
		}

		output.PrintInfo("⬇️  Running Download Test...")
		if err := target.DownloadTest(); err != nil {
			output.PrintError(fmt.Sprintf("Download failed: %v", err))
			return
		}

		output.PrintSuccess(fmt.Sprintf(
			"Download Speed: %s",
			formatSpeed(float64(target.DLSpeed)),
		))

		if !noUpload {
			output.PrintInfo("⬆️  Running Upload Test...")
			if err := target.UploadTest(); err != nil {
				output.PrintError(fmt.Sprintf("Upload failed: %v", err))
				return
			}

			output.PrintSuccess(fmt.Sprintf(
				"Upload Speed: %s",
				formatSpeed(float64(target.ULSpeed)),
			))
		} else {
			output.PrintWarning("Upload test skipped (--no-upload)")
		}

		fmt.Println("\n=== Speed Test Results ===")

		headers := []string{"Metric", "Value"}
		rows := [][]string{
			{"Server", fmt.Sprintf("%s (%s)", target.Name, target.Country)},
			{"Sponsor", target.Sponsor},
			{"Distance", fmt.Sprintf("%.2f km", target.Distance)},
			{"Ping", fmt.Sprintf("%.2f ms", float64(target.Latency.Milliseconds()))},
			{"Download", formatSpeed(float64(target.DLSpeed))},
		}

		if !noUpload {
			rows = append(rows, []string{
				"Upload", formatSpeed(float64(target.ULSpeed)),
			})
		}

		output.PrintTable(headers, rows)

		fmt.Println()
		assessConnection(
			float64(target.DLSpeed),
			float64(target.ULSpeed),
			float64(target.Latency.Milliseconds()),
		)
	},
}

func formatSpeed(speed float64) string {
	bitsPerSec := speed * 8
	mbps := bitsPerSec / 1_000_000

	if mbps >= 1000 {
		gbps := mbps / 1000
		return fmt.Sprintf("%.2f Gbps", gbps)
	}
	return fmt.Sprintf("%.2f Mbps", mbps)
}

func assessConnection(dlSpeed, ulSpeed, pingMs float64) {
	dlMbps := (dlSpeed * 8) / 1_000_000
	ulMbps := (ulSpeed * 8) / 1_000_000

	output.PrintInfo("📊 Connection Quality Assessment:")

	switch {
	case dlMbps >= 100:
		output.PrintSuccess(fmt.Sprintf("  Download: Excellent (%.0f Mbps)", dlMbps))
	case dlMbps >= 25:
		output.PrintSuccess(fmt.Sprintf("  Download: Good (%.0f Mbps)", dlMbps))
	case dlMbps >= 10:
		output.PrintWarning(fmt.Sprintf("  Download: Fair (%.0f Mbps)", dlMbps))
	default:
		output.PrintError("  Download: Poor (<10 Mbps)")
	}

	if ulMbps > 0 {
		switch {
		case ulMbps >= 50:
			output.PrintSuccess(fmt.Sprintf("  Upload: Excellent (%.0f Mbps)", ulMbps))
		case ulMbps >= 10:
			output.PrintSuccess(fmt.Sprintf("  Upload: Good (%.0f Mbps)", ulMbps))
		case ulMbps >= 5:
			output.PrintWarning(fmt.Sprintf("  Upload: Fair (%.0f Mbps)", ulMbps))
		default:
			output.PrintError("  Upload: Poor (<5 Mbps)")
		}
	}

	switch {
	case pingMs < 20:
		output.PrintSuccess("  Ping: Excellent (<20 ms)")
	case pingMs < 50:
		output.PrintSuccess("  Ping: Good (<50 ms)")
	case pingMs < 100:
		output.PrintWarning("  Ping: Fair (<100 ms)")
	default:
		output.PrintError("  Ping: Poor (>=100 ms)")
	}
}

func init() {
	rootCmd.AddCommand(speedtestCmd)

	speedtestCmd.Flags().BoolVarP(
		&noUpload,
		"no-upload",
		"u",
		false,
		"Skip upload test",
	)

	speedtestCmd.Flags().StringVarP(
		&serverID,
		"server",
		"s",
		"",
		"Specify server ID",
	)
}
