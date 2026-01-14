/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"net"
	"strings"

	"github.com/ARCoder181105/netdiag/pkg/output"
	"github.com/spf13/cobra"
)

// digCmd represents the dig command
var digCmd = &cobra.Command{
	Use:   "dig <domain> [type]",
	Short: "Perform a DNS lookup (A, MX, TXT, NS, CNAME)",
	Long: `Perform a DNS lookup to find records for a domain. 
If no type is specified, it defaults to 'A' (IP Address).

Supported Record Types:
  A      : IPv4 Address (Where is the website hosted?)
  MX     : Mail Exchange (Who handles email for this domain?)
  TXT    : Text Records (Verification tokens, SPF spam protection, etc.)
  NS     : Name Servers (Which servers manage this domain's DNS?)
  CNAME  : Canonical Name (Is this domain an alias for another domain?)

Examples:
  netdiag dig google.com        (Find IP)
  netdiag dig github.com MX     (Find Mail Servers)
  netdiag dig google.com TXT    (Read Text Records)`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		domain := args[0]
		recordType := "A"
		if len(args) == 2 {
			recordType = args[1]
		}

		recordType = strings.ToUpper(recordType)
		headers := []string{"Type", "Result"}
		var rows [][]string

		output.PrintInfo(fmt.Sprintf("Querying %s records for %s...", recordType, domain))

		switch recordType {
		case "A":
			ips, err := net.LookupIP(domain)
			if err != nil {
				output.PrintError("Error looking up A records: " + err.Error())
				return
			}
			for _, ip := range ips {
				if ip.To4() != nil {
					rows = append(rows, []string{"A", ip.String()})
				}
			}
		case "MX":
			mxs, err := net.LookupMX(domain)
			if err != nil {
				output.PrintError("Error looking up MX records: " + err.Error())
				return
			}
			for _, mx := range mxs {
				rows = append(rows, []string{"MX", fmt.Sprintf("%d %s", mx.Pref, mx.Host)})
			}
		case "TXT":
			txts, err := net.LookupTXT(domain)
			if err != nil {
				output.PrintError("Error looking up TXT records: " + err.Error())
				return
			}
			for _, txt := range txts {
				rows = append(rows, []string{"TXT", txt})
			}
		case "NS":
			nss, err := net.LookupNS(domain)
			if err != nil {
				output.PrintError("Error looking up NS records: " + err.Error())
				return
			}
			for _, ns := range nss {
				rows = append(rows, []string{"NS", ns.Host})
			}
		case "CNAME":
			cname, err := net.LookupCNAME(domain)
			if err != nil {
				output.PrintError("Error looking up CNAME: " + err.Error())
				return
			}
			rows = append(rows, []string{"CNAME", cname})
		default:
			output.PrintError("Unsupported record type: " + recordType)
			output.PrintInfo("Supported types: A, MX, TXT, NS, CNAME")
			return
		}

		if len(rows) == 0 {
			output.PrintWarning("No records found.")
			return
		}

		output.PrintTable(headers, rows)
	},
}

func init() {
	rootCmd.AddCommand(digCmd)
}