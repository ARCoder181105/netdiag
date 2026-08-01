// Package cmd implements the CLI commands.
package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ARCoder181105/netdiag/pkg/output"
	"github.com/ARCoder181105/netdiag/pkg/probe"
)

var (
	digServer  string
	digTimeout time.Duration
)

var digCmd = &cobra.Command{
	Use:   "dig <domain> [type]",
	Short: "Perform a DNS lookup (A, AAAA, MX, TXT, NS, CNAME)",
	Long: `Perform a DNS lookup to find records for a domain.
If no type is specified, it defaults to 'A'.

Supported Record Types:
  A      : IPv4 Address
  AAAA   : IPv6 Address
  MX     : Mail Exchange
  TXT    : Text Records
  NS     : Name Servers
  CNAME  : Canonical Name

Examples:
  netdiag dig google.com
  netdiag dig github.com MX
  netdiag dig google.com TXT
  netdiag dig google.com --server 8.8.8.8`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(_ *cobra.Command, args []string) {
		host := strings.TrimSpace(args[0])
		if host == "" {
			failUsage("no domain given")
		}

		recordType := "A"
		if len(args) == 2 {
			recordType = strings.ToUpper(strings.TrimSpace(args[1]))
			if !probe.IsSupportedRecordType(recordType) {
				failUsage(fmt.Sprintf(
					"unsupported record type %q: want one of %s",
					args[1], strings.Join(probe.SupportedRecordTypes, ", "),
				))
			}
		}

		prober := &probe.DigProber{
			Host:       host,
			Server:     digServer,
			RecordType: recordType,
			Timeout:    digTimeout,
		}

		runProbe(prober, host, probeOpts{
			Banner: fmt.Sprintf("Querying %s records for %s...", recordType, host),
			LogAttrs: func(r probe.Result) []any {
				if r.DNSData == nil {
					return nil
				}
				return []any{
					"record_type", recordType,
					"record_count", len(r.DNSData.Records),
					"server", orDash(r.DNSData.Server),
				}
			},
			Render: renderDig,
		})
	},
}

func renderDig(result probe.Result) {
	data := result.DNSData
	if data == nil || len(data.Records) == 0 {
		return
	}

	headers := []string{"Type", "Value"}
	rows := make([][]string, 0, len(data.Records))

	for _, record := range data.Records {
		rows = append(rows, []string{record.Type, record.Value})
	}

	fmt.Println()
	output.PrintTable(headers, rows)
}

func init() {
	rootCmd.AddCommand(digCmd)

	digCmd.Flags().StringVarP(
		&digServer, "server", "s", "",
		"Custom DNS server to query (e.g. 8.8.8.8 or 8.8.8.8:5353)",
	)
	digCmd.Flags().DurationVarP(
		&digTimeout, "timeout", "t", 5*time.Second,
		"Query timeout (e.g. 5s, 500ms)",
	)
}
