// Package cmd implements the CLI commands.
package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ARCoder181105/netdiag/pkg/probe"
)

var whoisTimeout time.Duration

var whoisCmd = &cobra.Command{
	Use:   "whois <domain>",
	Short: "Retrieve domain registration information",
	Long: `Query the WHOIS database to find information about a domain name,
including the registrar, creation date, and expiration date.

Examples:
  netdiag whois example.com
  netdiag whois example.com --timeout 20s`,
	Args: cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		domain := strings.TrimSpace(args[0])
		if domain == "" {
			failUsage("no domain given")
		}

		prober := &probe.WhoisProber{
			Domain:  domain,
			Timeout: whoisTimeout,
		}

		runProbe(prober, domain, probeOpts{
			Banner: fmt.Sprintf("Querying WHOIS for %s...", domain),
			Render: renderWhois,
		})
	},
}

func renderWhois(result probe.Result) {
	if result.WhoisData == nil || result.WhoisData.Raw == "" {
		return
	}

	fmt.Println()
	fmt.Println(result.WhoisData.Raw)
	fmt.Println()
}

func init() {
	rootCmd.AddCommand(whoisCmd)
	whoisCmd.Flags().DurationVarP(&whoisTimeout, "timeout", "t", 10*time.Second, "Query timeout (e.g. 10s)")
}
