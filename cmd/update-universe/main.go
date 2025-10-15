package main

import (
	"encoding/csv"
	"fmt"
	"mnmlsm/ibkr"
	"os"
	"strconv"
	"sync"
	"time"
)

type UniverseStock struct {
	Ticker string
	Name   string
	Price  float64
	Sector string
}

type updateResult struct {
	index   int
	price   float64
	success bool
	err     error
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

	// Use goroutines to parallelize updates
	const workers = 5 // Run 5 concurrent requests
	results := make(chan updateResult, len(stocks))
	jobs := make(chan int, len(stocks))
	var wg sync.WaitGroup

	// Start worker goroutines
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				result := updateResult{index: i}

				// Get stock price
				quote, err := client.GetQuote(stocks[i].Ticker)
				if err != nil {
					result.err = err
					result.success = false
					results <- result
					continue
				}

				// Check if price is valid
				if quote.Price == 0 {
					result.err = fmt.Errorf("got invalid price ($0.00)")
					result.success = false
					results <- result
					continue
				}

				result.price = quote.Price
				result.success = true
				results <- result

				// Rate limiting per worker
				time.Sleep(200 * time.Millisecond)
			}
		}()
	}

	// Send jobs to workers
	for i := range stocks {
		jobs <- i
	}
	close(jobs)

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process results and print progress
	successCount := 0
	errorCount := 0
	for result := range results {
		i := result.index
		fmt.Printf("[%d/%d] Updating %s (%s)...", i+1, len(stocks), stocks[i].Ticker, stocks[i].Name)

		if result.success {
			stocks[i].Price = result.price
			fmt.Printf(" ✅ Price: $%.2f\n", result.price)
			successCount++
		} else {
			fmt.Printf(" ❌ Failed: %v\n", result.err)
			errorCount++
		}
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
		if len(record) < 4 {
			continue
		}

		price, _ := strconv.ParseFloat(record[2], 64)

		stock := UniverseStock{
			Ticker: record[0],
			Name:   record[1],
			Price:  price,
			Sector: record[3],
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
	header := []string{"Ticker", "Name", "Price", "Sector"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write data rows
	for _, stock := range stocks {
		row := []string{
			stock.Ticker,
			stock.Name,
			fmt.Sprintf("%.2f", stock.Price),
			stock.Sector,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}
