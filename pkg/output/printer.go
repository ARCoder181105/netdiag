// Package output handles formatting and printing of messages and tables to stdout.
package output

import (
	"os"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

// PrintError prints a message in red.
func PrintError(msg string) {
	color.Red(msg)
}

// PrintSuccess prints a message in green.
func PrintSuccess(msg string) {
	color.Green(msg)
}

// PrintWarning prints a message in yellow.
func PrintWarning(msg string) {
	color.Yellow(msg)
}

// PrintInfo prints a message in cyan.
func PrintInfo(msg string) {
	color.Cyan(msg)
}

// PrintTable renders a table with headers and rows
func PrintTable(headers []string, rows [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.AppendBulk(rows)
	table.Render()
}
