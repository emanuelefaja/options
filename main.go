package main

import (
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
	mux.HandleFunc("/stocks", handleStocks)
	mux.HandleFunc("/analytics", handleAnalytics)

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
	
	renderPage(w, "index", web.PageData{
		Title:       "Options Tracker",
		CurrentPage: "home",
		Trades:      trades,
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
	trades := web.LoadTradesFromCSV("data/options.csv")
	currentStocks, closedStocks := web.LoadStocksWithHistory("data/stocks.csv")
	transactions := web.LoadTransactionsFromCSV("data/transactions.csv")
	analytics := web.CalculateAnalytics(trades, currentStocks, transactions)
	
	renderPage(w, "stocks", web.PageData{
		Title:        "Stocks - mnmlsm",
		CurrentPage:  "stocks",
		Stocks:       currentStocks,
		ClosedStocks: closedStocks,
		// Portfolio values for header
		TotalPortfolioValue:          analytics.TotalPortfolioValue,
		TotalPortfolioProfit:         analytics.TotalPortfolioProfit,
		TotalPortfolioProfitPercentage: analytics.TotalPortfolioProfitPercentage,
		TotalPortfolioValueFormatted:  web.FormatCurrency(analytics.TotalPortfolioValue),
		TotalPortfolioProfitFormatted: web.FormatCurrency(analytics.TotalPortfolioProfit),
		TotalPortfolioProfitPercentageFormatted: web.FormatPercentage(analytics.TotalPortfolioProfitPercentage),
	})
}

func handleAnalytics(w http.ResponseWriter, r *http.Request) {
	trades := web.LoadTradesFromCSV("data/options.csv")
	stocks := web.LoadStocksFromCSV("data/stocks.csv")
	transactions := web.LoadTransactionsFromCSV("data/transactions.csv")
	analytics := web.CalculateAnalytics(trades, stocks, transactions)
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