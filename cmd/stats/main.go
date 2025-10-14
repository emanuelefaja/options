package main

import (
	"fmt"
	"mnmlsm/web"
	"sort"
)

func main() {
	// Load all data
	transactions := web.LoadTransactionsFromCSV("data/transactions.csv")
	analytics := web.CalculateAnalytics(nil, nil, transactions)
	cashPosition := web.CalculateCashPosition(analytics)
	sectorExposure := web.CalculateSectorExposure()
	positionDetails := web.CalculatePositionDetails()

	// Calculate unrealized P&L
	stockTransactions := web.LoadStockTransactions("data/stocks_transactions.csv")
	stockPrices := web.LoadStockPrices("data/stock_prices.csv")
	positions := web.CalculateAllPositions(stockTransactions, stockPrices)

	totalUnrealizedPL := 0.0
	for _, pos := range positions {
		if pos.Type == "open" {
			totalUnrealizedPL += pos.UnrealizedPnL
		}
	}

	// Calculate weekly return
	weeklyPL := 0.0
	weeklyReturnPercent := 0.0
	if cashPosition.ActiveCapital > 0 {
		// Get last 7 days of returns
		daysToCheck := 7
		if daysToCheck > len(analytics.DailyReturns) {
			daysToCheck = len(analytics.DailyReturns)
		}

		for i := len(analytics.DailyReturns) - daysToCheck; i < len(analytics.DailyReturns); i++ {
			weeklyPL += analytics.DailyReturns[i].TotalReturns
		}

		weeklyReturnPercent = (weeklyPL / cashPosition.ActiveCapital) * 100
	}

	vix := web.LoadVIX("data/vix.csv")

	// Calculate total capital at risk (includes deposits)
	totalCapital := analytics.TotalDeposits + analytics.TotalPremiums + analytics.TotalStockProfitLoss
	capitalUtilization := 0.0
	if totalCapital > 0 {
		capitalUtilization = (cashPosition.ActiveCapital / totalCapital) * 100
	}

	// Print output
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("                 PORTFOLIO OVERVIEW")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Total Portfolio Value:     %s\n", web.FormatCurrency(analytics.TotalPortfolioValue))
	fmt.Printf("Total Profit/Loss:         %s\n", formatPL(analytics.TotalPortfolioProfit))
	fmt.Printf("Portfolio Return:          %s\n", web.FormatPercentage(analytics.TotalPortfolioProfitPercentage))
	fmt.Printf("Total Deposits:            %s\n", web.FormatCurrency(analytics.TotalDeposits))
	fmt.Printf("Days Active:               %d\n", analytics.DaysSinceStart)
	fmt.Printf("Time-Weighted Return:      %s (Ann: %s)\n",
		web.FormatPercentage(analytics.TimeWeightedReturn),
		web.FormatPercentage(analytics.TimeWeightedReturnAnnualized))
	fmt.Println()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("                 ANALYTICS METRICS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Total Premiums Collected:  %s\n", web.FormatCurrency(analytics.TotalPremiums))
	fmt.Printf("Total Stock P/L:           %s\n", formatPL(analytics.TotalStockProfitLoss))
	fmt.Printf("Daily Theta:               %s\n", web.FormatCurrency(analytics.DailyTheta))
	fmt.Printf("Premium Per Day:           %s\n", web.FormatCurrency(analytics.PremiumPerDay))
	fmt.Printf("Number of Option Trades:   %d\n", analytics.OptionTradesCount)
	fmt.Printf("Number of Stock Trades:    %d\n", analytics.StockTradesCount)
	fmt.Printf("Total Number of Trades:    %d\n", analytics.TotalTradesCount)
	fmt.Printf("Avg Return Per Option:     %s\n", web.FormatPercentage(analytics.AvgReturnPerTrade))
	fmt.Printf("Largest Premium:           %s\n", web.FormatCurrency(analytics.LargestPremium))
	fmt.Printf("Smallest Premium:          %s\n", web.FormatCurrency(analytics.SmallestPremium))
	fmt.Printf("Average Premium:           %s\n", web.FormatCurrency(analytics.AveragePremium))
	fmt.Println()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("                   RISK METRICS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("At Risk Capital:           %s (%.1f%% of %s)\n",
		web.FormatCurrency(cashPosition.ActiveCapital),
		capitalUtilization,
		web.FormatCurrency(totalCapital))

	// Weekly return status
	weeklyStatus := "✓ On Track"
	if weeklyReturnPercent < 1.0 {
		weeklyStatus = "⚠ Below Target"
	}
	fmt.Printf("Weekly Return Rate:        %s %s (%s)\n",
		weeklyStatus,
		web.FormatPercentage(weeklyReturnPercent),
		formatPL(weeklyPL))

	// Unrealized P/L status
	unrealizedStatus := "✓ Risk Compliant"
	unrealizedPercent := 0.0
	if cashPosition.ActiveCapital > 0 {
		unrealizedPercent = (totalUnrealizedPL / cashPosition.ActiveCapital) * 100
		if unrealizedPercent < -5 {
			unrealizedStatus = "⚠ Approaching Limit"
		}
		if unrealizedPercent < -10 {
			unrealizedStatus = "✗ Limit Exceeded"
		}
	}
	fmt.Printf("Unrealized P/L:            %s %s (%.1f%% of at risk)\n",
		unrealizedStatus,
		formatPL(totalUnrealizedPL),
		unrealizedPercent)

	fmt.Printf("Available Cash (Dry):      %s\n", web.FormatCurrency(cashPosition.DryPowder))
	fmt.Printf("Wise Balance:              %s\n", web.FormatCurrency(cashPosition.WiseBalance))
	fmt.Printf("VIX:                       %.2f\n", vix)
	fmt.Println()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("                 SECTOR EXPOSURE")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if len(sectorExposure) > 0 {
		for _, sector := range sectorExposure {
			sectorPercent := 0.0
			if totalCapital > 0 {
				sectorPercent = (sector.Amount / totalCapital) * 100
			}
			fmt.Printf("%-25s %s (%.1f%%)\n",
				sector.Sector,
				web.FormatCurrency(sector.Amount),
				sectorPercent)
		}
	} else {
		fmt.Println("No active positions")
	}
	fmt.Println()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("                 POSITION DETAILS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if len(positionDetails) > 0 {
		// Group by symbol
		symbolMap := make(map[string][]web.PositionDetail)
		for _, pos := range positionDetails {
			symbolMap[pos.Symbol] = append(symbolMap[pos.Symbol], pos)
		}

		// Sort symbols by total amount
		type symbolTotal struct {
			symbol string
			total  float64
		}
		var symbolTotals []symbolTotal
		for symbol, positions := range symbolMap {
			total := 0.0
			for _, pos := range positions {
				total += pos.Amount
			}
			symbolTotals = append(symbolTotals, symbolTotal{symbol, total})
		}
		sort.Slice(symbolTotals, func(i, j int) bool {
			return symbolTotals[i].total > symbolTotals[j].total
		})

		// Print top 10 positions
		displayCount := 10
		if displayCount > len(symbolTotals) {
			displayCount = len(symbolTotals)
		}

		for i := 0; i < displayCount; i++ {
			st := symbolTotals[i]
			positions := symbolMap[st.symbol]

			posPercent := 0.0
			if totalCapital > 0 {
				posPercent = (st.total / totalCapital) * 100
			}

			// Build type string
			typeStr := ""
			for j, pos := range positions {
				if j > 0 {
					typeStr += " + "
				}
				typeStr += pos.Type
			}

			status := ""
			if posPercent > 10 {
				status = "⚠"
			}

			fmt.Printf("%-6s %-15s %s (%.1f%%) %s\n",
				st.symbol,
				typeStr,
				web.FormatCurrency(st.total),
				posPercent,
				status)
		}

		if len(symbolTotals) > displayCount {
			fmt.Printf("\n... and %d more positions\n", len(symbolTotals)-displayCount)
		}
	} else {
		fmt.Println("No active positions")
	}
	fmt.Println()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func formatPL(value float64) string {
	formatted := web.FormatCurrency(value)
	if value < 0 {
		return formatted // Already has minus sign
	}
	return "+" + formatted
}
