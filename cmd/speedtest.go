// Package cmd implements the CLI commands.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ARCoder181105/netdiag/pkg/output"
	"github.com/ARCoder181105/netdiag/pkg/probe"
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
	Args: cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		prober := &probe.SpeedTestProber{
			ServerID: serverID,
			NoUpload: noUpload,
		}

		runProbe(prober, "internet", probeOpts{
			Banner: "🌐 Running Internet Speed Test (this may take a moment)...",
			LogAttrs: func(r probe.Result) []any {
				if r.SpeedTestData == nil {
					return nil
				}
				return []any{
					"download_mbps", r.SpeedTestData.DownloadMbps,
					"upload_mbps", r.SpeedTestData.UploadMbps,
					"ping_ms", r.SpeedTestData.PingMs,
				}
			},
			Render: renderSpeedtest,
		})
	},
}

func renderSpeedtest(result probe.Result) {
	data := result.SpeedTestData
	if data == nil {
		return
	}

	headers := []string{"Metric", "Value"}
	rows := [][]string{
		{"ISP", orDash(data.ISP)},
		{"Public IP", orDash(data.PublicIP)},
		{"Server", fmt.Sprintf("%s (%s)", data.ServerName, data.Country)},
		{"Sponsor", orDash(data.Sponsor)},
		{"Distance", fmt.Sprintf("%.2f km", data.DistanceKm)},
		{"Ping", fmt.Sprintf("%.2f ms", data.PingMs)},
		{"Download", fmt.Sprintf("%.2f Mbps", data.DownloadMbps)},
	}

	// ponytail: 0 Mbps reads as "skipped"; a genuine 0.00 upload hides the row.
	// Add SpeedTestData.UploadSkipped if that ever matters.
	if data.UploadMbps > 0 {
		rows = append(rows, []string{"Upload", fmt.Sprintf("%.2f Mbps", data.UploadMbps)})
	}

	fmt.Println()
	output.PrintTable(headers, rows)
	fmt.Println()
}

func init() {
	rootCmd.AddCommand(speedtestCmd)
	speedtestCmd.Flags().BoolVarP(&noUpload, "no-upload", "u", false, "Skip upload test")
	speedtestCmd.Flags().StringVarP(&serverID, "server", "s", "", "Specify speedtest server ID")
}
