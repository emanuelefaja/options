package ibkr

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Client represents an IBKR API client
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new IBKR API client
// It assumes the Client Portal Gateway is running on localhost:5001
func NewClient() *Client {
	// Create HTTP client that skips TLS verification for localhost
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return &Client{
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
		baseURL: "https://localhost:5001/v1/api",
	}
}

// SearchSymbol searches for a symbol and returns its ConID
func (c *Client) SearchSymbol(symbol string) (int, error) {
	url := fmt.Sprintf("%s/iserver/secdef/search?symbol=%s", c.baseURL, symbol)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return 0, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("reading response: %w", err)
	}

	var results []SearchResult
	if err := json.Unmarshal(body, &results); err != nil {
		return 0, fmt.Errorf("parsing response: %w", err)
	}

	if len(results) == 0 {
		return 0, fmt.Errorf("symbol not found: %s", symbol)
	}

	// Parse ConID from string to int
	var conid int
	if _, err := fmt.Sscanf(results[0].ConID, "%d", &conid); err != nil {
		return 0, fmt.Errorf("parsing ConID: %w", err)
	}

	return conid, nil
}

// GetMarketData fetches market data for given ConIDs
func (c *Client) GetMarketData(conids []int) ([]MarketDataResponse, error) {
	conidStrs := make([]string, len(conids))
	for i, conid := range conids {
		conidStrs[i] = strconv.Itoa(conid)
	}
	conidParam := strings.Join(conidStrs, ",")

	fields := "31,84,85,86,87,88,7295,7296,7741,7762,7764,7768"
	url := fmt.Sprintf("%s/iserver/marketdata/snapshot?conids=%s&fields=%s",
		c.baseURL, conidParam, fields)

	// Preflight request to initialize market data stream
	c.httpClient.Get(url)
	time.Sleep(500 * time.Millisecond)

	// Actual request
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching market data: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// First unmarshal into a slice of maps since fields are at root level
	var rawData []map[string]interface{}
	if err := json.Unmarshal(body, &rawData); err != nil {
		return nil, fmt.Errorf("parsing market data: %w", err)
	}

	// Convert to MarketDataResponse structs
	data := make([]MarketDataResponse, 0, len(rawData))
	for _, raw := range rawData {
		response := MarketDataResponse{
			Fields: make(map[string]interface{}),
		}

		// Extract conid and all other fields
		for key, value := range raw {
			if key == "conid" {
				response.ConID = parseInt(value)
			} else {
				// Store all other fields (market data) in Fields map
				response.Fields[key] = value
			}
		}

		data = append(data, response)
	}

	return data, nil
}

// parseFloat safely parses a float from an interface{}
func parseFloat(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}

// parseInt safely parses an int from an interface{}
func parseInt(val interface{}) int {
	switch v := val.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		i, _ := strconv.Atoi(v)
		return i
	default:
		return 0
	}
}
