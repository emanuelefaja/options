package main

import (
	"encoding/csv"
	"fmt"
	"math"
	"mnmlsm/ibkr"
	"os"
	"strconv"
	"time"
)

type UniverseStock struct {
	Ticker string
	Name   string
	Price  float64
	IV     float64
	Sector string
}

func main() {
	fmt.Println("🔄 Updating universe.csv with live market data...\n")

	// Read current universe.csv
	stocks, err := readUniverse("data/universe.csv")
	if err != nil {
		fmt.Printf("❌ Error reading universe.csv: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📊 Found %d stocks to update\n\n", len(stocks))

	// Create IBKR client
	client := ibkr.NewClient()

	// Update each stock
	successCount := 0
	errorCount := 0

	for i := range stocks {
		fmt.Printf("[%d/%d] Updating %s (%s)...", i+1, len(stocks), stocks[i].Ticker, stocks[i].Name)

		// Get stock price
		quote, err := client.GetQuote(stocks[i].Ticker)
		if err != nil {
			fmt.Printf(" ❌ Failed to get quote: %v\n", err)
			errorCount++
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Check if price is valid
		if quote.Price == 0 {
			fmt.Printf(" ❌ Got invalid price ($0.00) - is IBKR gateway running?\n")
			errorCount++
			time.Sleep(500 * time.Millisecond)
			continue
		}

		stocks[i].Price = quote.Price

		// Get IV from ATM options (front month)
		iv, err := getImpliedVolatility(client, stocks[i].Ticker, quote.Price)
		if err != nil {
			fmt.Printf(" ⚠️  Price updated ($%.2f), but IV fetch failed: %v\n", quote.Price, err)
			// Still count as partial success - we got the price
			successCount++
		} else {
			stocks[i].IV = iv
			fmt.Printf(" ✅ Price: $%.2f, IV: %.1f%%\n", quote.Price, iv)
			successCount++
		}

		// Rate limiting - be nice to the API
		time.Sleep(1 * time.Second)
	}

	fmt.Printf("\n📈 Update complete: %d succeeded, %d failed\n\n", successCount, errorCount)

	// Write updated data back to CSV
	fmt.Println("💾 Saving updated universe.csv...")
	if err := writeUniverse("data/universe.csv", stocks); err != nil {
		fmt.Printf("❌ Error writing universe.csv: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Universe updated successfully!")
}

func readUniverse(filename string) ([]UniverseStock, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var stocks []UniverseStock
	// Skip header row
	for i, record := range records {
		if i == 0 {
			continue
		}
		if len(record) < 5 {
			continue
		}

		price, _ := strconv.ParseFloat(record[2], 64)
		iv, _ := strconv.ParseFloat(record[3], 64)

		stock := UniverseStock{
			Ticker: record[0],
			Name:   record[1],
			Price:  price,
			IV:     iv,
			Sector: record[4],
		}
		stocks = append(stocks, stock)
	}

	return stocks, nil
}

func writeUniverse(filename string, stocks []UniverseStock) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Ticker", "Name", "Price", "IV", "Sector"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write data rows
	for _, stock := range stocks {
		row := []string{
			stock.Ticker,
			stock.Name,
			fmt.Sprintf("%.2f", stock.Price),
			fmt.Sprintf("%.1f", stock.IV),
			stock.Sector,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// getImpliedVolatility fetches the implied volatility from ATM options
func getImpliedVolatility(client *ibkr.Client, symbol string, currentPrice float64) (float64, error) {
	// Search for the underlying and get front month
	conID, months, err := client.SearchUnderlying(symbol, "NASDAQ")
	if err != nil {
		// Try NYSE if NASDAQ fails
		conID, months, err = client.SearchUnderlying(symbol, "NYSE")
		if err != nil {
			return 0, fmt.Errorf("underlying not found: %w", err)
		}
	}

	if len(months) == 0 {
		return 0, fmt.Errorf("no option months available")
	}

	// Use front month
	month := months[0]

	// Get strikes near current price (±2% range)
	strikeRange := currentPrice * 0.02
	strikes, err := client.GetStrikes(conID, month, currentPrice, strikeRange)
	if err != nil {
		return 0, fmt.Errorf("getting strikes: %w", err)
	}

	if len(strikes) == 0 {
		return 0, fmt.Errorf("no strikes found")
	}

	// Find the ATM strike (closest to current price)
	atmStrike := strikes[0]
	minDiff := math.Abs(strikes[0] - currentPrice)
	for _, strike := range strikes {
		diff := math.Abs(strike - currentPrice)
		if diff < minDiff {
			minDiff = diff
			atmStrike = strike
		}
	}

	// Get both call and put IV for ATM strike
	strikeStr := fmt.Sprintf("%.2f", atmStrike)
	var ivValues []float64

	// Try to get call IV
	callContracts, err := client.GetContractInfo(conID, month, strikeStr, "C")
	if err == nil && len(callContracts) > 0 {
		// Get pricing for first call contract
		pricing, err := client.GetOptionPricing(callContracts[0].ConID)
		if err == nil && pricing.ImpliedVol > 0 {
			ivValues = append(ivValues, pricing.ImpliedVol)
		}
		time.Sleep(300 * time.Millisecond)
	}

	// Try to get put IV
	putContracts, err := client.GetContractInfo(conID, month, strikeStr, "P")
	if err == nil && len(putContracts) > 0 {
		// Get pricing for first put contract
		pricing, err := client.GetOptionPricing(putContracts[0].ConID)
		if err == nil && pricing.ImpliedVol > 0 {
			ivValues = append(ivValues, pricing.ImpliedVol)
		}
	}

	if len(ivValues) == 0 {
		return 0, fmt.Errorf("no valid IV data")
	}

	// Average the IVs
	avgIV := 0.0
	for _, iv := range ivValues {
		avgIV += iv
	}
	avgIV = avgIV / float64(len(ivValues))

	return avgIV, nil
}
