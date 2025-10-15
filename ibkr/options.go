package ibkr

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
	"time"
)

// SearchUnderlying searches for an underlying security and returns its ConID and available option months
func (c *Client) SearchUnderlying(symbol, exchange string) (int, []string, error) {
	url := fmt.Sprintf("%s/iserver/secdef/search?symbol=%s", c.baseURL, symbol)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return 0, nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("reading response: %w", err)
	}

	var results []SearchResult
	if err := json.Unmarshal(body, &results); err != nil {
		return 0, nil, fmt.Errorf("parsing search results: %w", err)
	}

	// Find the contract matching the exchange and with OPT section
	for _, contract := range results {
		if contract.Description == exchange {
			// Parse ConID
			var conID int
			if _, err := fmt.Sscanf(contract.ConID, "%d", &conID); err != nil {
				continue
			}

			// Find OPT section
			for _, section := range contract.Sections {
				if section.SecType == "OPT" {
					// Parse months (semicolon-separated)
					months := strings.Split(section.Months, ";")
					return conID, months, nil
				}
			}
		}
	}

	return 0, nil, fmt.Errorf("no options found for %s on %s", symbol, exchange)
}

// GetStrikes fetches available strikes for a given option month
// strikeRange limits results to strikes within +/- strikeRange of currentPrice
func (c *Client) GetStrikes(conid int, month string, currentPrice, strikeRange float64) ([]float64, error) {
	url := fmt.Sprintf("%s/iserver/secdef/strikes?conid=%d&sectype=OPT&month=%s",
		c.baseURL, conid, month)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching strikes: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var strikes StrikesResponse
	if err := json.Unmarshal(body, &strikes); err != nil {
		return nil, fmt.Errorf("parsing strikes: %w", err)
	}

	// Filter strikes within range if specified
	if strikeRange > 0 {
		minStrike := currentPrice - strikeRange
		maxStrike := currentPrice + strikeRange

		filtered := make([]float64, 0)
		for _, strike := range strikes.Put {
			if strike >= minStrike && strike <= maxStrike {
				filtered = append(filtered, strike)
			}
		}
		return filtered, nil
	}

	return strikes.Put, nil
}

// GetContractInfo fetches detailed contract information for a specific option
func (c *Client) GetContractInfo(conid int, month, strike, right string) ([]ContractInfo, error) {
	url := fmt.Sprintf("%s/iserver/secdef/info?conid=%d&sectype=OPT&month=%s&strike=%s&right=%s",
		c.baseURL, conid, month, strike, right)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching contract info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var contracts []ContractInfo
	if err := json.Unmarshal(body, &contracts); err != nil {
		return nil, fmt.Errorf("parsing contract info: %w", err)
	}

	return contracts, nil
}

// GetOptionPricing fetches bid/ask and greeks for an option contract
func (c *Client) GetOptionPricing(conid int) (*OptionPricing, error) {
	// Request fields:
	// 84 = Bid, 85 = Ask, 86 = Bid Size, 88 = Ask Size
	// 31 = Last, 7283 = Implied Vol, 7308 = Delta
	fields := "31,84,85,86,88,7283,7308"
	url := fmt.Sprintf("%s/iserver/marketdata/snapshot?conids=%d&fields=%s",
		c.baseURL, conid, fields)

	// Preflight request to initialize
	c.httpClient.Get(url)
	time.Sleep(300 * time.Millisecond)

	// Actual request
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching option pricing: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var data []map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parsing market data: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no pricing data returned for conid %d", conid)
	}

	item := data[0]
	pricing := &OptionPricing{}

	pricing.Bid = parseFieldValue(item["84"])
	pricing.Ask = parseFieldValue(item["85"])
	pricing.LastPrice = parseFieldValue(item["31"])
	pricing.ImpliedVol = parseFieldValue(item["7283"])
	pricing.Delta = parseFieldValue(item["7308"])

	return pricing, nil
}

// parseFieldValue extracts float value from various field formats
func parseFieldValue(field interface{}) float64 {
	if field == nil {
		return 0
	}

	switch val := field.(type) {
	case float64:
		return val
	case string:
		var f float64
		fmt.Sscanf(val, "%f", &f)
		return f
	case map[string]interface{}:
		if v, ok := val["v"]; ok {
			return parseFieldValue(v)
		}
	}

	return 0
}

// GetLastPrice fetches the current price for a security
func (c *Client) GetLastPrice(conid int) (float64, error) {
	url := fmt.Sprintf("%s/iserver/marketdata/snapshot?conids=%d&fields=31",
		c.baseURL, conid)

	// Preflight request to initialize
	c.httpClient.Get(url)
	time.Sleep(1 * time.Second)

	// Actual request
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return 0, fmt.Errorf("fetching price: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("reading response: %w", err)
	}

	var data []map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, fmt.Errorf("parsing market data: %w", err)
	}

	if len(data) == 0 {
		return 0, fmt.Errorf("no price data returned for conid %d", conid)
	}

	field31 := data[0]["31"]

	// Try map format first ({"v": value})
	if field31Map, ok := field31.(map[string]interface{}); ok {
		if priceVal, ok := field31Map["v"]; ok {
			return parseFloat(priceVal), nil
		}
	}

	// Try direct value
	return parseFloat(field31), nil
}

// GetOptionChain is a higher-level function that fetches the complete option chain
// for a symbol, filtered by DTE, strike range, and option type (calls/puts)
func (c *Client) GetOptionChain(symbol, exchange, right string, maxDTE int, strikeRange float64) ([]ContractInfo, error) {
	// 1. Search for underlying
	conID, months, err := c.SearchUnderlying(symbol, exchange)
	if err != nil {
		return nil, err
	}

	// 2. Get current price
	currentPrice, err := c.GetLastPrice(conID)
	if err != nil {
		return nil, fmt.Errorf("getting current price: %w", err)
	}

	// 3. Filter months by DTE
	validMonths := filterMonthsByDTE(months, maxDTE)
	if len(validMonths) == 0 {
		return nil, fmt.Errorf("no option months within %d days", maxDTE)
	}

	// 4. Collect all contracts across valid months
	var allContracts []ContractInfo

	for _, month := range validMonths {
		// Get strikes for this month
		strikes, err := c.GetStrikes(conID, month, currentPrice, strikeRange)
		if err != nil {
			continue // Skip months with errors
		}

		// Rate limiting
		time.Sleep(200 * time.Millisecond)

		// Get contract info for each strike
		for _, strike := range strikes {
			strikeStr := fmt.Sprintf("%.2f", strike)
			contracts, err := c.GetContractInfo(conID, month, strikeStr, right)
			if err != nil {
				continue // Skip strikes with errors
			}

			allContracts = append(allContracts, contracts...)

			// Rate limiting
			time.Sleep(200 * time.Millisecond)
		}
	}

	return allContracts, nil
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func filterMonthsByDTE(months []string, maxDTE int) []string {
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

	monthStr := strings.ToUpper(month[:3])
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
