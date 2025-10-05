package web

import (
	"fmt"
	"sort"
)

// CalculateSymbolSummaries groups all positions by symbol and calculates aggregated metrics
func CalculateSymbolSummaries() []SymbolSummary {
	// Load option positions
	optionTransactions := LoadOptionTransactions("data/options_transactions.csv")
	optionPositions := CalculateOptionPositions(optionTransactions)

	// Load stock positions
	stockTransactions := LoadStockTransactions("data/stocks_transactions.csv")
	stockPrices := LoadStockPrices("data/stock_prices.csv")
	stockPositions := CalculateAllPositions(stockTransactions, stockPrices)

	// Group by symbol
	symbolMap := make(map[string]*SymbolSummary)

	// Process option positions
	for _, opt := range optionPositions {
		if _, exists := symbolMap[opt.Symbol]; !exists {
			symbolMap[opt.Symbol] = &SymbolSummary{
				Symbol: opt.Symbol,
			}
		}

		// Add premiums
		symbolMap[opt.Symbol].PremiumsCollected += opt.NetPremium

		// Track capital
		if opt.Status == "Open" {
			symbolMap[opt.Symbol].TotalCapital += opt.Capital
		}
	}

	// Process stock positions
	for _, stock := range stockPositions {
		if _, exists := symbolMap[stock.Symbol]; !exists {
			symbolMap[stock.Symbol] = &SymbolSummary{
				Symbol: stock.Symbol,
			}
		}

		// Add realized P/L from closed positions
		if stock.Type == "closed" {
			symbolMap[stock.Symbol].StockPL += stock.RealizedPnL
		}

		// Track capital from open positions
		if stock.Type == "open" {
			symbolMap[stock.Symbol].TotalCapital += stock.CostBasis
		}
	}

	// Calculate total P/L and format values
	var summaries []SymbolSummary
	for _, summary := range symbolMap {
		summary.TotalPL = summary.PremiumsCollected + summary.StockPL
		summary.TotalPLFormatted = formatCurrencyValue(summary.TotalPL)
		summary.PremiumsFormatted = formatCurrencyValue(summary.PremiumsCollected)
		summary.StockPLFormatted = formatCurrencyValue(summary.StockPL)
		summary.CapitalFormatted = FormatCurrency(summary.TotalCapital)
		summaries = append(summaries, *summary)
	}

	// Sort alphabetically by symbol
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Symbol < summaries[j].Symbol
	})

	return summaries
}

// GetSymbolDetails calculates detailed metrics for a single symbol
func GetSymbolDetails(symbol string, portfolioTotalPL float64) SymbolDetails {
	// Load all positions
	optionPositions := GetOptionPositionsBySymbol(symbol)
	stockPrices := LoadStockPrices("data/stock_prices.csv")
	stockPositions := CalculateAllPositions(LoadStockTransactions("data/stocks_transactions.csv"), stockPrices)

	details := SymbolDetails{
		Symbol: symbol,
	}

	// Calculate option metrics
	totalDTE := 0.0
	totalReturn := 0.0
	optionCount := 0

	for _, opt := range optionPositions {
		details.TotalPremiumCollected += opt.NetPremium
		details.NumberOfOptionsTrades++

		// Track DTE (even for closed positions, this is the original DTE)
		totalDTE += float64(opt.DaysToExpiry)

		// Track returns
		if opt.PercentReturn != 0 {
			totalReturn += opt.PercentReturn
			optionCount++
		}

		// Track current capital from open positions
		if opt.Status == "Open" {
			details.CurrentCapital += opt.Capital
		}
	}

	// Calculate averages
	if details.NumberOfOptionsTrades > 0 {
		details.AverageDTE = totalDTE / float64(details.NumberOfOptionsTrades)
	}
	if optionCount > 0 {
		details.AvgOptionReturn = totalReturn / float64(optionCount)
	}

	// Calculate stock metrics
	for _, stock := range stockPositions {
		if stock.Symbol == symbol {
			if stock.Type == "closed" {
				details.TotalStockPL += stock.RealizedPnL
			} else if stock.Type == "open" {
				details.CurrentCapital += stock.CostBasis
			}
		}
	}

	// Calculate total P/L
	details.TotalPL = details.TotalPremiumCollected + details.TotalStockPL

	// Calculate percentage of overall P/L
	if portfolioTotalPL != 0 {
		details.PercentOfOverallPL = (details.TotalPL / portfolioTotalPL) * 100
	}

	// Format all values
	details.TotalPremiumCollectedFormatted = formatCurrencyValue(details.TotalPremiumCollected)
	details.TotalStockPLFormatted = formatCurrencyValue(details.TotalStockPL)
	details.CurrentCapitalFormatted = FormatCurrency(details.CurrentCapital)
	details.TotalPLFormatted = formatCurrencyValue(details.TotalPL)
	details.PercentOfOverallPLFormatted = fmt.Sprintf("%.1f%%", details.PercentOfOverallPL)
	details.AverageDTEFormatted = fmt.Sprintf("%.1f", details.AverageDTE)
	details.AvgOptionReturnFormatted = fmt.Sprintf("%.2f%%", details.AvgOptionReturn)
	details.NumberOfOptionsTradesFormatted = fmt.Sprintf("%d", details.NumberOfOptionsTrades)

	return details
}

// GetStockPositionsBySymbol returns all stock positions (open + closed) for a symbol
func GetStockPositionsBySymbol(symbol string) []Stock {
	transactions := LoadStockTransactions("data/stocks_transactions.csv")
	stockPrices := LoadStockPrices("data/stock_prices.csv")
	positions := CalculateAllPositions(transactions, stockPrices)
	stocks := PositionsToStocks(positions)

	var filtered []Stock
	for _, stock := range stocks {
		if stock.Symbol == symbol {
			filtered = append(filtered, stock)
		}
	}

	return filtered
}

// GetOptionPositionsBySymbol returns all option positions for a symbol
func GetOptionPositionsBySymbol(symbol string) []OptionPosition {
	transactions := LoadOptionTransactions("data/options_transactions.csv")
	positions := CalculateOptionPositions(transactions)

	var filtered []OptionPosition
	for _, pos := range positions {
		if pos.Symbol == symbol {
			filtered = append(filtered, pos)
		}
	}

	return filtered
}

// formatCurrencyValue formats a currency value with sign handling for negative values
func formatCurrencyValue(value float64) string {
	if value < 0 {
		return fmt.Sprintf("-$%.2f", -value)
	}
	return fmt.Sprintf("$%.2f", value)
}
