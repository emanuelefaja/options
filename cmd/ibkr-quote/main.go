package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"mnmlsm/analysis"
	"mnmlsm/ibkr"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

func main() {
	// Command-line flags
	symbol := flag.String("symbol", "", "Stock symbol to query")
	format := flag.String("format", "table", "Output format (table or json)")
	premiumScan := flag.Bool("premium-scan", false, "Scan for premium opportunities")
	minReturn := flag.Float64("min-return", 100, "Minimum annualized return % for premium scan")
	maxDTE := flag.Int("max-dte", 4, "Maximum days to expiration")
	strikeRange := flag.Float64("strike-range", 5, "Strike price range around current price")
	right := flag.String("right", "P", "Option type: C (call) or P (put)")
	exchange := flag.String("exchange", "NASDAQ", "Exchange (NASDAQ, NYSE, etc.)")
	csvOutput := flag.String("csv", "", "Output results to CSV file")
	flag.Parse()

	if *symbol == "" {
		fmt.Println("Error: --symbol is required")
		flag.Usage()
		os.Exit(1)
	}

	// Create IBKR client
	client := ibkr.NewClient()

	if *premiumScan {
		// Run premium scan
		runPremiumScan(client, *symbol, *exchange, *right, *strikeRange, *minReturn, *maxDTE, *csvOutput)
	} else {
		// Get single quote
		runQuote(client, *symbol, *format)
	}
}

func runQuote(client *ibkr.Client, symbol, format string) {
	fmt.Printf("Fetching quote for %s...\n\n", symbol)

	quote, err := client.GetQuote(symbol)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if format == "json" {
		formatQuoteJSON(quote)
	} else {
		formatQuoteTable(quote)
	}
}

func runPremiumScan(client *ibkr.Client, symbol, exchange, right string, strikeRange, minReturn float64, maxDTE int, csvFile string) {
	fmt.Printf("🔍 Scanning %s %s options for premium opportunities...\n\n", symbol, right)

	// Create scanner
	scanner := analysis.NewScanner(client)

	// Set up scan parameters
	params := analysis.ScanParams{
		Symbol:      symbol,
		Exchange:    exchange,
		Right:       right,
		StrikeRange: strikeRange,
		MinReturn:   minReturn,
		MaxDTE:      maxDTE,
	}

	fmt.Println("1. Searching for underlying...")

	// Run scan with progress tracking
	contracts, err := scanner.ScanPremiums(params)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n5. Analyzing %d contracts...\n", len(contracts))

	if len(contracts) == 0 {
		fmt.Printf("\nNo contracts found meeting criteria (>%.0f%% annualized, ≤%d DTE)\n", minReturn, maxDTE)
		return
	}

	// Sort by annualized return (highest first)
	sort.Slice(contracts, func(i, j int) bool {
		return contracts[i].AnnualizedReturn > contracts[j].AnnualizedReturn
	})

	// Display results
	printPremiumTable(contracts)

	// Save to CSV if requested
	if csvFile != "" {
		if err := savePremiumsToCSV(contracts, csvFile); err != nil {
			fmt.Printf("\nError saving to CSV: %v\n", err)
		} else {
			fmt.Printf("\n✅ Results saved to %s\n", csvFile)
		}
	}
}

func formatQuoteTable(quote *ibkr.Quote) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintln(w, "SYMBOL\tPRICE\tCHANGE\tCHANGE%\tBID\tASK\tVOLUME")
	fmt.Fprintln(w, strings.Repeat("-", 80))

	changeSign := ""
	if quote.Change > 0 {
		changeSign = "+"
	}

	fmt.Fprintf(w, "%s\t$%.2f\t%s%.2f\t%s%.2f%%\t$%.2f\t$%.2f\t%d\n",
		quote.Symbol,
		quote.Price,
		changeSign, quote.Change,
		changeSign, quote.ChangePerc,
		quote.Bid,
		quote.Ask,
		quote.Volume,
	)

	w.Flush()
}

func formatQuoteJSON(quote *ibkr.Quote) {
	fmt.Printf(`{
  "symbol": "%s",
  "price": %.2f,
  "change": %.2f,
  "changePercent": %.2f,
  "bid": %.2f,
  "ask": %.2f,
  "volume": %d
}
`, quote.Symbol, quote.Price, quote.Change, quote.ChangePerc,
		quote.Bid, quote.Ask, quote.Volume)
}

func printPremiumTable(contracts []analysis.OptionContract) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "\n✅ Found %d qualifying contracts:\n\n", len(contracts))
	fmt.Fprintln(w, "STRIKE\tEXPIRY\tDTE\tPREMIUM\tPREM%\tANNUALIZED\tDELTA\tCAPITAL")
	fmt.Fprintln(w, strings.Repeat("-", 90))

	for _, c := range contracts {
		// Parse expiry date for display
		expiryDate, _ := time.Parse("20060102", c.MaturityDate)
		expiryStr := expiryDate.Format("Jan 02")

		fmt.Fprintf(w, "$%.2f\t%s\t%dd\t$%.0f\t%.2f%%\t%.0f%%\t%.3f\t$%.0f\n",
			c.Strike,
			expiryStr,
			c.DTE,
			c.Premium,
			c.PremiumPercent,
			c.AnnualizedReturn,
			c.Delta,
			c.CapitalRequired,
		)
	}

	w.Flush()
}

func savePremiumsToCSV(contracts []analysis.OptionContract, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"Symbol", "Strike", "Expiry", "DTE", "Premium",
		"Premium%", "Annualized%", "Delta", "Gamma", "Theta",
		"Vega", "IV", "Bid", "Ask", "Capital", "ConID",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write rows
	for _, c := range contracts {
		row := []string{
			c.Symbol,
			fmt.Sprintf("%.2f", c.Strike),
			c.MaturityDate,
			fmt.Sprintf("%d", c.DTE),
			fmt.Sprintf("%.2f", c.Premium),
			fmt.Sprintf("%.2f", c.PremiumPercent),
			fmt.Sprintf("%.2f", c.AnnualizedReturn),
			fmt.Sprintf("%.4f", c.Delta),
			fmt.Sprintf("%.4f", c.Gamma),
			fmt.Sprintf("%.4f", c.Theta),
			fmt.Sprintf("%.4f", c.Vega),
			fmt.Sprintf("%.2f", c.ImpliedVol),
			fmt.Sprintf("%.2f", c.Bid),
			fmt.Sprintf("%.2f", c.Ask),
			fmt.Sprintf("%.2f", c.CapitalRequired),
			fmt.Sprintf("%d", c.ConID),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}
