package web

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Analytics struct {
	TotalPremiums      float64
	TotalCapital       float64
	TotalActiveCapital float64
	PremiumPerDay      float64
	AvgReturnPerTrade  float64
	LargestPremium     float64
	SmallestPremium    float64
	AveragePremium     float64
	OptionTradesCount  int
	StockTradesCount   int
	TotalTradesCount   int
	// Options specific calculations
	OpenOptionsCount   int
	ClosedOptionsCount int
	OptionsActiveCapital float64
	CollectedPremiums  float64
	// Portfolio calculations
	TotalDeposits                float64
	TotalStockProfitLoss         float64
	TotalPortfolioValue          float64
	TotalPortfolioProfit         float64
	TotalPortfolioProfitPercentage float64
	// Daily returns data
	DailyReturns       []DailyReturn
	DailyReturnsJSON   string
}

type DailyReturn struct {
	Date          string  `json:"date"`
	Premiums      float64 `json:"premiums"`
	StockGains    float64 `json:"stockGains"`
	TotalReturns  float64 `json:"totalReturns"`
}

func CalculateAnalytics(trades []Trade, stocks []Stock, transactions []Transaction) Analytics {
	// Load and calculate option positions from new transaction system
	optionTransactions := LoadOptionTransactions("data/options_transactions.csv")
	optionPositions := CalculateOptionPositions(optionTransactions)

	var analytics Analytics
	var earliestDate *time.Time
	var totalReturns float64
	var returnCount int
	var premiumCount int
	analytics.SmallestPremium = 999999 // Initialize to large number

	// Process option positions instead of trades
	for _, pos := range optionPositions {
		// Count open vs closed options
		if pos.Status == "Open" {
			analytics.OpenOptionsCount++
		} else {
			analytics.ClosedOptionsCount++
		}

		// Track premiums
		netPremium := pos.NetPremium
		if netPremium > 0 {
			analytics.TotalPremiums += netPremium
			analytics.CollectedPremiums += pos.PremiumCollected
			premiumCount++

			// Track largest and smallest premiums
			if netPremium > analytics.LargestPremium {
				analytics.LargestPremium = netPremium
			}
			if netPremium < analytics.SmallestPremium {
				analytics.SmallestPremium = netPremium
			}
		}

		// Track capital
		if pos.Capital > 0 {
			analytics.TotalCapital += pos.Capital

			// Add to options active capital for open options
			if pos.Status == "Open" {
				analytics.OptionsActiveCapital += pos.Capital

				// Add to active capital only for open Puts (cash-secured puts)
				// Calls are covered calls, so that capital is already in stock positions
				if pos.OptionType == "Put" {
					analytics.TotalActiveCapital += pos.Capital
				}
			}
		}

		// Track returns
		if pos.PercentReturn != 0 {
			totalReturns += pos.PercentReturn
			returnCount++
		}

		// Parse position open date to find the earliest
		if openDate, err := time.Parse("2006-01-02", pos.OpenDate); err == nil {
			if earliestDate == nil || openDate.Before(*earliestDate) {
				earliestDate = &openDate
			}
		}
	}

	// Calculate premium per day
	if earliestDate != nil {
		daysSinceFirst := time.Since(*earliestDate).Hours() / 24
		if daysSinceFirst > 0 {
			analytics.PremiumPerDay = analytics.TotalPremiums / daysSinceFirst
		}
	}

	// Calculate average return per position
	if returnCount > 0 {
		analytics.AvgReturnPerTrade = totalReturns / float64(returnCount)
	}

	// Calculate average premium
	if premiumCount > 0 {
		analytics.AveragePremium = analytics.TotalPremiums / float64(premiumCount)
	}

	// Handle case where no premiums were found
	if analytics.SmallestPremium == 999999 {
		analytics.SmallestPremium = 0
	}

	// Count option positions (not raw trades)
	analytics.OptionTradesCount = len(optionPositions)

	// Calculate total deposits from transactions
	analytics.TotalDeposits = CalculateTotalDeposits(transactions)

	// Calculate total stock profit/loss from all positions
	// Load stock transactions to get closed positions and open positions' cost basis
	stockTransactions := LoadStockTransactions("data/stocks_transactions.csv")
	if len(stockTransactions) > 0 {
		positions := CalculateAllPositions(stockTransactions)
		openStockCount := 0
		closedStockCount := 0
		for _, pos := range positions {
			if pos.Type == "closed" {
				analytics.TotalStockProfitLoss += pos.RealizedPnL
				closedStockCount++
			} else if pos.Type == "open" {
				// Add open stock positions' cost basis to active capital
				analytics.TotalActiveCapital += pos.CostBasis
				openStockCount++
			}
		}
		// Count total stock trades as open + closed positions
		analytics.StockTradesCount = openStockCount + closedStockCount
	}

	analytics.TotalTradesCount = analytics.OptionTradesCount + analytics.StockTradesCount
	
	// Calculate portfolio totals
	analytics.TotalPortfolioValue = analytics.TotalDeposits + analytics.TotalPremiums + analytics.TotalStockProfitLoss
	analytics.TotalPortfolioProfit = analytics.TotalPremiums + analytics.TotalStockProfitLoss
	
	// Calculate portfolio profit percentage
	if analytics.TotalDeposits > 0 {
		analytics.TotalPortfolioProfitPercentage = (analytics.TotalPortfolioProfit / analytics.TotalDeposits) * 100
	}
	
	// Calculate daily returns
	analytics.DailyReturns = CalculateDailyReturnsNew(optionPositions, stockTransactions)
	
	// Convert daily returns to JSON for use in JavaScript
	if analytics.DailyReturns == nil {
		analytics.DailyReturns = []DailyReturn{}
	}
	
	jsonData, err := json.Marshal(analytics.DailyReturns)
	if err != nil {
		analytics.DailyReturnsJSON = "[]"
	} else {
		analytics.DailyReturnsJSON = string(jsonData)
	}
	
	return analytics
}

func FormatPercentage(value float64) string {
	return fmt.Sprintf("%.2f%%", value)
}

func FormatCurrency(amount float64) string {
	// Format with commas and no decimal places
	formatted := fmt.Sprintf("%.0f", amount)
	
	// Add commas
	parts := []string{}
	for i := len(formatted); i > 0; i -= 3 {
		start := i - 3
		if start < 0 {
			start = 0
		}
		parts = append([]string{formatted[start:i]}, parts...)
	}
	
	return "$" + strings.Join(parts, ",")
}

func CalculateDailyReturnsNew(optionPositions []OptionPosition, stockTransactions []StockTransaction) []DailyReturn {
	dailyMap := make(map[string]*DailyReturn)

	// Process option positions
	for _, pos := range optionPositions {
		// Use open date for premium collection
		if pos.OpenDate != "" {
			dateStr := pos.OpenDate

			if _, exists := dailyMap[dateStr]; !exists {
				dailyMap[dateStr] = &DailyReturn{
					Date: dateStr,
				}
			}

			// Add net premium to the open date
			dailyMap[dateStr].Premiums += pos.NetPremium
		}
	}

	// Process stock transactions for realized gains
	positions := CalculateAllPositions(stockTransactions)
	for _, pos := range positions {
		if pos.Type == "closed" {
			// Use the close date (sell date) for realized gains
			dateStr := pos.CloseDate
			if parsedDate, err := time.Parse("2006-01-02", pos.CloseDate); err == nil {
				dateStr = parsedDate.Format("2006-01-02")
			}

			if _, exists := dailyMap[dateStr]; !exists {
				dailyMap[dateStr] = &DailyReturn{
					Date: dateStr,
				}
			}

			// Use the already calculated realized P&L from the position
			dailyMap[dateStr].StockGains += pos.RealizedPnL
		}
	}

	// Convert map to sorted slice
	var dailyReturns []DailyReturn
	for _, dr := range dailyMap {
		dr.TotalReturns = dr.Premiums + dr.StockGains
		dailyReturns = append(dailyReturns, *dr)
	}

	// Sort by date
	for i := 0; i < len(dailyReturns)-1; i++ {
		for j := i + 1; j < len(dailyReturns); j++ {
			if dailyReturns[i].Date > dailyReturns[j].Date {
				dailyReturns[i], dailyReturns[j] = dailyReturns[j], dailyReturns[i]
			}
		}
	}

	return dailyReturns
}

// Keep the old function for backward compatibility
func CalculateDailyReturns(trades []Trade, stockTransactions []StockTransaction) []DailyReturn {
	dailyMap := make(map[string]*DailyReturn)
	
	// Process option premiums
	for _, trade := range trades {
		// Parse trade date
		tradeDate, err := time.Parse("January 2 2006", trade.DateOfTrade)
		if err != nil {
			continue
		}
		
		dateStr := tradeDate.Format("2006-01-02")
		
		// Parse premium
		premium := strings.TrimPrefix(trade.PremiumDollar, "$")
		premiumValue, err := strconv.ParseFloat(premium, 64)
		if err != nil {
			continue
		}
		
		// Initialize or update daily return
		if _, exists := dailyMap[dateStr]; !exists {
			dailyMap[dateStr] = &DailyReturn{
				Date: dateStr,
			}
		}
		
		dailyMap[dateStr].Premiums += premiumValue
	}
	
	// Process stock transactions for realized gains
	positions := CalculateAllPositions(stockTransactions)
	for _, pos := range positions {
		if pos.Type == "closed" {
			// Use the close date (sell date) for realized gains
			dateStr := pos.CloseDate
			if parsedDate, err := time.Parse("2006-01-02", pos.CloseDate); err == nil {
				dateStr = parsedDate.Format("2006-01-02")
			}
			
			if _, exists := dailyMap[dateStr]; !exists {
				dailyMap[dateStr] = &DailyReturn{
					Date: dateStr,
				}
			}
			
			// Use the already calculated realized P&L from the position
			dailyMap[dateStr].StockGains += pos.RealizedPnL
		}
	}
	
	// Convert map to sorted slice
	var dailyReturns []DailyReturn
	for _, dr := range dailyMap {
		dr.TotalReturns = dr.Premiums + dr.StockGains
		dailyReturns = append(dailyReturns, *dr)
	}
	
	// Sort by date
	for i := 0; i < len(dailyReturns)-1; i++ {
		for j := i + 1; j < len(dailyReturns); j++ {
			if dailyReturns[i].Date > dailyReturns[j].Date {
				dailyReturns[i], dailyReturns[j] = dailyReturns[j], dailyReturns[i]
			}
		}
	}
	
	return dailyReturns
}