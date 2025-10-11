package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"mnmlsm/web"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	mux := http.NewServeMux()

	// Static files
	fileServer := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// Routes
	mux.HandleFunc("/", handleHome)
	mux.HandleFunc("/options", handleOptions)
	mux.HandleFunc("/stocks", handleStocks)
	mux.HandleFunc("/stocks/", handleStockPages)
	mux.HandleFunc("/analytics", handleAnalytics)
	mux.HandleFunc("/risk", handleRisk)
	mux.HandleFunc("/rules", handleRules)

	log.Println("Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	transactions := web.LoadTransactionsFromCSV("data/transactions.csv")
	analytics := web.CalculateAnalytics(nil, nil, transactions)

	// Calculate stock performance metrics
	stockTransactions := web.LoadStockTransactions("data/stocks_transactions.csv")
	stockPerformance := web.CalculateStockPerformance(stockTransactions)

	// Calculate cash position
	cashPosition := web.CalculateCashPosition(analytics)
	cashPositionJSON := "[]"
	if jsonData, err := json.Marshal(cashPosition); err == nil {
		cashPositionJSON = string(jsonData)
	}

	totalUnrealizedPL := calculateTotalUnrealizedPL()
	vix := web.LoadVIX("data/vix.csv")

	renderPage(w, "home", web.PageData{
		Title:       "Home - mnmlsm",
		CurrentPage: "home",
		// Portfolio values for header
		TotalPortfolioValue:                     analytics.TotalPortfolioValue,
		TotalPortfolioProfit:                    analytics.TotalPortfolioProfit,
		TotalPortfolioProfitPercentage:          analytics.TotalPortfolioProfitPercentage,
		TotalUnrealizedPL:                       totalUnrealizedPL,
		TotalPortfolioValueFormatted:            web.FormatCurrency(analytics.TotalPortfolioValue),
		TotalPortfolioProfitFormatted:           web.FormatCurrency(analytics.TotalPortfolioProfit),
		TotalPortfolioProfitPercentageFormatted: web.FormatPercentage(analytics.TotalPortfolioProfitPercentage),
		TotalUnrealizedPLFormatted:              web.FormatCurrency(totalUnrealizedPL),
		VIX:                                     vix,
		VIXFormatted:                            fmt.Sprintf("%.2f", vix),
		// Stock performance metrics
		StockPerformance: stockPerformance,
		// Cash position data
		CashPosition:     cashPosition,
		CashPositionJSON: cashPositionJSON,
	})
}

func handleOptions(w http.ResponseWriter, r *http.Request) {
	// Load option positions from new transaction system
	optionTransactions := web.LoadOptionTransactions("data/options_transactions.csv")
	optionPositions := web.CalculateOptionPositions(optionTransactions)

	transactions := web.LoadTransactionsFromCSV("data/transactions.csv")
	analytics := web.CalculateAnalytics(nil, nil, transactions)

	totalUnrealizedPL := calculateTotalUnrealizedPL()
	vix := web.LoadVIX("data/vix.csv")

	renderPage(w, "options", web.PageData{
		Title:           "Options - mnmlsm",
		CurrentPage:     "options",
		OptionPositions: optionPositions,
		// Options page specific metrics
		OpenOptionsCount:     analytics.OpenOptionsCount,
		ClosedOptionsCount:   analytics.ClosedOptionsCount,
		OptionsActiveCapital: analytics.OptionsActiveCapital,
		TotalPremiums:        analytics.TotalPremiums,
		OptionsActiveCapitalFormatted: web.FormatCurrency(analytics.OptionsActiveCapital),
		TotalPremiumsFormatted:        web.FormatCurrency(analytics.TotalPremiums),
		// Portfolio values for header
		TotalPortfolioValue:          analytics.TotalPortfolioValue,
		TotalPortfolioProfit:         analytics.TotalPortfolioProfit,
		TotalPortfolioProfitPercentage: analytics.TotalPortfolioProfitPercentage,
		TotalUnrealizedPL:            totalUnrealizedPL,
		TotalPortfolioValueFormatted:  web.FormatCurrency(analytics.TotalPortfolioValue),
		TotalPortfolioProfitFormatted: web.FormatCurrency(analytics.TotalPortfolioProfit),
		TotalPortfolioProfitPercentageFormatted: web.FormatPercentage(analytics.TotalPortfolioProfitPercentage),
		TotalUnrealizedPLFormatted:   web.FormatCurrency(totalUnrealizedPL),
		VIX:                          vix,
		VIXFormatted:                 fmt.Sprintf("%.2f", vix),
	})
}

func handleStocks(w http.ResponseWriter, r *http.Request) {
	// Only handle exact /stocks path
	if r.URL.Path != "/stocks" {
		http.NotFound(w, r)
		return
	}

	// Load stock positions from transaction system
	stockTransactions := web.LoadStockTransactions("data/stocks_transactions.csv")
	stockPrices := web.LoadStockPrices("data/stock_prices.csv")
	stockPositions := web.CalculateAllPositions(stockTransactions, stockPrices)
	allStocks := web.PositionsToStocks(stockPositions)

	// Separate open and closed positions
	var currentStocks, closedStocks []web.Stock
	for _, stock := range allStocks {
		if stock.ExitDate == "" {
			currentStocks = append(currentStocks, stock)
		} else {
			closedStocks = append(closedStocks, stock)
		}
	}

	totalUnrealizedPL := calculateTotalUnrealizedPL()

	transactions := web.LoadTransactionsFromCSV("data/transactions.csv")
	analytics := web.CalculateAnalytics(nil, nil, transactions)

	// Load symbol summaries
	symbolSummaries := web.CalculateSymbolSummaries()
	vix := web.LoadVIX("data/vix.csv")

	renderPage(w, "stocks/index", web.PageData{
		Title:           "Stocks - mnmlsm",
		CurrentPage:     "stocks",
		Stocks:          currentStocks,
		ClosedStocks:    closedStocks,
		SymbolSummaries: symbolSummaries,
		// Portfolio values for header
		TotalPortfolioValue:                     analytics.TotalPortfolioValue,
		TotalPortfolioProfit:                    analytics.TotalPortfolioProfit,
		TotalPortfolioProfitPercentage:          analytics.TotalPortfolioProfitPercentage,
		TotalUnrealizedPL:                       totalUnrealizedPL,
		TotalPortfolioValueFormatted:            web.FormatCurrency(analytics.TotalPortfolioValue),
		TotalPortfolioProfitFormatted:           web.FormatCurrency(analytics.TotalPortfolioProfit),
		TotalPortfolioProfitPercentageFormatted: web.FormatPercentage(analytics.TotalPortfolioProfitPercentage),
		TotalUnrealizedPLFormatted:              web.FormatCurrency(totalUnrealizedPL),
		VIX:                                     vix,
		VIXFormatted:                            fmt.Sprintf("%.2f", vix),
	})
}

func handleStockPages(w http.ResponseWriter, r *http.Request) {
	// Extract symbol from URL (e.g., /stocks/AMD -> AMD)
	symbol := strings.ToUpper(strings.TrimPrefix(r.URL.Path, "/stocks/"))

	if symbol == "" {
		http.NotFound(w, r)
		return
	}

	// Load analytics for portfolio-wide metrics
	transactions := web.LoadTransactionsFromCSV("data/transactions.csv")
	analytics := web.CalculateAnalytics(nil, nil, transactions)

	// Get symbol-specific data
	symbolDetails := web.GetSymbolDetails(symbol, analytics.TotalPortfolioProfit)
	symbolStocks := web.GetStockPositionsBySymbol(symbol)
	symbolOptions := web.GetOptionPositionsBySymbol(symbol)

	// Return 404 if no data exists for this symbol
	if len(symbolStocks) == 0 && len(symbolOptions) == 0 {
		http.NotFound(w, r)
		return
	}

	totalUnrealizedPL := calculateTotalUnrealizedPL()
	vix := web.LoadVIX("data/vix.csv")

	renderPage(w, "stocks/detail", web.PageData{
		Title:          symbol + " - Stock Detail - mnmlsm",
		CurrentPage:    "stocks",
		Symbol:         symbol,
		SymbolDetails:  symbolDetails,
		SymbolStocks:   symbolStocks,
		SymbolOptions:  symbolOptions,
		// Portfolio values for header
		TotalPortfolioValue:                     analytics.TotalPortfolioValue,
		TotalPortfolioProfit:                    analytics.TotalPortfolioProfit,
		TotalPortfolioProfitPercentage:          analytics.TotalPortfolioProfitPercentage,
		TotalUnrealizedPL:                       totalUnrealizedPL,
		TotalPortfolioValueFormatted:            web.FormatCurrency(analytics.TotalPortfolioValue),
		TotalPortfolioProfitFormatted:           web.FormatCurrency(analytics.TotalPortfolioProfit),
		TotalPortfolioProfitPercentageFormatted: web.FormatPercentage(analytics.TotalPortfolioProfitPercentage),
		TotalUnrealizedPLFormatted:              web.FormatCurrency(totalUnrealizedPL),
		VIX:                                     vix,
		VIXFormatted:                            fmt.Sprintf("%.2f", vix),
	})
}

func handleAnalytics(w http.ResponseWriter, r *http.Request) {
	transactions := web.LoadTransactionsFromCSV("data/transactions.csv")
	analytics := web.CalculateAnalytics(nil, nil, transactions)

	// Calculate net worth data
	netWorthData := web.CalculateNetWorth(analytics.TotalPortfolioValue)
	netWorthJSON := "[]"
	if len(netWorthData) > 0 {
		if jsonData, err := json.Marshal(netWorthData); err == nil {
			netWorthJSON = string(jsonData)
		}
	}

	totalUnrealizedPL := calculateTotalUnrealizedPL()
	vix := web.LoadVIX("data/vix.csv")

	renderPage(w, "analytics", web.PageData{
		Title:              "Analytics - Options Tracker",
		CurrentPage:        "analytics",
		TotalPremiums:      analytics.TotalPremiums,
		TotalCapital:       analytics.TotalCapital,
		TotalActiveCapital: analytics.TotalActiveCapital,
		PremiumPerDay:      analytics.PremiumPerDay,
		AvgReturnPerTrade:  analytics.AvgReturnPerTrade,
		LargestPremium:     analytics.LargestPremium,
		SmallestPremium:    analytics.SmallestPremium,
		AveragePremium:     analytics.AveragePremium,
		OptionTradesCount:  analytics.OptionTradesCount,
		StockTradesCount:   analytics.StockTradesCount,
		TotalTradesCount:   analytics.TotalTradesCount,
		DaysSinceStart:     analytics.DaysSinceStart,
		TotalPremiumsFormatted:      web.FormatCurrency(analytics.TotalPremiums),
		TotalCapitalFormatted:       web.FormatCurrency(analytics.TotalCapital),
		TotalActiveCapitalFormatted: web.FormatCurrency(analytics.TotalActiveCapital),
		PremiumPerDayFormatted:      web.FormatCurrency(analytics.PremiumPerDay),
		AvgReturnPerTradeFormatted:  web.FormatPercentage(analytics.AvgReturnPerTrade),
		LargestPremiumFormatted:     web.FormatCurrency(analytics.LargestPremium),
		SmallestPremiumFormatted:    web.FormatCurrency(analytics.SmallestPremium),
		AveragePremiumFormatted:     web.FormatCurrency(analytics.AveragePremium),
		// Portfolio values for header
		TotalPortfolioValue:          analytics.TotalPortfolioValue,
		TotalPortfolioProfit:         analytics.TotalPortfolioProfit,
		TotalPortfolioProfitPercentage: analytics.TotalPortfolioProfitPercentage,
		TotalUnrealizedPL:            totalUnrealizedPL,
		TotalStockProfit:             analytics.TotalStockProfitLoss,
		TotalPortfolioValueFormatted:  web.FormatCurrency(analytics.TotalPortfolioValue),
		TotalPortfolioProfitFormatted: web.FormatCurrency(analytics.TotalPortfolioProfit),
		TotalPortfolioProfitPercentageFormatted: web.FormatPercentage(analytics.TotalPortfolioProfitPercentage),
		TotalUnrealizedPLFormatted:   web.FormatCurrency(totalUnrealizedPL),
		TotalStockProfitFormatted:    web.FormatCurrency(analytics.TotalStockProfitLoss),
		TotalDepositsFormatted:       web.FormatCurrency(analytics.TotalDeposits),
		DailyTheta:                   analytics.DailyTheta,
		DailyThetaFormatted:          web.FormatCurrency(analytics.DailyTheta),
		VIX:                          vix,
		VIXFormatted:                 fmt.Sprintf("%.2f", vix),
		// Daily returns data
		DailyReturns:     analytics.DailyReturns,
		DailyReturnsJSON: analytics.DailyReturnsJSON,
		// Net worth data
		NetWorthData:     netWorthData,
		NetWorthDataJSON: netWorthJSON,
	})
}

func handleRisk(w http.ResponseWriter, r *http.Request) {
	transactions := web.LoadTransactionsFromCSV("data/transactions.csv")
	analytics := web.CalculateAnalytics(nil, nil, transactions)

	// Calculate cash position for risk metrics
	cashPosition := web.CalculateCashPosition(analytics)
	cashPositionJSON := "[]"
	if jsonData, err := json.Marshal(cashPosition); err == nil {
		cashPositionJSON = string(jsonData)
	}

	// Calculate sector exposure
	sectorExposure := web.CalculateSectorExposure()
	sectorExposureJSON := "[]"
	if jsonData, err := json.Marshal(sectorExposure); err == nil {
		sectorExposureJSON = string(jsonData)
	}

	// Calculate position details
	positionDetails := web.CalculatePositionDetails()
	positionDetailsJSON := "[]"
	if jsonData, err := json.Marshal(positionDetails); err == nil {
		positionDetailsJSON = string(jsonData)
	}

	totalUnrealizedPL := calculateTotalUnrealizedPL()

	// Calculate weekly performance metrics
	weeklyPerf := web.CalculateWeeklyPerformance(analytics.TotalPortfolioValue)

	renderPage(w, "risk", web.PageData{
		Title:       "Risk - mnmlsm",
		CurrentPage: "risk",
		// Cash position data for risk page
		CashPosition:     cashPosition,
		CashPositionJSON: cashPositionJSON,
		// Sector exposure data
		SectorExposure:     sectorExposure,
		SectorExposureJSON: sectorExposureJSON,
		// Position details data
		PositionDetails:     positionDetails,
		PositionDetailsJSON: positionDetailsJSON,
		// Analytics for additional metrics
		TotalActiveCapital:              analytics.TotalActiveCapital,
		TotalActiveCapitalFormatted:     web.FormatCurrency(analytics.TotalActiveCapital),
		// Weekly performance metrics
		WeeklyReturnPercent:   weeklyPerf.WeeklyReturnPercent,
		WeeklyReturnFormatted: weeklyPerf.WeeklyReturnFormatted,
		WeeklyPL:              weeklyPerf.WeeklyPL,
		WeeklyPLFormatted:     weeklyPerf.WeeklyPLFormatted,
		DaysRemainingInWeek:   weeklyPerf.DaysRemainingInWeek,
		WeeklyReturnStatus:    weeklyPerf.WeeklyReturnStatus,
		WeekStartDate:         weeklyPerf.WeekStartDate,
		TargetWeeklyReturn:    weeklyPerf.TargetWeeklyReturn,
		// Portfolio values for header
		TotalPortfolioValue:                     analytics.TotalPortfolioValue,
		TotalPortfolioProfit:                    analytics.TotalPortfolioProfit,
		TotalPortfolioProfitPercentage:          analytics.TotalPortfolioProfitPercentage,
		TotalUnrealizedPL:                       totalUnrealizedPL,
		TotalPortfolioValueFormatted:            web.FormatCurrency(analytics.TotalPortfolioValue),
		TotalPortfolioProfitFormatted:           web.FormatCurrency(analytics.TotalPortfolioProfit),
		TotalPortfolioProfitPercentageFormatted: web.FormatPercentage(analytics.TotalPortfolioProfitPercentage),
		TotalUnrealizedPLFormatted:              web.FormatCurrency(totalUnrealizedPL),
	})
}

func handleRules(w http.ResponseWriter, r *http.Request) {
	transactions := web.LoadTransactionsFromCSV("data/transactions.csv")
	analytics := web.CalculateAnalytics(nil, nil, transactions)

	totalUnrealizedPL := calculateTotalUnrealizedPL()
	vix := web.LoadVIX("data/vix.csv")

	renderPage(w, "rules", web.PageData{
		Title:       "Rules - mnmlsm",
		CurrentPage: "rules",
		// Portfolio values for header
		TotalPortfolioValue:          analytics.TotalPortfolioValue,
		TotalPortfolioProfit:         analytics.TotalPortfolioProfit,
		TotalPortfolioProfitPercentage: analytics.TotalPortfolioProfitPercentage,
		TotalUnrealizedPL:            totalUnrealizedPL,
		TotalPortfolioValueFormatted:  web.FormatCurrency(analytics.TotalPortfolioValue),
		TotalPortfolioProfitFormatted: web.FormatCurrency(analytics.TotalPortfolioProfit),
		TotalPortfolioProfitPercentageFormatted: web.FormatPercentage(analytics.TotalPortfolioProfitPercentage),
		TotalUnrealizedPLFormatted:   web.FormatCurrency(totalUnrealizedPL),
		VIX:                          vix,
		VIXFormatted:                 fmt.Sprintf("%.2f", vix),
	})
}

func calculateTotalUnrealizedPL() float64 {
	stockTransactions := web.LoadStockTransactions("data/stocks_transactions.csv")
	stockPrices := web.LoadStockPrices("data/stock_prices.csv")
	positions := web.CalculateAllPositions(stockTransactions, stockPrices)

	totalUnrealizedPL := 0.0
	for _, pos := range positions {
		if pos.Type == "open" {
			totalUnrealizedPL += pos.UnrealizedPnL
		}
	}
	return totalUnrealizedPL
}

func renderPage(w http.ResponseWriter, page string, data web.PageData) {
	tmplFiles := []string{
		"layouts/main.html",
		"components/sidebar.html",
		"components/header.html",
		filepath.Join("pages", page+".html"),
	}

	funcMap := template.FuncMap{
		"hasPrefix": strings.HasPrefix,
		"isPositive": func(s string) bool {
			// Remove $ and commas, check if the number is positive
			cleaned := strings.TrimPrefix(s, "$")
			cleaned = strings.ReplaceAll(cleaned, ",", "")
			cleaned = strings.TrimSuffix(cleaned, "%")
			if val, err := strconv.ParseFloat(cleaned, 64); err == nil {
				return val > 0
			}
			return false
		},
		"isNegative": func(s string) bool {
			// Remove $ and commas, check if the number is negative
			cleaned := strings.TrimPrefix(s, "$")
			cleaned = strings.ReplaceAll(cleaned, ",", "")
			cleaned = strings.TrimSuffix(cleaned, "%")
			if val, err := strconv.ParseFloat(cleaned, 64); err == nil {
				return val < 0
			}
			return false
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFiles(tmplFiles...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}