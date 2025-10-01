package main

import (
	"encoding/json"
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
	mux.HandleFunc("/rules", handleRules)

	log.Println("Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	trades := web.LoadTradesFromCSV("data/options.csv")
	stocks := web.LoadStocksFromCSV("data/stocks.csv")
	transactions := web.LoadTransactionsFromCSV("data/transactions.csv")
	analytics := web.CalculateAnalytics(trades, stocks, transactions)

	// Calculate stock performance metrics
	stockTransactions := web.LoadStockTransactions("data/stocks_transactions.csv")
	stockPerformance := web.CalculateStockPerformance(stockTransactions)

	renderPage(w, "home", web.PageData{
		Title:       "Home - mnmlsm",
		CurrentPage: "home",
		// Portfolio values for header
		TotalPortfolioValue:                     analytics.TotalPortfolioValue,
		TotalPortfolioProfit:                    analytics.TotalPortfolioProfit,
		TotalPortfolioProfitPercentage:          analytics.TotalPortfolioProfitPercentage,
		TotalPortfolioValueFormatted:            web.FormatCurrency(analytics.TotalPortfolioValue),
		TotalPortfolioProfitFormatted:           web.FormatCurrency(analytics.TotalPortfolioProfit),
		TotalPortfolioProfitPercentageFormatted: web.FormatPercentage(analytics.TotalPortfolioProfitPercentage),
		// Stock performance metrics
		StockPerformance: stockPerformance,
	})
}

func handleOptions(w http.ResponseWriter, r *http.Request) {
	// Load option positions from new transaction system
	optionTransactions := web.LoadOptionTransactions("data/options_transactions.csv")
	optionPositions := web.CalculateOptionPositions(optionTransactions)

	// Keep loading old trades for now to ensure compatibility
	trades := web.LoadTradesFromCSV("data/options.csv")
	stocks := web.LoadStocksFromCSV("data/stocks.csv")
	transactions := web.LoadTransactionsFromCSV("data/transactions.csv")
	analytics := web.CalculateAnalytics(trades, stocks, transactions)

	renderPage(w, "options", web.PageData{
		Title:           "Options - mnmlsm",
		CurrentPage:     "options",
		Trades:          trades, // Keep for now, will be replaced by OptionPositions in template update
		OptionPositions: optionPositions,
		// Options page specific metrics
		OpenOptionsCount:     analytics.OpenOptionsCount,
		ClosedOptionsCount:   analytics.ClosedOptionsCount,
		OptionsActiveCapital: analytics.OptionsActiveCapital,
		CollectedPremiums:    analytics.CollectedPremiums,
		OptionsActiveCapitalFormatted: web.FormatCurrency(analytics.OptionsActiveCapital),
		CollectedPremiumsFormatted:    web.FormatCurrency(analytics.CollectedPremiums),
		// Portfolio values for header
		TotalPortfolioValue:          analytics.TotalPortfolioValue,
		TotalPortfolioProfit:         analytics.TotalPortfolioProfit,
		TotalPortfolioProfitPercentage: analytics.TotalPortfolioProfitPercentage,
		TotalPortfolioValueFormatted:  web.FormatCurrency(analytics.TotalPortfolioValue),
		TotalPortfolioProfitFormatted: web.FormatCurrency(analytics.TotalPortfolioProfit),
		TotalPortfolioProfitPercentageFormatted: web.FormatPercentage(analytics.TotalPortfolioProfitPercentage),
	})
}

func handleStocks(w http.ResponseWriter, r *http.Request) {
	// Only handle exact /stocks path
	if r.URL.Path != "/stocks" {
		http.NotFound(w, r)
		return
	}

	trades := web.LoadTradesFromCSV("data/options.csv")

	// Load stock positions from transaction system
	stockTransactions := web.LoadStockTransactions("data/stocks_transactions.csv")
	stockPositions := web.CalculateAllPositions(stockTransactions)
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

	transactions := web.LoadTransactionsFromCSV("data/transactions.csv")
	analytics := web.CalculateAnalytics(trades, currentStocks, transactions)

	// Load symbol summaries
	symbolSummaries := web.CalculateSymbolSummaries()

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
		TotalPortfolioValueFormatted:            web.FormatCurrency(analytics.TotalPortfolioValue),
		TotalPortfolioProfitFormatted:           web.FormatCurrency(analytics.TotalPortfolioProfit),
		TotalPortfolioProfitPercentageFormatted: web.FormatPercentage(analytics.TotalPortfolioProfitPercentage),
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
	trades := web.LoadTradesFromCSV("data/options.csv")
	stocks := web.LoadStocksFromCSV("data/stocks.csv")
	transactions := web.LoadTransactionsFromCSV("data/transactions.csv")
	analytics := web.CalculateAnalytics(trades, stocks, transactions)

	// Get symbol-specific data
	symbolDetails := web.GetSymbolDetails(symbol, analytics.TotalPortfolioProfit)
	symbolStocks := web.GetStockPositionsBySymbol(symbol)
	symbolOptions := web.GetOptionPositionsBySymbol(symbol)

	// Return 404 if no data exists for this symbol
	if len(symbolStocks) == 0 && len(symbolOptions) == 0 {
		http.NotFound(w, r)
		return
	}

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
		TotalPortfolioValueFormatted:            web.FormatCurrency(analytics.TotalPortfolioValue),
		TotalPortfolioProfitFormatted:           web.FormatCurrency(analytics.TotalPortfolioProfit),
		TotalPortfolioProfitPercentageFormatted: web.FormatPercentage(analytics.TotalPortfolioProfitPercentage),
	})
}

func handleAnalytics(w http.ResponseWriter, r *http.Request) {
	trades := web.LoadTradesFromCSV("data/options.csv")
	stocks := web.LoadStocksFromCSV("data/stocks.csv")
	transactions := web.LoadTransactionsFromCSV("data/transactions.csv")
	analytics := web.CalculateAnalytics(trades, stocks, transactions)

	// Calculate net worth data
	netWorthData := web.CalculateNetWorth(analytics.TotalPortfolioValue)
	netWorthJSON := "[]"
	if len(netWorthData) > 0 {
		if jsonData, err := json.Marshal(netWorthData); err == nil {
			netWorthJSON = string(jsonData)
		}
	}

	renderPage(w, "analytics", web.PageData{
		Title:              "Analytics - Options Tracker",
		CurrentPage:        "analytics",
		Trades:             trades,
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
		TotalStockProfit:             analytics.TotalStockProfitLoss,
		TotalPortfolioValueFormatted:  web.FormatCurrency(analytics.TotalPortfolioValue),
		TotalPortfolioProfitFormatted: web.FormatCurrency(analytics.TotalPortfolioProfit),
		TotalPortfolioProfitPercentageFormatted: web.FormatPercentage(analytics.TotalPortfolioProfitPercentage),
		TotalStockProfitFormatted:    web.FormatCurrency(analytics.TotalStockProfitLoss),
		TotalDepositsFormatted:       web.FormatCurrency(analytics.TotalDeposits),
		// Daily returns data
		DailyReturns:     analytics.DailyReturns,
		DailyReturnsJSON: analytics.DailyReturnsJSON,
		// Net worth data
		NetWorthData:     netWorthData,
		NetWorthDataJSON: netWorthJSON,
	})
}

func handleRules(w http.ResponseWriter, r *http.Request) {
	trades := web.LoadTradesFromCSV("data/options.csv")
	stocks := web.LoadStocksFromCSV("data/stocks.csv")
	transactions := web.LoadTransactionsFromCSV("data/transactions.csv")
	analytics := web.CalculateAnalytics(trades, stocks, transactions)

	renderPage(w, "rules", web.PageData{
		Title:       "Rules - mnmlsm",
		CurrentPage: "rules",
		// Portfolio values for header
		TotalPortfolioValue:          analytics.TotalPortfolioValue,
		TotalPortfolioProfit:         analytics.TotalPortfolioProfit,
		TotalPortfolioProfitPercentage: analytics.TotalPortfolioProfitPercentage,
		TotalPortfolioValueFormatted:  web.FormatCurrency(analytics.TotalPortfolioValue),
		TotalPortfolioProfitFormatted: web.FormatCurrency(analytics.TotalPortfolioProfit),
		TotalPortfolioProfitPercentageFormatted: web.FormatPercentage(analytics.TotalPortfolioProfitPercentage),
	})
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