package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

// HandleHome renders the home page with performance metrics
func HandleHome(w http.ResponseWriter, r *http.Request) {
	common := loadCommonData()

	// Calculate stock performance metrics
	stockTransactions := LoadStockTransactions("data/stocks_transactions.csv")
	stockPerformance := CalculateStockPerformance(stockTransactions)

	// Calculate option performance metrics
	optionTransactions := LoadOptionTransactions("data/options_transactions.csv")
	optionPerformance := CalculateOptionPerformance(optionTransactions)

	// Calculate cash position
	cashPosition := CalculateCashPosition(common.analytics)
	cashPositionJSON := "[]"
	if jsonData, err := json.Marshal(cashPosition); err == nil {
		cashPositionJSON = string(jsonData)
	}

	pageData := PageData{
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

// HandleOptions renders the options page with all option positions
func HandleOptions(w http.ResponseWriter, r *http.Request) {
	common := loadCommonData()

	// Load option positions from new transaction system
	optionTransactions := LoadOptionTransactions("data/options_transactions.csv")
	optionPositions := CalculateOptionPositions(optionTransactions)

	pageData := PageData{
		Title:           "Options - mnmlsm",
		CurrentPage:     "options",
		OptionPositions: optionPositions,
		// Options page specific metrics
		OpenOptionsCount:              common.analytics.OpenOptionsCount,
		ClosedOptionsCount:            common.analytics.ClosedOptionsCount,
		OptionsActiveCapital:          common.analytics.OptionsActiveCapital,
		TotalPremiums:                 common.analytics.TotalPremiums,
		OptionsActiveCapitalFormatted: FormatCurrency(common.analytics.OptionsActiveCapital),
		TotalPremiumsFormatted:        FormatCurrency(common.analytics.TotalPremiums),
	}

	enrichPageData(&pageData, common)
	renderPage(w, "options", pageData)
}

// HandleStocks renders the stocks page with open and closed positions
func HandleStocks(w http.ResponseWriter, r *http.Request) {
	// Only handle exact /stocks path
	if r.URL.Path != "/stocks" {
		http.NotFound(w, r)
		return
	}

	common := loadCommonData()

	// Load stock positions from transaction system
	stockTransactions := LoadStockTransactions("data/stocks_transactions.csv")
	stockPrices := LoadStockPrices("data/universe.csv")
	stockPositions := CalculateAllPositions(stockTransactions, stockPrices)
	allStocks := PositionsToStocks(stockPositions)

	// Separate open and closed positions
	var currentStocks, closedStocks []Stock
	for _, stock := range allStocks {
		if stock.ExitDate == "" {
			currentStocks = append(currentStocks, stock)
		} else {
			closedStocks = append(closedStocks, stock)
		}
	}

	// Load symbol summaries
	symbolSummaries := CalculateSymbolSummaries()

	pageData := PageData{
		Title:           "Stocks - mnmlsm",
		CurrentPage:     "stocks",
		Stocks:          currentStocks,
		ClosedStocks:    closedStocks,
		SymbolSummaries: symbolSummaries,
	}

	enrichPageData(&pageData, common)
	renderPage(w, "stocks/index", pageData)
}

// HandleStockPages renders individual stock detail pages
func HandleStockPages(w http.ResponseWriter, r *http.Request) {
	// Extract symbol from URL (e.g., /stocks/AMD -> AMD)
	symbol := strings.ToUpper(strings.TrimPrefix(r.URL.Path, "/stocks/"))

	if symbol == "" {
		http.NotFound(w, r)
		return
	}

	common := loadCommonData()

	// Get symbol-specific data
	symbolDetails := GetSymbolDetails(symbol, common.analytics.TotalPortfolioProfit)
	symbolStocks := GetStockPositionsBySymbol(symbol)
	symbolOptions := GetOptionPositionsBySymbol(symbol)

	// Return 404 if no data exists for this symbol
	if len(symbolStocks) == 0 && len(symbolOptions) == 0 {
		http.NotFound(w, r)
		return
	}

	pageData := PageData{
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

// HandleAnalytics renders the analytics page with portfolio metrics
func HandleAnalytics(w http.ResponseWriter, r *http.Request) {
	common := loadCommonData()

	// Calculate net worth data
	netWorthData := CalculateNetWorth(common.analytics.TotalPortfolioValue)
	netWorthJSON := "[]"
	var totalNetWorth float64
	if len(netWorthData) > 0 {
		if jsonData, err := json.Marshal(netWorthData); err == nil {
			netWorthJSON = string(jsonData)
		}
		// Get the latest (current) total net worth from the last entry
		totalNetWorth = netWorthData[len(netWorthData)-1].TotalNetWorth
	}

	pageData := PageData{
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
		TotalPremiumsFormatted:      FormatCurrency(common.analytics.TotalPremiums),
		TotalCapitalFormatted:       FormatCurrency(common.analytics.TotalCapital),
		TotalActiveCapitalFormatted: FormatCurrency(common.analytics.TotalActiveCapital),
		PremiumPerDayFormatted:      FormatCurrency(common.analytics.PremiumPerDay),
		AvgReturnPerTradeFormatted:  FormatPercentage(common.analytics.AvgReturnPerTrade),
		LargestPremiumFormatted:     FormatCurrency(common.analytics.LargestPremium),
		SmallestPremiumFormatted:    FormatCurrency(common.analytics.SmallestPremium),
		AveragePremiumFormatted:     FormatCurrency(common.analytics.AveragePremium),
		TotalStockProfit:            common.analytics.TotalStockProfitLoss,
		TotalStockProfitFormatted:   FormatCurrency(common.analytics.TotalStockProfitLoss),
		TotalDepositsFormatted:      FormatCurrency(common.analytics.TotalDeposits),
		DailyTheta:                  common.analytics.DailyTheta,
		DailyThetaFormatted:         FormatCurrency(common.analytics.DailyTheta),
		// Daily returns data
		DailyReturns:     common.analytics.DailyReturns,
		DailyReturnsJSON: common.analytics.DailyReturnsJSON,
		// Net worth data
		NetWorthData:           netWorthData,
		NetWorthDataJSON:       netWorthJSON,
		TotalNetWorth:          totalNetWorth,
		TotalNetWorthFormatted: FormatCurrency(totalNetWorth),
		// Projected $1M data
		ProjectedMillionDateFormatted: common.analytics.ProjectedMillionDateFormatted,
		DaysToMillion:                 common.analytics.DaysToMillion,
	}

	enrichPageData(&pageData, common)
	renderPage(w, "analytics", pageData)
}

// HandleRisk renders the risk management page
func HandleRisk(w http.ResponseWriter, r *http.Request) {
	common := loadCommonData()

	// Calculate cash position for risk metrics
	cashPosition := CalculateCashPosition(common.analytics)
	cashPositionJSON := "[]"
	if jsonData, err := json.Marshal(cashPosition); err == nil {
		cashPositionJSON = string(jsonData)
	}

	// Calculate sector exposure
	sectorExposure := CalculateSectorExposure()
	sectorExposureJSON := "[]"
	if jsonData, err := json.Marshal(sectorExposure); err == nil {
		sectorExposureJSON = string(jsonData)
	}

	// Calculate position details
	positionDetails := CalculatePositionDetails()
	positionDetailsJSON := "[]"
	if jsonData, err := json.Marshal(positionDetails); err == nil {
		positionDetailsJSON = string(jsonData)
	}

	pageData := PageData{
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
		TotalActiveCapitalFormatted: FormatCurrency(common.analytics.TotalActiveCapital),
		// Daily returns data for client-side weekly calculation
		DailyReturns:     common.analytics.DailyReturns,
		DailyReturnsJSON: common.analytics.DailyReturnsJSON,
	}

	enrichPageData(&pageData, common)
	renderPage(w, "risk", pageData)
}

// HandleRules renders the trading rules page
func HandleRules(w http.ResponseWriter, r *http.Request) {
	common := loadCommonData()

	pageData := PageData{
		Title:       "Rules - mnmlsm",
		CurrentPage: "rules",
	}

	enrichPageData(&pageData, common)
	renderPage(w, "rules", pageData)
}

// commonData holds data shared across all pages (header, portfolio metrics, etc.)
type commonData struct {
	analytics         Analytics
	totalUnrealizedPL float64
	vix               float64
}

// loadCommonData loads all shared data that appears on every page
func loadCommonData() commonData {
	transactions := LoadTransactionsFromCSV("data/transactions.csv")
	analytics := CalculateAnalytics(nil, nil, transactions)

	stockTransactions := LoadStockTransactions("data/stocks_transactions.csv")
	stockPrices := LoadStockPrices("data/universe.csv")
	positions := CalculateAllPositions(stockTransactions, stockPrices)

	totalUnrealizedPL := 0.0
	for _, pos := range positions {
		if pos.Type == "open" {
			totalUnrealizedPL += pos.UnrealizedPnL
		}
	}

	vix := LoadVIX("data/vix.csv")

	return commonData{
		analytics:         analytics,
		totalUnrealizedPL: totalUnrealizedPL,
		vix:               vix,
	}
}

// enrichPageData adds common portfolio/header data to PageData
func enrichPageData(data *PageData, common commonData) {
	data.TotalPortfolioValue = common.analytics.TotalPortfolioValue
	data.TotalPortfolioProfit = common.analytics.TotalPortfolioProfit
	data.TotalPortfolioProfitPercentage = common.analytics.TotalPortfolioProfitPercentage
	data.TotalUnrealizedPL = common.totalUnrealizedPL
	data.VIX = common.vix

	// Formatted versions
	data.TotalPortfolioValueFormatted = FormatCurrency(common.analytics.TotalPortfolioValue)
	data.TotalPortfolioProfitFormatted = FormatCurrency(common.analytics.TotalPortfolioProfit)
	data.TotalPortfolioProfitPercentageFormatted = FormatPercentage(common.analytics.TotalPortfolioProfitPercentage)
	data.TotalUnrealizedPLFormatted = FormatCurrency(common.totalUnrealizedPL)
	data.VIXFormatted = fmt.Sprintf("%.2f", common.vix)

	// Time-Weighted Return
	data.TimeWeightedReturn = common.analytics.TimeWeightedReturn
	data.TimeWeightedReturnAnnualized = common.analytics.TimeWeightedReturnAnnualized
	data.TimeWeightedReturnFormatted = FormatPercentage(common.analytics.TimeWeightedReturn)
	data.TimeWeightedReturnAnnualizedFormatted = FormatPercentage(common.analytics.TimeWeightedReturnAnnualized)
}

// renderPage renders an HTML template with the given page data
func renderPage(w http.ResponseWriter, page string, data PageData) {
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
