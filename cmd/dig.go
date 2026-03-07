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

	"github.com/ARCoder181105/netdiag/pkg/output"
	"github.com/ARCoder181105/netdiag/pkg/probe"
	"github.com/spf13/cobra"
)

var digServer string
var digTimeout int

// digCmd represents the dig command
var digCmd = &cobra.Command{
	Use:   "dig <domain> [type]",
	Short: "Perform a DNS lookup (A, MX, TXT, NS, CNAME)",
	Long: `Perform a DNS lookup to find records for a domain.
If no type is specified, it defaults to 'A'.

Supported Record Types:
  A      : IPv4 Address
  MX     : Mail Exchange
  TXT    : Text Records
  NS     : Name Servers
  CNAME  : Canonical Name

Examples:
  netdiag dig google.com
  netdiag dig github.com MX
  netdiag dig google.com TXT`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(_ *cobra.Command, args []string) {

		recordType := "A"
		if len(args) == 2 {
			recordType = args[1]
		}

		prober := &probe.DigProber{
			Host:       args[0],
			Server:     digServer,
			RecordType: recordType,
			Timeout:    time.Duration(digTimeout) * time.Second,
		}

		result, err := prober.Probe(context.Background())

		// 1️⃣ Hard failure fallback (for JSON consistency)
		if err != nil {
			result = probe.Result{
				Target:    args[0],
				ProbeType: "dns",
				Success:   false,
				Severity:  probe.SeverityError,
				Message:   err.Error(),
				TimeStamp: time.Now(),
			}
		}

		if jsonOutput {
			output.PrintJSON(result)
			return
		}

		output.PrintInfo(fmt.Sprintf(
			"Querying %s records for %s...",
			strings.ToUpper(recordType),
			args[0],
		))

		if !result.Success || result.DNSData == nil {
			output.PrintError(result.Message)
			return
		}

		headers := []string{"Type", "Value"}
		var rows [][]string

		for _, record := range result.DNSData.Records {
			rows = append(rows, []string{
				record.Type,
				record.Value,
			})
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
	rootCmd.AddCommand(digCmd)

	digCmd.Flags().StringVarP(
		&digServer,
		"server",
		"s",
		"",
		"Custom DNS server to query (e.g., 8.8.8.8 or 8.8.8.8:5333)",
	)

	digCmd.Flags().IntVarP(
		&digTimeout,
		"timeout",
		"t",
		5,
		"Timeout in seconds",
	)
}
