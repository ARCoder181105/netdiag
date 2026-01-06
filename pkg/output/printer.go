package output

import (
	"os"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

// In this package it contains the output formatters for the tool with colors scheme

func PrintError(msg string) {
	color.Red(msg)
}

func PrintSuccess(msg string) {
	color.Green(msg)
}

func PrintWarning(msg string) {
	color.Yellow(msg)
}

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