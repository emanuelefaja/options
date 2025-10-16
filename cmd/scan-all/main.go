package main

import (
	"flag"
	"fmt"
	"os"

	"mnmlsm/analysis"
	"mnmlsm/ibkr"
)

func main() {
	// Command line flags
	right := flag.String("right", "P", "Option type: C for calls, P for puts")
	minReturn := flag.Float64("min-return", 100, "Minimum annualized return percentage")
	strikeRange := flag.Float64("strike-range", 5.0, "Strike range around current price in dollars (e.g., 5.0 = $5)")
	numExpiries := flag.Int("expiries", 2, "Number of Friday expiries to scan")
	output := flag.String("output", "data/options-chain.csv", "Output CSV file path")
	solarSystem := flag.String("input", "data/solar-system.csv", "Input solar-system.csv file path")

	flag.Parse()

	// Validate right parameter
	if *right != "P" && *right != "C" {
		fmt.Fprintf(os.Stderr, "Error: --right must be 'P' or 'C'\n")
		os.Exit(1)
	}

	// Create IBKR client
	client := ibkr.NewClient()

	// Create scanner
	scanner := analysis.NewScanner(client)

	// Setup batch scan parameters
	params := analysis.BatchScanParams{
		SolarSystemCSV: *solarSystem,
		OutputCSV:      *output,
		Right:          *right,
		MinReturn:      *minReturn,
		StrikeRange:    *strikeRange,
		NumExpiries:    *numExpiries,
	}

	// Run batch scan
	if err := scanner.ScanAllStocks(params); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
