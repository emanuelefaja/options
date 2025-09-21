package web

import (
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
	ROIPercentage      float64
	AvgReturnPerTrade  float64
	LargestPremium     float64
	SmallestPremium    float64
	AveragePremium     float64
	OptionTradesCount  int
	StockTradesCount   int
	TotalTradesCount   int
	// Portfolio calculations
	TotalDeposits                float64
	TotalStockProfitLoss         float64
	TotalPortfolioValue          float64
	TotalPortfolioProfit         float64
	TotalPortfolioProfitPercentage float64
}

func CalculateAnalytics(trades []Trade, stocks []Stock, transactions []Transaction) Analytics {
	var analytics Analytics
	var earliestDate *time.Time
	var totalReturns float64
	var returnCount int
	var premiumCount int
	analytics.SmallestPremium = 999999 // Initialize to large number
	
	for _, trade := range trades {
		// Parse premium (remove $ and convert to float)
		premium := strings.TrimPrefix(trade.PremiumDollar, "$")
		if p, err := strconv.ParseFloat(premium, 64); err == nil {
			analytics.TotalPremiums += p
			premiumCount++
			
			// Track largest and smallest premiums
			if p > analytics.LargestPremium {
				analytics.LargestPremium = p
			}
			if p < analytics.SmallestPremium {
				analytics.SmallestPremium = p
			}
		}
		
		// Parse capital (remove $ and commas, then convert to float)
		capital := strings.TrimPrefix(trade.Capital, "$")
		capital = strings.ReplaceAll(capital, ",", "")
		if c, err := strconv.ParseFloat(capital, 64); err == nil {
			analytics.TotalCapital += c
			
			// Add to active capital if ongoing
			if trade.Outcome == "Ongoing" {
				analytics.TotalActiveCapital += c
			}
		}
		
		// Parse return percentage for average calculation
		returnStr := strings.TrimSuffix(trade.PercentReturn, "%")
		if r, err := strconv.ParseFloat(returnStr, 64); err == nil {
			totalReturns += r
			returnCount++
		}
		
		// Parse trade date to find the earliest
		if tradeDate, err := time.Parse("January 2 2006", trade.DateOfTrade); err == nil {
			if earliestDate == nil || tradeDate.Before(*earliestDate) {
				earliestDate = &tradeDate
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
	
	// Calculate ROI percentage
	if analytics.TotalCapital > 0 {
		analytics.ROIPercentage = (analytics.TotalPremiums / analytics.TotalCapital) * 100
	}
	
	// Calculate average return per trade
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
	
	// Count trades
	analytics.OptionTradesCount = len(trades)
	analytics.StockTradesCount = len(stocks)
	analytics.TotalTradesCount = analytics.OptionTradesCount + analytics.StockTradesCount
	
	// Calculate total deposits from transactions
	analytics.TotalDeposits = CalculateTotalDeposits(transactions)
	
	// Calculate total stock profit/loss from all positions
	// Load stock transactions to get closed positions
	stockTransactions := LoadStockTransactions("data/stocks_transactions.csv")
	if len(stockTransactions) > 0 {
		positions := CalculateAllPositions(stockTransactions)
		for _, pos := range positions {
			if pos.Type == "closed" {
				analytics.TotalStockProfitLoss += pos.RealizedPnL
			}
		}
	}
	
	// Calculate portfolio totals
	analytics.TotalPortfolioValue = analytics.TotalDeposits + analytics.TotalPremiums + analytics.TotalStockProfitLoss
	analytics.TotalPortfolioProfit = analytics.TotalPremiums + analytics.TotalStockProfitLoss
	
	// Calculate portfolio profit percentage
	if analytics.TotalDeposits > 0 {
		analytics.TotalPortfolioProfitPercentage = (analytics.TotalPortfolioProfit / analytics.TotalDeposits) * 100
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