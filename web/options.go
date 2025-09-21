package web

import (
	"encoding/csv"
	"log"
	"os"
)

type Trade struct {
	Stock           string
	Type            string
	Outcome         string
	DateOfTrade     string
	Expiry          string
	Days            string
	PremiumDollar   string
	Premium         string
	StockPrice      string
	Strike          string
	Contracts       string
	Capital         string
	ReturnOnStock   string
	AnnualizedReturn string
	PercentReturn   string
	IncomePerDay    string
	Notes           string
	OptionsID       string
}

func LoadTradesFromCSV(filename string) []Trade {
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Error opening CSV file: %v", err)
		return []Trade{}
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Printf("Error reading CSV file: %v", err)
		return []Trade{}
	}

	var trades []Trade
	for i, record := range records {
		if i == 0 {
			continue
		}
		if len(record) >= 18 {
			trade := Trade{
				Stock:            record[0],
				Type:             record[1],
				Outcome:          record[2],
				DateOfTrade:      record[3],
				Expiry:           record[4],
				Days:             record[5],
				PremiumDollar:    record[6],
				Premium:          record[7],
				StockPrice:       record[8],
				Strike:           record[9],
				Contracts:        record[10],
				Capital:          record[11],
				ReturnOnStock:    record[12],
				AnnualizedReturn: record[13],
				PercentReturn:    record[14],
				IncomePerDay:     record[15],
				Notes:            record[16],
				OptionsID:        record[17],
			}
			trades = append(trades, trade)
		}
	}
	return trades
}