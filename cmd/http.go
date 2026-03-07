/*
Copyright © 2026 ARCoder181105 <EMAIL ADDRESS>
*/

// Package cmd implements the CLI commands.
package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ARCoder181105/netdiag/pkg/logger"
	"github.com/ARCoder181105/netdiag/pkg/output"
	"github.com/ARCoder181105/netdiag/pkg/probe"
)

var (
	timeOut int
	method  string
	skipTLS bool
)

var httpCmd = &cobra.Command{
	Use:   "http <url>",
	Short: "Check website status and SSL certificate",
	Long: `Check the HTTP status and SSL certificate expiration of a website.

Examples:
  netdiag http example.com
  netdiag http https://example.com
  netdiag http example.com --timeout 10
  netdiag http example.com --method POST
  netdiag http example.com --skip-tls`,
	Args: cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {

		url := args[0]
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "https://" + url
		}

		prober := &probe.HTTPProber{
			URL:           url,
			Method:        method,
			Timeout:       time.Duration(timeOut) * time.Second,
			SkipTLSVerify: skipTLS,
		}

		result, err := prober.Probe(context.Background())

		if err != nil {
			result = probe.Result{
				Target:    url,
				ProbeType: "http",
				Success:   false,
				Severity:  probe.SeverityError,
				Message:   err.Error(),
				TimeStamp: time.Now(),
			}
		}

		// ── Structured logging ────────────────────────────────────────────────
		if result.Success && result.HTTPData != nil {
			logger.Log.Info("http check completed",
				"target", result.Target,
				"status_code", result.HTTPData.StatusCode,
				"latency_ms", result.Latency.Milliseconds(),
				"tls_valid", result.HTTPData.TLSValid,
				"tls_days_left", result.HTTPData.TLSDaysLeft,
			)
		} else {
			logger.Log.Error("http check failed",
				"target", result.Target,
				"error", result.Message,
			)
		}
		// ─────────────────────────────────────────────────────────────────────

		if jsonOutput {
			output.PrintJSON(result)
			return
		}

		if result.HTTPData == nil {
			output.PrintError(result.Message)
			return
		}

		data := result.HTTPData

		headers := []string{
			"URL", "Method", "Status", "Latency",
			"Redirects", "TLS Valid", "TLS Days", "Content Length",
		}

		tlsDays := "-"
		if data.TLSDaysLeft > 0 {
			tlsDays = fmt.Sprintf("%d", data.TLSDaysLeft)
		}

		rows := [][]string{
			{
				result.Target,
				method,
				fmt.Sprintf("%d", data.StatusCode),
				result.Latency.String(),
				fmt.Sprintf("%d", data.Redirects),
				fmt.Sprintf("%t", data.TLSValid),
				tlsDays,
				fmt.Sprintf("%d", data.ContentLength),
			},
		}

		fmt.Println()
		output.PrintTable(headers, rows)

		switch result.Severity {
		case probe.SeverityOK:
			output.PrintSuccess(result.Message)
		case probe.SeverityWarning:
			output.PrintWarning(result.Message)
		case probe.SeverityError:
			output.PrintError(result.Message)
		default:
			output.PrintInfo(result.Message)
		}
	},
}

func init() {
	rootCmd.AddCommand(httpCmd)
	httpCmd.Flags().IntVarP(&timeOut, "timeout", "t", 5, "Timeout for the request (seconds)")
	httpCmd.Flags().StringVarP(&method, "method", "m", "GET", "HTTP method for the request")
	httpCmd.Flags().BoolVar(&skipTLS, "skip-tls", false, "Skip TLS certificate verification")
}
