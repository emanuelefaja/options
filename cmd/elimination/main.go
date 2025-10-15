package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"mnmlsm/analysis"
)

func main() {
	fmt.Println("🪐 Running Stock Elimination Filters...")
	fmt.Println()

	result, err := analysis.RunElimination()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Printf("📊 Portfolio Metrics:\n")
	fmt.Printf("   Total Net Worth:  %s\n", formatCurrency(result.TotalNetWorth))
	fmt.Printf("   Dry Powder:       %s\n", formatCurrency(result.DryPowder))
	fmt.Printf("   10%% Position Max: %s\n", formatCurrency(result.TotalNetWorth*0.10))
	fmt.Printf("   20%% Sector Max:   %s\n", formatCurrency(result.TotalNetWorth*0.20))
	fmt.Println()

	// Print survivors
	fmt.Printf("✅ %d stocks passed all filters:\n\n", len(result.Survivors))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SYMBOL\tPRICE\tSECTOR\tPOS COST\tPOS%\tSECTOR%\tSTOCK\tPUT\tEXISTING CAP")
	fmt.Fprintln(w, "------\t-----\t------\t--------\t----\t-------\t-----\t---\t------------")

	for _, c := range result.Survivors {
		stockFlag := " "
		if c.ExistingStockPosition {
			stockFlag = "✓"
		}

		putFlag := " "
		if c.ExistingPutPosition {
			putFlag = "✓"
		}

		fmt.Fprintf(w, "%s\t$%.2f\t%s\t$%.0f\t%.1f%%\t%.1f%%\t%s\t%s\t$%.0f\n",
			c.Symbol,
			c.Price,
			truncate(c.Sector, 15),
			c.PositionCost,
			c.PositionSizePercent,
			c.SectorPercent,
			stockFlag,
			putFlag,
			c.ExistingCapital,
		)
	}
	w.Flush()

	fmt.Println()
	fmt.Printf("❌ %d stocks eliminated:\n\n", len(result.Eliminated))

	// Print eliminated stocks
	for symbol, reason := range result.Eliminated {
		fmt.Printf("   %s: %s\n", symbol, reason)
	}

	fmt.Println()
	fmt.Printf("💾 Saved to: data/solar-system.csv\n")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

func formatCurrency(amount float64) string {
	// Handle negative numbers
	isNegative := amount < 0
	if isNegative {
		amount = -amount
	}

	// Format with no decimal places
	formatted := fmt.Sprintf("%.0f", amount)

	// Add commas
	var result string
	for i, digit := range formatted {
		if i > 0 && (len(formatted)-i)%3 == 0 {
			result += ","
		}
		result += string(digit)
	}

	if isNegative {
		return "-$" + result
	}
	return "$" + result
}
