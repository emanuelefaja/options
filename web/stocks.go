package web

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Stock struct {
	Symbol          string
	Shares          string
	AvgCost         string
	Capital         string
	ProfitLoss      string
	ReturnPerc      string
	EntryDate       string
	ExitDate        string
	StocksID        string
	CurrentPrice    string
	MarketValue     string
	UnrealizedPnL   string
	UnrealizedPerc  string
}

type StockTransaction struct {
	Date          string
	Type          string
	Symbol        string
	Shares        float64
	Price         float64
	Amount        float64
	Commission    float64
	TransactionID string
}

type Lot struct {
	Date      string
	Shares    float64
	Price     float64
	CostBasis float64
}

type Position struct {
	Symbol         string
	Type           string  // "open" or "closed"
	Shares         float64
	AvgBuyPrice    float64
	AvgSellPrice   float64
	CostBasis      float64
	SaleProceeds   float64
	RealizedPnL    float64
	ReturnPerc     float64
	OpenDate       string
	CloseDate      string
	CurrentPrice   float64
	MarketValue    float64
	UnrealizedPnL  float64
	UnrealizedPerc float64
}

func LoadStockTransactions(filename string) []StockTransaction {
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Error opening stock transactions CSV file: %v", err)
		return []StockTransaction{}
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Printf("Error reading stock transactions CSV file: %v", err)
		return []StockTransaction{}
	}

	var transactions []StockTransaction
	for i, record := range records {
		if i == 0 {
			continue
		}
		if len(record) >= 7 {
			shares, _ := strconv.ParseFloat(record[3], 64)
			price, _ := strconv.ParseFloat(record[4], 64)
			amount, _ := strconv.ParseFloat(record[5], 64)
			commission, _ := strconv.ParseFloat(record[6], 64)
			
			transaction := StockTransaction{
				Date:          record[0],
				Type:          record[1],
				Symbol:        record[2],
				Shares:        shares,
				Price:         price,
				Amount:        amount,
				Commission:    commission,
				TransactionID: strconv.Itoa(i),
			}
			transactions = append(transactions, transaction)
		}
	}
	return transactions
}

func LoadStockPrices(filename string) map[string]float64 {
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Error opening stock prices CSV file: %v", err)
		return make(map[string]float64)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Printf("Error reading stock prices CSV file: %v", err)
		return make(map[string]float64)
	}

	prices := make(map[string]float64)
	for i, record := range records {
		if i == 0 {
			continue
		}
		if len(record) >= 2 {
			ticker := record[0]
			price, _ := strconv.ParseFloat(record[1], 64)
			prices[ticker] = price
		}
	}
	return prices
}

func CalculateAllPositions(transactions []StockTransaction, stockPrices map[string]float64) []Position {
	var positions []Position
	symbolLots := make(map[string][]Lot)
	
	for _, tx := range transactions {
		if tx.Type == "Buy" {
			lot := Lot{
				Date:      tx.Date,
				Shares:    tx.Shares,
				Price:     tx.Price,
				CostBasis: tx.Amount + tx.Commission,
			}
			symbolLots[tx.Symbol] = append(symbolLots[tx.Symbol], lot)
			
		} else if tx.Type == "Sell" {
			lots := symbolLots[tx.Symbol]
			remainingToSell := tx.Shares
			saleProceeds := tx.Amount - tx.Commission
			
			var closedLots []Lot
			var newLots []Lot
			totalCostBasisSold := 0.0
			
			for _, lot := range lots {
				if remainingToSell <= 0 {
					newLots = append(newLots, lot)
					continue
				}
				
				if lot.Shares <= remainingToSell {
					closedLots = append(closedLots, lot)
					totalCostBasisSold += lot.CostBasis
					remainingToSell -= lot.Shares
				} else {
					shareFraction := remainingToSell / lot.Shares
					costBasisFraction := lot.CostBasis * shareFraction
					
					closedLot := Lot{
						Date:      lot.Date,
						Shares:    remainingToSell,
						Price:     lot.Price,
						CostBasis: costBasisFraction,
					}
					closedLots = append(closedLots, closedLot)
					totalCostBasisSold += costBasisFraction
					
					lot.Shares -= remainingToSell
					lot.CostBasis -= costBasisFraction
					newLots = append(newLots, lot)
					remainingToSell = 0
				}
			}
			
			symbolLots[tx.Symbol] = newLots
			
			if len(closedLots) > 0 {
				totalShares := 0.0
				avgBuyPrice := 0.0
				openDate := closedLots[0].Date
				
				for _, lot := range closedLots {
					totalShares += lot.Shares
					avgBuyPrice += lot.Price * lot.Shares
					if lot.Date < openDate {
						openDate = lot.Date
					}
				}
				avgBuyPrice = avgBuyPrice / totalShares
				
				pnl := saleProceeds - totalCostBasisSold
				returnPerc := 0.0
				if totalCostBasisSold > 0 {
					returnPerc = (pnl / totalCostBasisSold) * 100
				}
				
				closedPos := Position{
					Symbol:       tx.Symbol,
					Type:         "closed",
					Shares:       totalShares,
					AvgBuyPrice:  avgBuyPrice,
					AvgSellPrice: tx.Price,
					CostBasis:    totalCostBasisSold,
					SaleProceeds: saleProceeds,
					RealizedPnL:  pnl,
					ReturnPerc:   returnPerc,
					OpenDate:     openDate,
					CloseDate:    tx.Date,
				}
				positions = append(positions, closedPos)
			}
		}
	}
	
	for symbol, lots := range symbolLots {
		if len(lots) > 0 {
			totalShares := 0.0
			totalCostBasis := 0.0
			avgPrice := 0.0
			openDate := lots[0].Date
			
			for _, lot := range lots {
				totalShares += lot.Shares
				totalCostBasis += lot.CostBasis
				avgPrice += lot.Price * lot.Shares
				if lot.Date < openDate {
					openDate = lot.Date
				}
			}
			
			if totalShares > 0 {
				avgPrice = avgPrice / totalShares

				currentPrice := stockPrices[symbol]
				marketValue := currentPrice * totalShares
				unrealizedPnL := marketValue - totalCostBasis
				unrealizedPerc := 0.0
				if totalCostBasis > 0 {
					unrealizedPerc = (unrealizedPnL / totalCostBasis) * 100
				}

				openPos := Position{
					Symbol:         symbol,
					Type:           "open",
					Shares:         totalShares,
					AvgBuyPrice:    totalCostBasis / totalShares,
					CostBasis:      totalCostBasis,
					OpenDate:       openDate,
					CurrentPrice:   currentPrice,
					MarketValue:    marketValue,
					UnrealizedPnL:  unrealizedPnL,
					UnrealizedPerc: unrealizedPerc,
				}
				positions = append(positions, openPos)
			}
		}
	}
	
	sort.Slice(positions, func(i, j int) bool {
		if positions[i].Symbol != positions[j].Symbol {
			return positions[i].Symbol < positions[j].Symbol
		}
		if positions[i].Type != positions[j].Type {
			return positions[i].Type == "open"
		}
		return positions[i].OpenDate < positions[j].OpenDate
	})
	
	return positions
}

func formatStockDate(dateStr string) string {
	if dateStr == "" {
		return ""
	}
	
	// Parse the date string (format: "2025-08-26")
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr // Return original if parsing fails
	}
	
	// Format as "September 10 2025"
	return t.Format("January 2 2006")
}

func PositionsToStocks(positions []Position) []Stock {
	var stocks []Stock
	
	for i, pos := range positions {
		stock := Stock{
			Symbol:     pos.Symbol,
			Shares:     fmt.Sprintf("%.0f", pos.Shares),
			AvgCost:    fmt.Sprintf("$%.2f", pos.AvgBuyPrice),
			Capital:    FormatCurrency(pos.CostBasis),
			StocksID:   strconv.Itoa(i + 1),
		}
		
		if pos.Type == "open" {
			stock.ProfitLoss = "$0.00"
			stock.ReturnPerc = "0.00%"
			stock.EntryDate = formatStockDate(pos.OpenDate)
			stock.ExitDate = ""
			stock.CurrentPrice = fmt.Sprintf("$%.2f", pos.CurrentPrice)
			stock.MarketValue = FormatCurrency(pos.MarketValue)
			stock.UnrealizedPnL = fmt.Sprintf("$%.2f", pos.UnrealizedPnL)
			stock.UnrealizedPerc = fmt.Sprintf("%.2f%%", pos.UnrealizedPerc)
		} else {
			stock.ProfitLoss = fmt.Sprintf("$%.2f", pos.RealizedPnL)
			stock.ReturnPerc = fmt.Sprintf("%.2f%%", pos.ReturnPerc)
			stock.EntryDate = formatStockDate(pos.OpenDate)
			stock.ExitDate = formatStockDate(pos.CloseDate)
		}
		
		stocks = append(stocks, stock)
	}
	
	return stocks
}

func LoadStocksWithPositions(filename string) []Stock {
	transactionsFile := strings.Replace(filename, "stocks.csv", "stocks_transactions.csv", 1)
	pricesFile := strings.Replace(filename, "stocks.csv", "stock_prices.csv", 1)

	transactions := LoadStockTransactions(transactionsFile)
	stockPrices := LoadStockPrices(pricesFile)

	if len(transactions) > 0 {
		positions := CalculateAllPositions(transactions, stockPrices)
		return PositionsToStocks(positions)
	}

	return []Stock{}
}

// Keep old functions for backward compatibility
func LoadStocksFromCSV(filename string) []Stock {
	stocks := LoadStocksWithPositions(filename)
	// Filter only open positions for backward compatibility
	var openStocks []Stock
	for _, stock := range stocks {
		if stock.ExitDate == "" {
			openStocks = append(openStocks, stock)
		}
	}
	return openStocks
}