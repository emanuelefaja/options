package analysis

import (
	"fmt"
	"math"
	"mnmlsm/ibkr"
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
		time.Sleep(200 * time.Millisecond)

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
			time.Sleep(200 * time.Millisecond)

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

			// Calculate metrics using per-share pricing
			premiumPerShare := midPrice
			premiumPercent := (premiumPerShare / strike) * 100
			annualizedReturn := (premiumPercent / float64(dte)) * 365

			// Total premium for 100 shares (for display)
			totalPremium := premiumPerShare * 100

			// Filter by minimum return
			if annualizedReturn < params.MinReturn {
				continue
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
				Premium:          totalPremium, // Total for 100 shares
				PremiumPercent:   premiumPercent,
				AnnualizedReturn: annualizedReturn,
				CapitalRequired:  strike * 100, // For cash-secured put
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
