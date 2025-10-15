package analysis

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"mnmlsm/web"
)

// StockCandidate represents a stock that survived elimination filters
type StockCandidate struct {
	Symbol                string
	Name                  string
	Price                 float64
	Sector                string
	PositionCost          float64 // Cost for 100 shares
	PositionSizePercent   float64 // % of total net worth after adding
	SectorExposure        float64 // Total sector capital after adding
	SectorPercent         float64 // % of total net worth in sector after adding
	ExistingStockPosition bool    // Do we already hold stock?
	ExistingPutPosition   bool    // Do we already have a cash-secured put?
	ExistingCapital       float64 // Current capital deployed in this symbol
}

// EliminationResult contains the filtering results
type EliminationResult struct {
	Survivors     []StockCandidate
	Eliminated    map[string]string // symbol -> reason
	TotalNetWorth float64
	DryPowder     float64
}

// RunElimination filters universe.csv through 3 criteria and outputs solar-system.csv
func RunElimination() (*EliminationResult, error) {
	// 1. Calculate total net worth
	totalNetWorth, err := calculateTotalNetWorth()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate total net worth: %w", err)
	}

	// 2. Get dry powder (available capital)
	dryPowder, err := getDryPowder()
	if err != nil {
		return nil, fmt.Errorf("failed to get dry powder: %w", err)
	}

	// 3. Get current positions (stocks + puts)
	stockPositions := getCurrentStockPositions()
	putPositions := getCurrentPutPositions()

	// Combine positions by symbol
	existingCapitalBySymbol := make(map[string]float64)
	hasStockPosition := make(map[string]bool)
	hasPutPosition := make(map[string]bool)

	for symbol, capital := range stockPositions {
		existingCapitalBySymbol[symbol] += capital
		hasStockPosition[symbol] = true
	}

	for symbol, capital := range putPositions {
		existingCapitalBySymbol[symbol] += capital
		hasPutPosition[symbol] = true
	}

	// 4. Get current sector exposure
	sectorExposure := getCurrentSectorExposure()

	// 5. Load universe
	universe, err := loadUniverse()
	if err != nil {
		return nil, fmt.Errorf("failed to load universe: %w", err)
	}

	// 6. Filter each stock
	result := &EliminationResult{
		Survivors:     []StockCandidate{},
		Eliminated:    make(map[string]string),
		TotalNetWorth: totalNetWorth,
		DryPowder:     dryPowder,
	}

	for _, stock := range universe {
		positionCost := stock.Price * 100 // Cost for 100 shares
		existingCapital := existingCapitalBySymbol[stock.Symbol]

		// Filter #1: Available Capital Check
		if positionCost > dryPowder {
			result.Eliminated[stock.Symbol] = fmt.Sprintf("Insufficient capital: need $%.0f, have $%.0f", positionCost, dryPowder)
			continue
		}

		// Filter #2: Position Size Limit (10% max)
		newTotalCapital := existingCapital + positionCost
		positionPercent := (newTotalCapital / totalNetWorth) * 100

		if positionPercent > 10.0 {
			result.Eliminated[stock.Symbol] = fmt.Sprintf("Position too large: %.1f%% of net worth (max 10%%)", positionPercent)
			continue
		}

		// Filter #3: Sector Concentration (20% max)
		currentSectorCapital := sectorExposure[stock.Sector]
		newSectorCapital := currentSectorCapital + positionCost
		sectorPercent := (newSectorCapital / totalNetWorth) * 100

		if sectorPercent > 20.0 {
			result.Eliminated[stock.Symbol] = fmt.Sprintf("Sector too concentrated: %.1f%% in %s (max 20%%)", sectorPercent, stock.Sector)
			continue
		}

		// Stock passed all filters!
		candidate := StockCandidate{
			Symbol:                stock.Symbol,
			Name:                  stock.Name,
			Price:                 stock.Price,
			Sector:                stock.Sector,
			PositionCost:          positionCost,
			PositionSizePercent:   positionPercent,
			SectorExposure:        newSectorCapital,
			SectorPercent:         sectorPercent,
			ExistingStockPosition: hasStockPosition[stock.Symbol],
			ExistingPutPosition:   hasPutPosition[stock.Symbol],
			ExistingCapital:       existingCapital,
		}

		result.Survivors = append(result.Survivors, candidate)
	}

	// 7. Write solar-system.csv
	if err := writeSolarSystemCSV(result.Survivors); err != nil {
		return nil, fmt.Errorf("failed to write solar-system.csv: %w", err)
	}

	return result, nil
}

// UniverseStock represents a stock from universe.csv
type UniverseStock struct {
	Symbol string
	Name   string
	Price  float64
	Sector string
}

// loadUniverse loads stocks from data/universe.csv
func loadUniverse() ([]UniverseStock, error) {
	file, err := os.Open("data/universe.csv")
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
	for i, record := range records {
		if i == 0 || len(record) < 4 {
			continue // Skip header
		}

		price, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			continue
		}

		stocks = append(stocks, UniverseStock{
			Symbol: record[0],
			Name:   record[1],
			Price:  price,
			Sector: record[3],
		})
	}

	return stocks, nil
}

// calculateTotalNetWorth returns portfolio value + Wise balance
func calculateTotalNetWorth() (float64, error) {
	// Load all data
	transactions := web.LoadTransactionsFromCSV("data/transactions.csv")
	optionTransactions := web.LoadOptionTransactions("data/options_transactions.csv")
	stockTransactions := web.LoadStockTransactions("data/stocks_transactions.csv")
	stockPrices := web.LoadStockPrices("data/universe.csv")

	// Calculate portfolio value
	totalDeposits := web.CalculateTotalDeposits(transactions)
	optionPositions := web.CalculateOptionPositions(optionTransactions)
	stockPositions := web.CalculateAllPositions(stockTransactions, stockPrices)

	var totalPremiums float64
	for _, pos := range optionPositions {
		totalPremiums += pos.NetPremium
	}

	var totalStockPL float64
	for _, pos := range stockPositions {
		if pos.Type == "closed" {
			totalStockPL += pos.RealizedPnL
		}
	}

	portfolioValue := totalDeposits + totalPremiums + totalStockPL

	// Get Wise balance
	wiseBalance, err := getWiseBalance()
	if err != nil {
		return 0, err
	}

	return portfolioValue + wiseBalance, nil
}

// getDryPowder returns available cash in brokerage
func getDryPowder() (float64, error) {
	transactions := web.LoadTransactionsFromCSV("data/transactions.csv")
	optionTransactions := web.LoadOptionTransactions("data/options_transactions.csv")
	stockTransactions := web.LoadStockTransactions("data/stocks_transactions.csv")
	stockPrices := web.LoadStockPrices("data/universe.csv")

	totalDeposits := web.CalculateTotalDeposits(transactions)
	optionPositions := web.CalculateOptionPositions(optionTransactions)
	stockPositions := web.CalculateAllPositions(stockTransactions, stockPrices)

	var totalPremiums float64
	for _, pos := range optionPositions {
		totalPremiums += pos.NetPremium
	}

	var totalStockPL float64
	var activeCapital float64

	for _, pos := range stockPositions {
		if pos.Type == "closed" {
			totalStockPL += pos.RealizedPnL
		} else if pos.Type == "open" {
			activeCapital += pos.CostBasis
		}
	}

	// Add open put options to active capital
	for _, pos := range optionPositions {
		if pos.Status == "Open" && pos.OptionType == "Put" {
			activeCapital += pos.Capital
		}
	}

	dryPowder := totalDeposits + totalPremiums + totalStockPL - activeCapital
	return dryPowder, nil
}

// getCurrentStockPositions returns map of symbol -> cost basis for open stock positions
func getCurrentStockPositions() map[string]float64 {
	stockTransactions := web.LoadStockTransactions("data/stocks_transactions.csv")
	stockPrices := web.LoadStockPrices("data/universe.csv")
	positions := web.CalculateAllPositions(stockTransactions, stockPrices)

	result := make(map[string]float64)
	for _, pos := range positions {
		if pos.Type == "open" {
			result[pos.Symbol] = pos.CostBasis
		}
	}

	return result
}

// getCurrentPutPositions returns map of symbol -> capital for open cash-secured puts
func getCurrentPutPositions() map[string]float64 {
	optionTransactions := web.LoadOptionTransactions("data/options_transactions.csv")
	positions := web.CalculateOptionPositions(optionTransactions)

	result := make(map[string]float64)
	for _, pos := range positions {
		if pos.Status == "Open" && pos.OptionType == "Put" {
			result[pos.Symbol] += pos.Capital
		}
	}

	return result
}

// getCurrentSectorExposure returns map of sector -> capital
func getCurrentSectorExposure() map[string]float64 {
	exposures := web.CalculateSectorExposure()

	result := make(map[string]float64)
	for _, exp := range exposures {
		result[exp.Sector] = exp.Amount
	}

	return result
}

// getWiseBalance returns the latest Wise account balance
func getWiseBalance() (float64, error) {
	file, err := os.Open("data/wise.csv")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return 0, err
	}

	if len(records) < 2 {
		return 0, fmt.Errorf("wise.csv has insufficient data")
	}

	// Get last row (most recent month)
	lastRow := records[len(records)-1]
	if len(lastRow) < 2 {
		return 0, fmt.Errorf("wise.csv row format incorrect")
	}

	balance, err := strconv.ParseFloat(lastRow[1], 64)
	if err != nil {
		return 0, err
	}

	return balance, nil
}

// writeSolarSystemCSV writes the surviving candidates to data/solar-system.csv
func writeSolarSystemCSV(candidates []StockCandidate) error {
	file, err := os.Create("data/solar-system.csv")
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"Symbol", "Name", "Price", "Sector",
		"PositionCost", "PositionSizePercent",
		"SectorExposure", "SectorPercent",
		"ExistingStockPosition", "ExistingPutPosition",
		"ExistingCapital",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write rows
	for _, c := range candidates {
		row := []string{
			c.Symbol,
			c.Name,
			fmt.Sprintf("%.2f", c.Price),
			c.Sector,
			fmt.Sprintf("%.2f", c.PositionCost),
			fmt.Sprintf("%.2f", c.PositionSizePercent),
			fmt.Sprintf("%.2f", c.SectorExposure),
			fmt.Sprintf("%.2f", c.SectorPercent),
			fmt.Sprintf("%t", c.ExistingStockPosition),
			fmt.Sprintf("%t", c.ExistingPutPosition),
			fmt.Sprintf("%.2f", c.ExistingCapital),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}
