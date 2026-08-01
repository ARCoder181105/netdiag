// Package cmd implements the CLI commands.
package cmd

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

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
		target, err := normalizeURL(args[0])
		if err != nil {
			failUsage(err.Error())
		}

		reqTimeout := time.Duration(timeOut) * time.Second
		requirePositiveDuration("--timeout", reqTimeout)

		// Normalize once so the request and the rendered table agree.
		method = strings.ToUpper(strings.TrimSpace(method))

		// Not in JSON mode: stdout must stay parseable. JSON callers get the
		// same information from the tls_verify_skipped field.
		if skipTLS && !jsonOutput {
			output.PrintWarning("TLS certificate verification is disabled (--skip-tls)")
		}

		prober := &probe.HTTPProber{
			URL:           target,
			Method:        method,
			Timeout:       reqTimeout,
			SkipTLSVerify: skipTLS,
		}

		runProbe(prober, target, probeOpts{
			LogAttrs: func(r probe.Result) []any {
				if r.HTTPData == nil {
					return nil
				}
				return []any{
					"status_code", r.HTTPData.StatusCode,
					"tls_valid", r.HTTPData.TLSValid,
					"tls_days_left", r.HTTPData.TLSDaysLeft,
				}
			},
			Render: renderHTTP,
		})
	},
}

// normalizeURL defaults a bare host to https and rejects anything that is not
// a usable http(s) URL, rather than letting it fail deep inside net/http.
func normalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("no URL given")
	}

	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", raw, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported scheme %q: only http and https are supported", parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("invalid URL %q: no host", raw)
	}

	return parsed.String(), nil
}

func renderHTTP(result probe.Result) {
	data := result.HTTPData
	if data == nil {
		return
	}

	tlsValid := "-"
	tlsDays := "-"
	if data.TLSChecked {
		tlsValid = fmt.Sprintf("%t", data.TLSValid)
		if data.TLSVerifySkipped {
			tlsValid += " (unverified)"
		}
		tlsDays = fmt.Sprintf("%d", data.TLSDaysLeft)
	}

	headers := []string{
		"URL", "Method", "Status", "Latency",
		"Redirects", "TLS Valid", "TLS Days", "Issuer", "Content Length",
	}

	rows := [][]string{{
		result.Target,
		method,
		fmt.Sprintf("%d", data.StatusCode),
		result.Latency.Round(time.Millisecond).String(),
		fmt.Sprintf("%d", data.Redirects),
		tlsValid,
		tlsDays,
		orDash(data.TLSIssuer),
		fmt.Sprintf("%d", data.ContentLength),
	}}

	fmt.Println()
	output.PrintTable(headers, rows)
}

// orDash renders empty strings as "-" so table cells never look truncated.
func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func init() {
	rootCmd.AddCommand(httpCmd)
	httpCmd.Flags().IntVarP(&timeOut, "timeout", "t", 5, "Timeout for the request (seconds)")
	httpCmd.Flags().StringVarP(&method, "method", "m", "GET", "HTTP method for the request")
	httpCmd.Flags().BoolVar(&skipTLS, "skip-tls", false, "Skip TLS certificate verification (insecure)")
}
