package analysis

import (
	"encoding/csv"
	"fmt"
	"math"
	"mnmlsm/ibkr"
	"os"
	"sort"
	"strconv"
	"time"
)

// Scanner performs options premium scanning using an IBKR client
type Scanner struct {
	client *ibkr.Client
}

// NewScanner creates a new premium scanner
func NewScanner(client *ibkr.Client) *Scanner {
	return &Scanner{
		client: client,
	}
}

// ScanPremiums scans for option premium opportunities based on given parameters
// Returns a list of OptionContracts that meet the criteria
func (s *Scanner) ScanPremiums(params ScanParams) ([]OptionContract, error) {
	// 1. Search for underlying and get option months
	conID, months, err := s.client.SearchUnderlying(params.Symbol, params.Exchange)
	if err != nil {
		return nil, fmt.Errorf("searching underlying: %w", err)
	}

	// 2. Get current stock price
	currentPrice, err := s.client.GetLastPrice(conID)
	if err != nil {
		return nil, fmt.Errorf("getting current price: %w", err)
	}

	// 3. Use the first (front) month for scanning
	if len(months) == 0 {
		return nil, fmt.Errorf("no option months available")
	}
	month := months[0]

	// 4. Scan the front month for qualifying contracts
	var qualifyingContracts []OptionContract

	// Get strikes for this month
	strikes, err := s.client.GetStrikes(conID, month, currentPrice, params.StrikeRange)
	if err != nil {
		return nil, fmt.Errorf("getting strikes: %w", err)
	}

	// Process each strike
	for _, strike := range strikes {
		strikeStr := fmt.Sprintf("%.2f", strike)

		// Get contract info
		contracts, err := s.client.GetContractInfo(conID, month, strikeStr, params.Right)
		if err != nil {
			continue // Skip strikes with errors
		}

		// Rate limiting
		time.Sleep(20 * time.Millisecond) // 50 req/s rate limit (60 max)

		// Process each contract (usually multiple expiries per strike/month)
		for _, contract := range contracts {
			// Calculate DTE first to filter
			dte := CalculateDaysToExpiry(contract.MaturityDate)

			// Skip if beyond max DTE
			if params.MaxDTE > 0 && dte > params.MaxDTE {
				continue
			}

			// Get pricing and greeks
			pricing, err := s.client.GetOptionPricing(contract.ConID)
			if err != nil {
				continue // Skip contracts with pricing errors
			}

			// Rate limiting
			time.Sleep(20 * time.Millisecond) // 50 req/s rate limit (60 max)

			// Skip if no valid bid or ask
			if pricing.Bid <= 0 && pricing.Ask <= 0 {
				continue
			}

			// Calculate mid price (or use bid/ask if one is missing)
			midPrice := pricing.Bid
			if pricing.Ask > 0 {
				if pricing.Bid > 0 {
					midPrice = (pricing.Bid + pricing.Ask) / 2
				} else {
					midPrice = pricing.Ask
				}
			}

			// Calculate intrinsic and extrinsic value
			var intrinsicValue float64
			var isITM bool

			if params.Right == "P" {
				// Put: intrinsic = max(0, strike - stock price)
				intrinsicValue = math.Max(0, strike-currentPrice)
				isITM = strike > currentPrice
			} else {
				// Call: intrinsic = max(0, stock price - strike)
				intrinsicValue = math.Max(0, currentPrice-strike)
				isITM = currentPrice > strike
			}

			// Extrinsic value (time premium) = total premium - intrinsic
			extrinsicValue := math.Max(0, midPrice-intrinsicValue)

			// Calculate metrics using EXTRINSIC VALUE (time premium only)
			// This represents the actual return on your capital, not just ITM movement
			premiumPercent := (extrinsicValue / strike) * 100
			annualizedReturn := (premiumPercent / float64(dte)) * 365

			// Total premium for 100 shares (for display)
			totalPremium := midPrice * 100
			totalExtrinsic := extrinsicValue * 100
			totalIntrinsic := intrinsicValue * 100

			// Filter by minimum return (based on extrinsic value)
			if annualizedReturn < params.MinReturn {
				continue
			}

			// Calculate Probability of Profit (1 - |Delta|)
			pop := (1 - math.Abs(pricing.Delta)) * 100

			// Calculate Efficiency (risk-adjusted return)
			// Efficiency = AnnualizedReturn / (1 - POP)
			efficiency := 0.0
			if pop < 100 {
				efficiency = annualizedReturn / (1 - (pop / 100))
			}

			// Build OptionContract
			optContract := OptionContract{
				Symbol:           params.Symbol,
				Strike:           strike,
				Right:            params.Right,
				MaturityDate:     contract.MaturityDate,
				ConID:            contract.ConID,
				UnderlyingConID:  conID,
				Bid:              pricing.Bid,
				Ask:              pricing.Ask,
				MidPrice:         midPrice,
				UnderlyingPrice:  currentPrice,
				Delta:            pricing.Delta,
				Gamma:            pricing.Gamma,
				Theta:            pricing.Theta,
				Vega:             pricing.Vega,
				ImpliedVol:       pricing.ImpliedVol,
				DTE:              dte,
				Premium:          totalPremium,    // Total for 100 shares
				IntrinsicValue:   totalIntrinsic,  // Intrinsic for 100 shares
				ExtrinsicValue:   totalExtrinsic,  // Extrinsic for 100 shares
				PremiumPercent:   premiumPercent,  // Based on extrinsic
				AnnualizedReturn: annualizedReturn, // Based on extrinsic
				CapitalRequired:  strike * 100,     // For cash-secured put
				POP:              pop,
				Efficiency:       efficiency,
				IsITM:            isITM,
			}

			qualifyingContracts = append(qualifyingContracts, optContract)
		}
	}

	return qualifyingContracts, nil
}

// CalculateDaysToExpiry calculates days until option expiration
func CalculateDaysToExpiry(maturityDate string) int {
	// Parse maturity date (format: "20241220")
	expiryTime, err := time.Parse("20060102", maturityDate)
	if err != nil {
		return 0
	}

	// Set to market close time (4 PM ET)
	expiry := time.Date(expiryTime.Year(), expiryTime.Month(), expiryTime.Day(),
		16, 0, 0, 0, time.FixedZone("EST", -5*3600))

	now := time.Now()
	duration := expiry.Sub(now)
	days := int(math.Round(duration.Hours() / 24))

	if days < 0 {
		return 0
	}
	return days
}

// CalculateAnnualizedReturn calculates annualized return percentage
func CalculateAnnualizedReturn(premium, capitalRequired float64, days int) float64 {
	if days == 0 || capitalRequired == 0 {
		return 0
	}

	returnPercent := (premium / capitalRequired) * 100
	annualized := (returnPercent / float64(days)) * 365

	return annualized
}

// filterMonthsByDTE filters option months by maximum DTE
func (s *Scanner) filterMonthsByDTE(months []string, maxDTE int) []string {
	now := time.Now()
	var validMonths []string

	for _, month := range months {
		// Parse month string (format: "JAN24", "FEB24", etc.)
		monthDate, err := parseMonthString(month)
		if err != nil {
			continue
		}

		// Calculate DTE (third Friday of the month)
		expiry := getThirdFriday(monthDate)
		dte := int(math.Round(expiry.Sub(now).Hours() / 24))

		if dte >= 0 && dte <= maxDTE {
			validMonths = append(validMonths, month)
		}
	}

	return validMonths
}

// Helper functions

func parseMonthString(month string) (time.Time, error) {
	// Format: "JAN24" → 2024-01-01
	if len(month) < 5 {
		return time.Time{}, fmt.Errorf("invalid month format: %s", month)
	}

	monthMap := map[string]int{
		"JAN": 1, "FEB": 2, "MAR": 3, "APR": 4,
		"MAY": 5, "JUN": 6, "JUL": 7, "AUG": 8,
		"SEP": 9, "OCT": 10, "NOV": 11, "DEC": 12,
	}

	monthStr := month[:3]
	yearStr := "20" + month[3:5]

	monthNum, ok := monthMap[monthStr]
	if !ok {
		return time.Time{}, fmt.Errorf("invalid month: %s", monthStr)
	}

	year := 0
	fmt.Sscanf(yearStr, "%d", &year)

	return time.Date(year, time.Month(monthNum), 1, 0, 0, 0, 0, time.UTC), nil
}

func getThirdFriday(monthDate time.Time) time.Time {
	// Start at the first day of the month
	current := time.Date(monthDate.Year(), monthDate.Month(), 1, 16, 0, 0, 0, time.UTC)

	// Find first Friday
	for current.Weekday() != time.Friday {
		current = current.AddDate(0, 0, 1)
	}

	// Add two weeks to get third Friday
	return current.AddDate(0, 0, 14)
}

// ScanAllStocks scans all stocks from solar-system.csv and saves to options-chain.csv
func (s *Scanner) ScanAllStocks(params BatchScanParams) error {
	// Load stocks from solar-system.csv
	stocks, err := loadSolarSystem(params.SolarSystemCSV)
	if err != nil {
		return fmt.Errorf("loading solar-system.csv: %w", err)
	}

	// Initialize output CSV
	if err := initializeCSV(params.OutputCSV); err != nil {
		return fmt.Errorf("initializing CSV: %w", err)
	}

	fmt.Printf("🪐 Scanning %d stocks from solar-system.csv\n", len(stocks))
	fmt.Printf("   Right: %s, Min Return: %.0f%%, Expiries: %d\n\n", params.Right, params.MinReturn, params.NumExpiries)

	totalContracts := 0
	successCount := 0
	failedStocks := []string{}

	for i, stock := range stocks {
		fmt.Printf("[%d/%d] Processing %s...\n", i+1, len(stocks), stock.Symbol)

		// Scan this stock
		contracts, err := s.scanStockMultiExpiry(stock, params)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
			failedStocks = append(failedStocks, fmt.Sprintf("%s: %v", stock.Symbol, err))
			continue
		}

		// Save all contracts to CSV
		for _, contract := range contracts {
			if err := appendContractToCSV(contract, params.OutputCSV); err != nil {
				return fmt.Errorf("appending to CSV: %w", err)
			}
			totalContracts++
		}

		fmt.Printf("   ✅ Found %d contracts\n\n", len(contracts))
		successCount++
	}

	// Summary
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("✨ Scan Complete!\n")
	fmt.Printf("   Success: %d/%d stocks\n", successCount, len(stocks))
	fmt.Printf("   Total Contracts: %d\n", totalContracts)
	fmt.Printf("   Saved to: %s\n", params.OutputCSV)

	if len(failedStocks) > 0 {
		fmt.Printf("\n❌ Failed stocks:\n")
		for _, failure := range failedStocks {
			fmt.Printf("   %s\n", failure)
		}
	}

	return nil
}

// scanStockMultiExpiry scans one stock across multiple expiries
func (s *Scanner) scanStockMultiExpiry(stock SolarSystemStock, params BatchScanParams) ([]OptionContract, error) {
	// Get underlying and option months
	// Use NASDAQ as default exchange (matches ibkr-quote behavior)
	conID, months, err := s.client.SearchUnderlying(stock.Symbol, "NASDAQ")
	if err != nil {
		return nil, fmt.Errorf("searching underlying: %w", err)
	}

	// Get current stock price
	currentPrice, err := s.client.GetLastPrice(conID)
	if err != nil {
		return nil, fmt.Errorf("getting price: %w", err)
	}

	fmt.Printf("   Price: $%.2f\n", currentPrice)

	// Get next N Friday expiries
	targetExpiries := getNextFridayExpiries(months, params.NumExpiries)
	if len(targetExpiries) == 0 {
		return nil, fmt.Errorf("no valid expiries found")
	}

	fmt.Printf("   Expiries: %s\n", formatExpiries(targetExpiries))

	var allContracts []OptionContract

	// Scan each expiry
	for _, month := range targetExpiries {
		// Get strikes
		strikes, err := s.client.GetStrikes(conID, month, currentPrice, params.StrikeRange)
		if err != nil {
			fmt.Printf("   ⚠️  Skipping %s: %v\n", month, err)
			continue
		}

		expiryContracts := 0

		// Process each strike
		for _, strike := range strikes {
			strikeStr := fmt.Sprintf("%.2f", strike)

			// Get contract info
			contracts, err := s.client.GetContractInfo(conID, month, strikeStr, params.Right)
			if err != nil {
				continue
			}

			time.Sleep(20 * time.Millisecond) // 50 req/s rate limit (60 max)

			// Process each contract
			for _, contract := range contracts {
				dte := CalculateDaysToExpiry(contract.MaturityDate)

				// Get pricing
				pricing, err := s.client.GetOptionPricing(contract.ConID)
				if err != nil {
					continue
				}

				time.Sleep(20 * time.Millisecond) // 50 req/s rate limit (60 max)

				// Skip if no valid bid or ask
				if pricing.Bid <= 0 && pricing.Ask <= 0 {
					continue
				}

				// Calculate mid price
				midPrice := pricing.Bid
				if pricing.Ask > 0 {
					if pricing.Bid > 0 {
						midPrice = (pricing.Bid + pricing.Ask) / 2
					} else {
						midPrice = pricing.Ask
					}
				}

				// Calculate intrinsic and extrinsic value
				var intrinsicValue float64
				var isITM bool

				if params.Right == "P" {
					intrinsicValue = math.Max(0, strike-currentPrice)
					isITM = strike > currentPrice
				} else {
					intrinsicValue = math.Max(0, currentPrice-strike)
					isITM = currentPrice > strike
				}

				extrinsicValue := math.Max(0, midPrice-intrinsicValue)

				// Calculate metrics
				premiumPercent := (extrinsicValue / strike) * 100
				annualizedReturn := (premiumPercent / float64(dte)) * 365

				// Filter by minimum return
				if annualizedReturn < params.MinReturn {
					continue
				}

				totalPremium := midPrice * 100
				totalExtrinsic := extrinsicValue * 100
				totalIntrinsic := intrinsicValue * 100

				// Calculate POP and Efficiency
				pop := (1 - math.Abs(pricing.Delta)) * 100
				efficiency := 0.0
				if pop < 100 {
					efficiency = annualizedReturn / (1 - (pop / 100))
				}

				// Build contract
				optContract := OptionContract{
					Symbol:           stock.Symbol,
					Strike:           strike,
					Right:            params.Right,
					MaturityDate:     contract.MaturityDate,
					ConID:            contract.ConID,
					UnderlyingConID:  conID,
					Bid:              pricing.Bid,
					Ask:              pricing.Ask,
					MidPrice:         midPrice,
					UnderlyingPrice:  currentPrice,
					Delta:            pricing.Delta,
					Gamma:            pricing.Gamma,
					Theta:            pricing.Theta,
					Vega:             pricing.Vega,
					ImpliedVol:       pricing.ImpliedVol,
					DTE:              dte,
					Premium:          totalPremium,
					IntrinsicValue:   totalIntrinsic,
					ExtrinsicValue:   totalExtrinsic,
					PremiumPercent:   premiumPercent,
					AnnualizedReturn: annualizedReturn,
					CapitalRequired:  strike * 100,
					POP:              pop,
					Efficiency:       efficiency,
					IsITM:            isITM,
				}

				allContracts = append(allContracts, optContract)
				expiryContracts++

				// Progress feedback
				itmStr := "OTM"
				if isITM {
					itmStr = "ITM"
				}
				fmt.Printf("      $%.2f (%s, %dd): $%.0f → %.0f%% ann\n",
					strike, itmStr, dte, totalExtrinsic, annualizedReturn)
			}
		}

		if expiryContracts > 0 {
			fmt.Printf("   📅 %s: %d contracts\n", month, expiryContracts)
		}
	}

	return allContracts, nil
}

// getNextFridayExpiries returns the next N Friday expiries from available months
func getNextFridayExpiries(months []string, count int) []string {
	type expiryDate struct {
		month string
		date  time.Time
	}

	now := time.Now()
	var expiries []expiryDate

	for _, month := range months {
		monthDate, err := parseMonthString(month)
		if err != nil {
			continue
		}

		expiry := getThirdFriday(monthDate)

		// Only include future expiries
		if expiry.After(now) {
			expiries = append(expiries, expiryDate{
				month: month,
				date:  expiry,
			})
		}
	}

	// Sort by date ascending
	sort.Slice(expiries, func(i, j int) bool {
		return expiries[i].date.Before(expiries[j].date)
	})

	// Take first N
	result := []string{}
	for i := 0; i < count && i < len(expiries); i++ {
		result = append(result, expiries[i].month)
	}

	return result
}

// formatExpiries formats expiry months for display
func formatExpiries(months []string) string {
	if len(months) == 0 {
		return "none"
	}

	result := ""
	for i, month := range months {
		if i > 0 {
			result += ", "
		}
		// Parse and format nicely
		monthDate, err := parseMonthString(month)
		if err == nil {
			expiry := getThirdFriday(monthDate)
			result += expiry.Format("Jan 2")
		} else {
			result += month
		}
	}
	return result
}

// SolarSystemStock represents a stock from solar-system.csv
type SolarSystemStock struct {
	Symbol string
	Price  float64
}

// loadSolarSystem loads stocks from solar-system.csv
func loadSolarSystem(filepath string) ([]SolarSystemStock, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var stocks []SolarSystemStock
	for i, record := range records {
		if i == 0 || len(record) < 3 {
			continue // Skip header
		}

		price, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			continue
		}

		stocks = append(stocks, SolarSystemStock{
			Symbol: record[0],
			Price:  price,
		})
	}

	return stocks, nil
}

// initializeCSV creates the CSV file with header row
func initializeCSV(filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{
		"Symbol", "Strike", "Right", "MaturityDate", "DTE",
		"Premium", "IntrinsicValue", "ExtrinsicValue",
		"PremiumPercent", "AnnualizedReturn", "POP", "Efficiency",
		"ITM", "Delta", "Gamma", "Theta", "Vega", "ImpliedVol",
		"Bid", "Ask", "MidPrice", "UnderlyingPrice",
		"CapitalRequired", "ConID", "UnderlyingConID",
	}

	return writer.Write(header)
}

// appendContractToCSV appends one contract to the CSV file
func appendContractToCSV(contract OptionContract, filepath string) error {
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	itmStr := "false"
	if contract.IsITM {
		itmStr = "true"
	}

	row := []string{
		contract.Symbol,
		fmt.Sprintf("%.2f", contract.Strike),
		contract.Right,
		contract.MaturityDate,
		fmt.Sprintf("%d", contract.DTE),
		fmt.Sprintf("%.2f", contract.Premium),
		fmt.Sprintf("%.2f", contract.IntrinsicValue),
		fmt.Sprintf("%.2f", contract.ExtrinsicValue),
		fmt.Sprintf("%.2f", contract.PremiumPercent),
		fmt.Sprintf("%.2f", contract.AnnualizedReturn),
		fmt.Sprintf("%.2f", contract.POP),
		fmt.Sprintf("%.2f", contract.Efficiency),
		itmStr,
		fmt.Sprintf("%.4f", contract.Delta),
		fmt.Sprintf("%.4f", contract.Gamma),
		fmt.Sprintf("%.4f", contract.Theta),
		fmt.Sprintf("%.4f", contract.Vega),
		fmt.Sprintf("%.4f", contract.ImpliedVol),
		fmt.Sprintf("%.2f", contract.Bid),
		fmt.Sprintf("%.2f", contract.Ask),
		fmt.Sprintf("%.2f", contract.MidPrice),
		fmt.Sprintf("%.2f", contract.UnderlyingPrice),
		fmt.Sprintf("%.2f", contract.CapitalRequired),
		fmt.Sprintf("%d", contract.ConID),
		fmt.Sprintf("%d", contract.UnderlyingConID),
	}

	return writer.Write(row)
}
