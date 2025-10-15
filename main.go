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
	common := loadCommonData()

	// Calculate stock performance metrics
	stockTransactions := web.LoadStockTransactions("data/stocks_transactions.csv")
	stockPerformance := web.CalculateStockPerformance(stockTransactions)

	// Calculate option performance metrics
	optionTransactions := web.LoadOptionTransactions("data/options_transactions.csv")
	optionPerformance := web.CalculateOptionPerformance(optionTransactions)

	// Calculate cash position
	cashPosition := web.CalculateCashPosition(common.analytics)
	cashPositionJSON := "[]"
	if jsonData, err := json.Marshal(cashPosition); err == nil {
		cashPositionJSON = string(jsonData)
	}

	pageData := web.PageData{
		Title:       "Home - mnmlsm",
		CurrentPage: "home",
		// Stock performance metrics
		StockPerformance: stockPerformance,
		// Options performance metrics
		OptionPerformance: optionPerformance,
		// Cash position data
		CashPosition:     cashPosition,
		CashPositionJSON: cashPositionJSON,
	}

	enrichPageData(&pageData, common)
	renderPage(w, "home", pageData)
}

func handleOptions(w http.ResponseWriter, r *http.Request) {
	common := loadCommonData()

	// Load option positions from new transaction system
	optionTransactions := web.LoadOptionTransactions("data/options_transactions.csv")
	optionPositions := web.CalculateOptionPositions(optionTransactions)

	pageData := web.PageData{
		Title:           "Options - mnmlsm",
		CurrentPage:     "options",
		OptionPositions: optionPositions,
		// Options page specific metrics
		OpenOptionsCount:              common.analytics.OpenOptionsCount,
		ClosedOptionsCount:            common.analytics.ClosedOptionsCount,
		OptionsActiveCapital:          common.analytics.OptionsActiveCapital,
		TotalPremiums:                 common.analytics.TotalPremiums,
		OptionsActiveCapitalFormatted: web.FormatCurrency(common.analytics.OptionsActiveCapital),
		TotalPremiumsFormatted:        web.FormatCurrency(common.analytics.TotalPremiums),
	}

	enrichPageData(&pageData, common)
	renderPage(w, "options", pageData)
}

func handleStocks(w http.ResponseWriter, r *http.Request) {
	// Only handle exact /stocks path
	if r.URL.Path != "/stocks" {
		http.NotFound(w, r)
		return
	}

	common := loadCommonData()

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

	// Load symbol summaries
	symbolSummaries := web.CalculateSymbolSummaries()

	pageData := web.PageData{
		Title:           "Stocks - mnmlsm",
		CurrentPage:     "stocks",
		Stocks:          currentStocks,
		ClosedStocks:    closedStocks,
		SymbolSummaries: symbolSummaries,
	}

	enrichPageData(&pageData, common)
	renderPage(w, "stocks/index", pageData)
}

func handleStockPages(w http.ResponseWriter, r *http.Request) {
	// Extract symbol from URL (e.g., /stocks/AMD -> AMD)
	symbol := strings.ToUpper(strings.TrimPrefix(r.URL.Path, "/stocks/"))

	if symbol == "" {
		http.NotFound(w, r)
		return
	}

	common := loadCommonData()

	// Get symbol-specific data
	symbolDetails := web.GetSymbolDetails(symbol, common.analytics.TotalPortfolioProfit)
	symbolStocks := web.GetStockPositionsBySymbol(symbol)
	symbolOptions := web.GetOptionPositionsBySymbol(symbol)

	// Return 404 if no data exists for this symbol
	if len(symbolStocks) == 0 && len(symbolOptions) == 0 {
		http.NotFound(w, r)
		return
	}

	pageData := web.PageData{
		Title:         symbol + " - Stock Detail - mnmlsm",
		CurrentPage:   "stocks",
		Symbol:        symbol,
		SymbolDetails: symbolDetails,
		SymbolStocks:  symbolStocks,
		SymbolOptions: symbolOptions,
	}

	enrichPageData(&pageData, common)
	renderPage(w, "stocks/detail", pageData)
}

func handleAnalytics(w http.ResponseWriter, r *http.Request) {
	common := loadCommonData()

	// Calculate net worth data
	netWorthData := web.CalculateNetWorth(common.analytics.TotalPortfolioValue)
	netWorthJSON := "[]"
	var totalNetWorth float64
	if len(netWorthData) > 0 {
		if jsonData, err := json.Marshal(netWorthData); err == nil {
			netWorthJSON = string(jsonData)
		}
		// Get the latest (current) total net worth from the last entry
		totalNetWorth = netWorthData[len(netWorthData)-1].TotalNetWorth
	}

	pageData := web.PageData{
		Title:              "Analytics - Options Tracker",
		CurrentPage:        "analytics",
		TotalPremiums:      common.analytics.TotalPremiums,
		TotalCapital:       common.analytics.TotalCapital,
		TotalActiveCapital: common.analytics.TotalActiveCapital,
		PremiumPerDay:      common.analytics.PremiumPerDay,
		AvgReturnPerTrade:  common.analytics.AvgReturnPerTrade,
		LargestPremium:     common.analytics.LargestPremium,
		SmallestPremium:    common.analytics.SmallestPremium,
		AveragePremium:     common.analytics.AveragePremium,
		OptionTradesCount:  common.analytics.OptionTradesCount,
		StockTradesCount:   common.analytics.StockTradesCount,
		TotalTradesCount:   common.analytics.TotalTradesCount,
		DaysSinceStart:     common.analytics.DaysSinceStart,
		TotalPremiumsFormatted:      web.FormatCurrency(common.analytics.TotalPremiums),
		TotalCapitalFormatted:       web.FormatCurrency(common.analytics.TotalCapital),
		TotalActiveCapitalFormatted: web.FormatCurrency(common.analytics.TotalActiveCapital),
		PremiumPerDayFormatted:      web.FormatCurrency(common.analytics.PremiumPerDay),
		AvgReturnPerTradeFormatted:  web.FormatPercentage(common.analytics.AvgReturnPerTrade),
		LargestPremiumFormatted:     web.FormatCurrency(common.analytics.LargestPremium),
		SmallestPremiumFormatted:    web.FormatCurrency(common.analytics.SmallestPremium),
		AveragePremiumFormatted:     web.FormatCurrency(common.analytics.AveragePremium),
		TotalStockProfit:            common.analytics.TotalStockProfitLoss,
		TotalStockProfitFormatted:   web.FormatCurrency(common.analytics.TotalStockProfitLoss),
		TotalDepositsFormatted:      web.FormatCurrency(common.analytics.TotalDeposits),
		DailyTheta:                  common.analytics.DailyTheta,
		DailyThetaFormatted:         web.FormatCurrency(common.analytics.DailyTheta),
		// Daily returns data
		DailyReturns:     common.analytics.DailyReturns,
		DailyReturnsJSON: common.analytics.DailyReturnsJSON,
		// Net worth data
		NetWorthData:           netWorthData,
		NetWorthDataJSON:       netWorthJSON,
		TotalNetWorth:          totalNetWorth,
		TotalNetWorthFormatted: web.FormatCurrency(totalNetWorth),
		// Projected $1M data
		ProjectedMillionDateFormatted: common.analytics.ProjectedMillionDateFormatted,
		DaysToMillion:                 common.analytics.DaysToMillion,
	}

	enrichPageData(&pageData, common)
	renderPage(w, "analytics", pageData)
}

func handleRisk(w http.ResponseWriter, r *http.Request) {
	common := loadCommonData()

	// Calculate cash position for risk metrics
	cashPosition := web.CalculateCashPosition(common.analytics)
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

	pageData := web.PageData{
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
		TotalActiveCapital:          common.analytics.TotalActiveCapital,
		TotalActiveCapitalFormatted: web.FormatCurrency(common.analytics.TotalActiveCapital),
		// Daily returns data for client-side weekly calculation
		DailyReturns:     common.analytics.DailyReturns,
		DailyReturnsJSON: common.analytics.DailyReturnsJSON,
	}

	enrichPageData(&pageData, common)
	renderPage(w, "risk", pageData)
}

func handleRules(w http.ResponseWriter, r *http.Request) {
	common := loadCommonData()

	pageData := web.PageData{
		Title:       "Rules - mnmlsm",
		CurrentPage: "rules",
	}

	enrichPageData(&pageData, common)
	renderPage(w, "rules", pageData)
}

// commonData holds data shared across all pages (header, portfolio metrics, etc.)
type commonData struct {
	analytics         web.Analytics
	totalUnrealizedPL float64
	vix               float64
}

// loadCommonData loads all shared data that appears on every page
func loadCommonData() commonData {
	transactions := web.LoadTransactionsFromCSV("data/transactions.csv")
	analytics := web.CalculateAnalytics(nil, nil, transactions)

	stockTransactions := web.LoadStockTransactions("data/stocks_transactions.csv")
	stockPrices := web.LoadStockPrices("data/stock_prices.csv")
	positions := web.CalculateAllPositions(stockTransactions, stockPrices)

	totalUnrealizedPL := 0.0
	for _, pos := range positions {
		if pos.Type == "open" {
			totalUnrealizedPL += pos.UnrealizedPnL
		}
	}

	vix := web.LoadVIX("data/vix.csv")

	return commonData{
		analytics:         analytics,
		totalUnrealizedPL: totalUnrealizedPL,
		vix:               vix,
	}
}

// enrichPageData adds common portfolio/header data to PageData
func enrichPageData(data *web.PageData, common commonData) {
	data.TotalPortfolioValue = common.analytics.TotalPortfolioValue
	data.TotalPortfolioProfit = common.analytics.TotalPortfolioProfit
	data.TotalPortfolioProfitPercentage = common.analytics.TotalPortfolioProfitPercentage
	data.TotalUnrealizedPL = common.totalUnrealizedPL
	data.VIX = common.vix

	// Formatted versions
	data.TotalPortfolioValueFormatted = web.FormatCurrency(common.analytics.TotalPortfolioValue)
	data.TotalPortfolioProfitFormatted = web.FormatCurrency(common.analytics.TotalPortfolioProfit)
	data.TotalPortfolioProfitPercentageFormatted = web.FormatPercentage(common.analytics.TotalPortfolioProfitPercentage)
	data.TotalUnrealizedPLFormatted = web.FormatCurrency(common.totalUnrealizedPL)
	data.VIXFormatted = fmt.Sprintf("%.2f", common.vix)

	// Time-Weighted Return
	data.TimeWeightedReturn = common.analytics.TimeWeightedReturn
	data.TimeWeightedReturnAnnualized = common.analytics.TimeWeightedReturnAnnualized
	data.TimeWeightedReturnFormatted = web.FormatPercentage(common.analytics.TimeWeightedReturn)
	data.TimeWeightedReturnAnnualizedFormatted = web.FormatPercentage(common.analytics.TimeWeightedReturnAnnualized)
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