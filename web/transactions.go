package web

import (
	"encoding/csv"
	"log"
	"os"
	"strconv"
	"strings"
)

type Transaction struct {
	Date   string
	Type   string
	Amount string
}

func LoadTransactionsFromCSV(filename string) []Transaction {
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Error opening transactions CSV file: %v", err)
		return []Transaction{}
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Printf("Error reading transactions CSV file: %v", err)
		return []Transaction{}
	}

	var transactions []Transaction
	for i, record := range records {
		if i == 0 {
			continue
		}
		if len(record) >= 3 {
			transaction := Transaction{
				Date:   record[0],
				Type:   record[1],
				Amount: record[2],
			}
			transactions = append(transactions, transaction)
		}
	}
	return transactions
}

func CalculateTotalDeposits(transactions []Transaction) float64 {
	var total float64
	for _, t := range transactions {
		if t.Type == "Deposit" {
			amount := strings.TrimPrefix(t.Amount, "$")
			amount = strings.ReplaceAll(amount, ",", "")
			if a, err := strconv.ParseFloat(amount, 64); err == nil {
				total += a
			}
		}
	}
	return total
}