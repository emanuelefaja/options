package web

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
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
	DaysSinceStart     int
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
	DailyTheta                   float64
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

type StockPerformance struct {
	WinRate              float64
	WinCount             int
	LossCount            int
	TotalClosedCount     int
	AvgWin               float64
	AvgLoss              float64
	WinRateFormatted     string
	AvgWinFormatted      string
	AvgLossFormatted     string
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

			// Calculate Daily Theta for open positions
			if pos.DaysToExpiry > 0 {
				analytics.DailyTheta += pos.NetPremium / float64(pos.DaysToExpiry)
			}
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
	stockPrices := LoadStockPrices("data/stock_prices.csv")
	if len(stockTransactions) > 0 {
		positions := CalculateAllPositions(stockTransactions, stockPrices)
		openStockCount := 0
		closedStockCount := 0

		// Find earliest stock transaction date for days since start
		var earliestStockDate *time.Time
		for _, txn := range stockTransactions {
			if txnDate, err := time.Parse("2006-01-02", txn.Date); err == nil {
				if earliestStockDate == nil || txnDate.Before(*earliestStockDate) {
					earliestStockDate = &txnDate
				}
			}
		}

		// Calculate days since start
		if earliestStockDate != nil {
			analytics.DaysSinceStart = int(time.Since(*earliestStockDate).Hours() / 24)
		}

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
	// Handle negative numbers
	isNegative := amount < 0
	if isNegative {
		amount = -amount
	}

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

	result := "$" + strings.Join(parts, ",")
	if isNegative {
		result = "-" + result
	}

	return result
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
	stockPrices := LoadStockPrices("data/stock_prices.csv")
	positions := CalculateAllPositions(stockTransactions, stockPrices)
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

func CalculateStockPerformance(stockTransactions []StockTransaction) StockPerformance {
	stockPrices := LoadStockPrices("data/stock_prices.csv")
	positions := CalculateAllPositions(stockTransactions, stockPrices)

	var perf StockPerformance
	var totalWins float64
	var totalLosses float64

	for _, pos := range positions {
		if pos.Type == "closed" {
			perf.TotalClosedCount++

			if pos.RealizedPnL > 0 {
				perf.WinCount++
				totalWins += pos.RealizedPnL
			} else if pos.RealizedPnL < 0 {
				perf.LossCount++
				totalLosses += pos.RealizedPnL
			}
		}
	}

	// Calculate win rate
	if perf.TotalClosedCount > 0 {
		perf.WinRate = (float64(perf.WinCount) / float64(perf.TotalClosedCount)) * 100
	}

	// Calculate average win
	if perf.WinCount > 0 {
		perf.AvgWin = totalWins / float64(perf.WinCount)
	}

	// Calculate average loss
	if perf.LossCount > 0 {
		perf.AvgLoss = totalLosses / float64(perf.LossCount)
	}

	// Format values
	perf.WinRateFormatted = FormatPercentage(perf.WinRate)
	perf.AvgWinFormatted = FormatCurrency(perf.AvgWin)
	perf.AvgLossFormatted = FormatCurrency(perf.AvgLoss)

	return perf
}

// CalculatePortfolioValueAsOf calculates the portfolio value as of a specific date
// It includes: deposits + options premiums + realized stock P&L (only counting transactions up to the date)
func CalculatePortfolioValueAsOf(asOfDate time.Time) float64 {
	// Load all data sources
	transactions := LoadTransactionsFromCSV("data/transactions.csv")
	optionTransactions := LoadOptionTransactions("data/options_transactions.csv")
	stockTransactions := LoadStockTransactions("data/stocks_transactions.csv")

	var portfolioValue float64

	// 1. Calculate deposits up to this date
	for _, t := range transactions {
		txDate, err := time.Parse("January 2 2006", t.Date)
		if err != nil {
			continue
		}
		if !txDate.After(asOfDate) && t.Type == "Deposit" {
			amount := strings.TrimPrefix(t.Amount, "$")
			amount = strings.ReplaceAll(amount, ",", "")
			if a, err := strconv.ParseFloat(amount, 64); err == nil {
				portfolioValue += a
			}
		}
	}

	// 2. Calculate options premiums for positions opened by this date
	// Filter option transactions up to the date
	var filteredOptionTxns []OptionTransaction
	for _, tx := range optionTransactions {
		txDate, err := time.Parse("2006-01-02", tx.Date)
		if err != nil {
			continue
		}
		if !txDate.After(asOfDate) {
			filteredOptionTxns = append(filteredOptionTxns, tx)
		}
	}

	// Calculate positions from filtered transactions
	optionPositions := CalculateOptionPositions(filteredOptionTxns)
	for _, pos := range optionPositions {
		// Only count net premiums (collected - paid - commissions)
		portfolioValue += pos.NetPremium
	}

	// 3. Calculate realized stock P&L from sales by this date
	// Filter stock transactions up to the date
	var filteredStockTxns []StockTransaction
	for _, tx := range stockTransactions {
		txDate, err := time.Parse("2006-01-02", tx.Date)
		if err != nil {
			continue
		}
		if !txDate.After(asOfDate) {
			filteredStockTxns = append(filteredStockTxns, tx)
		}
	}

	// Calculate positions from filtered transactions - only count closed positions
	stockPrices := make(map[string]float64) // Empty map since we only need realized P&L
	positions := CalculateAllPositions(filteredStockTxns, stockPrices)
	for _, pos := range positions {
		if pos.Type == "closed" {
			portfolioValue += pos.RealizedPnL
		}
	}

	return portfolioValue
}

func CalculateNetWorth(totalPortfolioValue float64) []NetWorthMonth {
	// Load wise.csv
	file, err := os.Open("data/wise.csv")
	if err != nil {
		return []NetWorthMonth{}
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return []NetWorthMonth{}
	}

	var netWorthData []NetWorthMonth

	// Get current time for comparison
	now := time.Now()
	currentMonth := now.Format("2006-01")

	// Skip header and process each row
	for i, record := range records {
		if i == 0 {
			continue // Skip header
		}

		if len(record) < 2 {
			continue
		}

		month := record[0]
		savingsBalance, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			continue
		}

		var brokerageBalance float64

		// If this is the current month, use live portfolio value
		// Otherwise, calculate historical value as of end of month
		if month == currentMonth {
			brokerageBalance = totalPortfolioValue
		} else {
			// Parse month and get last day of that month
			monthDate, err := time.Parse("2006-01", month)
			if err != nil {
				continue
			}
			// Get the last day of the month
			endOfMonth := time.Date(monthDate.Year(), monthDate.Month()+1, 0, 23, 59, 59, 0, time.UTC)

			// Calculate portfolio value as of that date
			brokerageBalance = CalculatePortfolioValueAsOf(endOfMonth)
		}

		netWorthData = append(netWorthData, NetWorthMonth{
			Month:            month,
			SavingsBalance:   savingsBalance,
			BrokerageBalance: brokerageBalance,
			TotalNetWorth:    savingsBalance + brokerageBalance,
		})
	}

	return netWorthData
}

func CalculateCashPosition(analytics Analytics) CashPosition {
	// Active Capital: Money currently tied up in positions
	activeCapital := analytics.TotalActiveCapital

	// Dry Powder: Available cash in brokerage
	// = Total Deposits + Premiums Earned + Stock P/L - Active Capital
	dryPowder := analytics.TotalDeposits + analytics.TotalPremiums + analytics.TotalStockProfitLoss - activeCapital

	// Wise Balance: Get latest balance from wise.csv
	wiseBalance := 0.0
	file, err := os.Open("data/wise.csv")
	if err == nil {
		defer file.Close()
		reader := csv.NewReader(file)
		records, err := reader.ReadAll()
		if err == nil && len(records) > 1 {
			// Get the last row (most recent month)
			lastRow := records[len(records)-1]
			if len(lastRow) >= 2 {
				wiseBalance, _ = strconv.ParseFloat(lastRow[1], 64)
			}
		}
	}

	return CashPosition{
		ActiveCapital: activeCapital,
		DryPowder:     dryPowder,
		WiseBalance:   wiseBalance,
	}
}

func LoadVIX(filePath string) float64 {
	file, err := os.Open(filePath)
	if err != nil {
		return 0.0
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil || len(records) < 2 {
		return 0.0
	}

	// Get the last row (most recent VIX value)
	lastRow := records[len(records)-1]
	if len(lastRow) >= 2 {
		vix, err := strconv.ParseFloat(lastRow[1], 64)
		if err == nil {
			return vix
		}
	}

	return 0.0
}