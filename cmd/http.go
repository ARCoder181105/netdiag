package cmd

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ARCoder181105/netdiag/pkg/output"
	"github.com/spf13/cobra"
)

var (
	timeOut int
	method  string
)

// httpCmd represents the http command
var httpCmd = &cobra.Command{
	Use:   "http <url>",
	Short: "Check website status and SSL certificate",
	Long: `Check the HTTP status and SSL certificate expiration of a website.
	
Examples:
  yourapp http example.com
  yourapp http https://example.com
  yourapp http example.com --timeout 10`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		host := args[0]
		if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
			host = "https://" + host
		}

		// Create client with timeout
		client := &http.Client{
			Timeout: time.Duration(timeOut) * time.Second,
		}

		// Make request
		req, err := http.NewRequest(method, host, nil)
		if err != nil {
			output.PrintError(fmt.Sprintf("Error creating request: %v", err))
			return
		}

		start := time.Now()
		resp, err := client.Do(req)
		duration := time.Since(start)
		if err != nil {
			output.PrintError(fmt.Sprintf("Error making request: %v", err))
			return
		}
		defer resp.Body.Close()

		// Status
		printStatusCode(resp.StatusCode)
		// Latency
		output.PrintInfo(fmt.Sprintf("Latency: %v", duration))

		// Check if HTTPS and analyze SSL
		if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
			analyzeSSL(resp.TLS)
		} else {
			output.PrintWarning("No SSL/TLS connection detected")
		}
	},
}

func printStatusCode(statusCode int) {
	msg := fmt.Sprintf("Status Code: %d", statusCode)

	if statusCode >= 200 && statusCode < 300 {
		output.PrintSuccess(msg)
	} else if statusCode >= 500 {
		output.PrintError(msg)
	} else if statusCode >= 400 {
		output.PrintWarning(msg)
	} else {
		output.PrintInfo(msg)
	}
}

func analyzeSSL(tlsState *tls.ConnectionState) {
	cert := tlsState.PeerCertificates[0]

	now := time.Now()
	daysRemaining := int(cert.NotAfter.Sub(now).Hours() / 24)

	// SSL info
	fmt.Println("\nSSL Certificate Information:")
	fmt.Println(strings.Repeat("-", 50))

	headers := []string{"Property", "Value"}
	rows := [][]string{
		{"Subject", cert.Subject.CommonName},
		{"Issuer", cert.Issuer.CommonName},
		{"Valid From", cert.NotBefore.Format("2006-01-02 15:04:05")},
		{"Valid Until", cert.NotAfter.Format("2006-01-02 15:04:05")},
		{"Days Remaining", fmt.Sprintf("%d days", daysRemaining)},
	}

	output.PrintTable(headers, rows)

	// Warning if expiring soon
	if daysRemaining < 30 {
		output.PrintWarning(fmt.Sprintf("\n⚠️  WARNING: Certificate expires in %d days!", daysRemaining))
	} else if daysRemaining < 0 {
		output.PrintError(fmt.Sprintf("\n❌ ERROR: Certificate expired %d days ago!", -daysRemaining))
	} else {
		output.PrintSuccess(fmt.Sprintf("\n✓ Certificate is valid for %d more days", daysRemaining))
	}
}

func init() {
	rootCmd.AddCommand(httpCmd)

	httpCmd.Flags().IntVarP(&timeOut, "timeout", "t", 5, "Timeout for the request (seconds)")
	httpCmd.Flags().StringVarP(&method, "method", "m", "GET", "HTTP method for the request")
}
