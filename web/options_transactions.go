package web

import (
	"encoding/csv"
	"log"
	"math"
	"os"
	"strconv"
	"time"
)

type OptionTransaction struct {
	Date        string
	Action      string  // "Sell to Open", "Buy to Close", "Expired", "Assigned", "Exercised"
	Symbol      string
	OptionType  string  // "Call" or "Put"
	Strike      float64
	Expiry      string
	Contracts   int
	Premium     float64  // Positive for credit, negative for debit
	StockPrice  float64
	Commission  float64
	PositionID  string
	Notes       string
}

type OptionPosition struct {
	PositionID        string
	Symbol            string
	OptionType        string
	Strike            float64
	Expiry            string
	Contracts         int
	Status            string  // "Open", "Expired", "Assigned", "Closed Early", "Rolled"
	OpenDate          string
	CloseDate         string
	PremiumCollected  float64
	PremiumPaid       float64
	NetPremium        float64
	Commissions       float64
	MaxProfit         float64
	DaysHeld          int
	DaysToExpiry      int
	AnnualizedReturn  float64
	PercentReturn     float64
	Capital           float64  // For calculating returns
}

func LoadOptionTransactions(filename string) []OptionTransaction {
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Error opening options transactions CSV file: %v", err)
		return []OptionTransaction{}
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Printf("Error reading options transactions CSV file: %v", err)
		return []OptionTransaction{}
	}

	var transactions []OptionTransaction
	for i, record := range records {
		if i == 0 || len(record) < 12 {
			continue
		}

		strike, _ := strconv.ParseFloat(record[4], 64)
		contracts, _ := strconv.Atoi(record[6])
		premium, _ := strconv.ParseFloat(record[7], 64)
		stockPrice, _ := strconv.ParseFloat(record[8], 64)
		commission, _ := strconv.ParseFloat(record[9], 64)

		transaction := OptionTransaction{
			Date:        record[0],
			Action:      record[1],
			Symbol:      record[2],
			OptionType:  record[3],
			Strike:      strike,
			Expiry:      record[5],
			Contracts:   contracts,
			Premium:     premium,
			StockPrice:  stockPrice,
			Commission:  commission,
			PositionID:  record[10],
			Notes:       record[11],
		}
		transactions = append(transactions, transaction)
	}
	return transactions
}

func CalculateOptionPositions(transactions []OptionTransaction) []OptionPosition {
	positionMap := make(map[string]*OptionPosition)

	for _, tx := range transactions {
		if tx.PositionID == "" {
			continue
		}

		pos, exists := positionMap[tx.PositionID]
		if !exists {
			// New position
			pos = &OptionPosition{
				PositionID: tx.PositionID,
				Symbol:     tx.Symbol,
				OptionType: tx.OptionType,
				Strike:     tx.Strike,
				Expiry:     tx.Expiry,
				Contracts:  tx.Contracts,
				Status:     "Open",
			}
			positionMap[tx.PositionID] = pos
		}

		// Process the transaction
		switch tx.Action {
		case "Sell to Open":
			pos.OpenDate = tx.Date
			pos.PremiumCollected += tx.Premium
			pos.Commissions += tx.Commission

			// Calculate capital requirement
			if tx.OptionType == "Put" {
				// Cash-secured put
				pos.Capital = tx.Strike * float64(tx.Contracts) * 100
			} else {
				// Covered call - use stock price as capital
				if tx.StockPrice > 0 {
					pos.Capital = tx.StockPrice * float64(tx.Contracts) * 100
				} else {
					pos.Capital = tx.Strike * float64(tx.Contracts) * 100
				}
			}

		case "Buy to Close":
			pos.PremiumPaid += math.Abs(tx.Premium)
			pos.Commissions += tx.Commission
			pos.CloseDate = tx.Date
			pos.Status = "Closed Early"

		case "Expired":
			pos.CloseDate = tx.Date
			pos.Status = "Expired"

		case "Assigned":
			pos.CloseDate = tx.Date
			pos.Status = "Assigned"

		case "Exercised":
			pos.CloseDate = tx.Date
			pos.Status = "Exercised"
		}
	}

	// Calculate metrics for each position
	var positions []OptionPosition
	for _, pos := range positionMap {
		// Calculate net premium
		pos.NetPremium = pos.PremiumCollected - pos.PremiumPaid - pos.Commissions
		pos.MaxProfit = pos.PremiumCollected

		// Calculate days held
		if pos.OpenDate != "" && pos.CloseDate != "" {
			openTime, _ := time.Parse("2006-01-02", pos.OpenDate)
			closeTime, _ := time.Parse("2006-01-02", pos.CloseDate)
			pos.DaysHeld = int(closeTime.Sub(openTime).Hours() / 24)
		}

		// Calculate days to expiry from open date
		if pos.OpenDate != "" && pos.Expiry != "" {
			openTime, _ := time.Parse("2006-01-02", pos.OpenDate)
			expiryTime, _ := time.Parse("2006-01-02", pos.Expiry)
			pos.DaysToExpiry = int(expiryTime.Sub(openTime).Hours() / 24)
			if pos.DaysToExpiry < 1 {
				pos.DaysToExpiry = 1
			}
		}

		// Calculate returns
		if pos.Capital > 0 {
			pos.PercentReturn = (pos.NetPremium / pos.Capital) * 100

			// Calculate annualized return based on days to expiry (not days held)
			if pos.DaysToExpiry > 0 {
				pos.AnnualizedReturn = (pos.PercentReturn / float64(pos.DaysToExpiry)) * 365
			}
		}

		// If position is still open and close date is empty, check expiry
		if pos.Status == "Open" && pos.CloseDate == "" {
			if expiryTime, err := time.Parse("2006-01-02", pos.Expiry); err == nil {
				if time.Now().After(expiryTime) {
					// Position has expired but not marked
					pos.Status = "Expired"
					pos.CloseDate = pos.Expiry
				}
			}
		}

		positions = append(positions, *pos)
	}

	return positions
}

// Helper function to format currency for display
func (p OptionPosition) FormatNetPremium() string {
	return FormatCurrency(p.NetPremium)
}

func (p OptionPosition) FormatCapital() string {
	return FormatCurrency(p.Capital)
}

func (p OptionPosition) FormatPercentReturn() string {
	return FormatPercentage(p.PercentReturn)
}

func (p OptionPosition) FormatAnnualizedReturn() string {
	return FormatPercentage(p.AnnualizedReturn)
}