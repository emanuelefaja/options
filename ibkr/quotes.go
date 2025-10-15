package ibkr

import (
	"fmt"
	"time"
)

// GetQuote fetches a single stock quote
func (c *Client) GetQuote(symbol string) (*Quote, error) {
	// Search for symbol
	conid, err := c.SearchSymbol(symbol)
	if err != nil {
		return nil, fmt.Errorf("searching symbol: %w", err)
	}

	// Get market data
	data, err := c.GetMarketData([]int{conid})
	if err != nil {
		return nil, fmt.Errorf("getting market data: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no market data returned for %s", symbol)
	}

	return parseQuote(symbol, data[0])
}

// GetQuotes fetches multiple stock quotes
func (c *Client) GetQuotes(symbols []string) ([]*Quote, error) {
	// Search for all symbols first
	conids := make([]int, 0, len(symbols))
	symbolMap := make(map[int]string) // conid -> symbol

	for _, symbol := range symbols {
		conid, err := c.SearchSymbol(symbol)
		if err != nil {
			return nil, fmt.Errorf("searching symbol %s: %w", symbol, err)
		}
		conids = append(conids, conid)
		symbolMap[conid] = symbol

		// Rate limiting - IBKR has strict limits
		time.Sleep(100 * time.Millisecond)
	}

	// Get market data for all conids
	data, err := c.GetMarketData(conids)
	if err != nil {
		return nil, fmt.Errorf("getting market data: %w", err)
	}

	// Parse quotes
	quotes := make([]*Quote, 0, len(data))
	for _, d := range data {
		symbol := symbolMap[d.ConID]
		quote, err := parseQuote(symbol, d)
		if err != nil {
			return nil, fmt.Errorf("parsing quote for %s: %w", symbol, err)
		}
		quotes = append(quotes, quote)
	}

	return quotes, nil
}

// parseQuote converts MarketDataResponse to Quote
func parseQuote(symbol string, data MarketDataResponse) (*Quote, error) {
	fields := data.Fields

	quote := &Quote{
		Symbol: symbol,
	}

	// Field mappings from IBKR API (verified from actual responses):
	// 31 = Last Price
	// 84 = Bid
	// 85 = Ask Size (not Ask!)
	// 86 = Ask (not Ask Size!)
	// 88 = Bid Size
	// 87 = Volume (formatted string like "65.7M")
	// 87_raw = Volume (raw number)
	// 7295 = Previous Close
	// 7296 = Change
	// 7762 = Total Volume

	if val, ok := fields["31"]; ok {
		quote.Price = parseFloat(val)
	}
	if val, ok := fields["84"]; ok {
		quote.Bid = parseFloat(val)
	}
	if val, ok := fields["86"]; ok {
		// Field 86 is Ask, not 85!
		quote.Ask = parseFloat(val)
	}
	// Try 87_raw first (raw number), fall back to 7762 (total volume)
	if val, ok := fields["87_raw"]; ok {
		quote.Volume = parseInt(val)
	} else if val, ok := fields["7762"]; ok {
		quote.Volume = parseInt(val)
	}
	if val, ok := fields["7295"]; ok {
		quote.PrevClose = parseFloat(val)
	}
	if val, ok := fields["7296"]; ok {
		quote.Change = parseFloat(val)
	}

	// Calculate change/changePerc if not provided
	if quote.Change == 0 && quote.PrevClose > 0 {
		quote.Change = quote.Price - quote.PrevClose
	}
	if quote.ChangePerc == 0 && quote.PrevClose > 0 {
		quote.ChangePerc = (quote.Change / quote.PrevClose) * 100
	}

	return quote, nil
}
