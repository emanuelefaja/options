package ibkr

// Quote represents a stock quote from IBKR
type Quote struct {
	Symbol      string
	Price       float64
	Change      float64
	ChangePerc  float64
	Volume      int
	Bid         float64
	Ask         float64
	High        float64
	Low         float64
	Open        float64
	PrevClose   float64
	Description string
	Exchange    string
}

// SearchResult represents a symbol search result
type SearchResult struct {
	ConID       string    `json:"conid"` // IBKR returns this as a string
	Symbol      string    `json:"symbol"`
	Description string    `json:"description"`
	Type        string    `json:"instrumentType"`
	Sections    []Section `json:"sections"`
}

// Section represents a security section (STK, OPT, etc.)
type Section struct {
	SecType string `json:"secType"`
	Months  string `json:"months"`
}

// MarketDataResponse represents raw market data from IBKR
// Note: Field data comes at the root level, not nested in a "fields" object
type MarketDataResponse struct {
	ConID int `json:"conid"`
	// All other fields are market data fields (31, 84, 85, etc.) at root level
	// We'll unmarshal this as a generic map to handle all fields
	Fields map[string]interface{} `json:"-"` // Skip this in JSON unmarshaling
}

// SecDefSearchResponse represents security definition search response for options
type SecDefSearchResponse struct {
	ConID  int      `json:"conid"`
	Symbol string   `json:"symbol"`
	Months []string `json:"months"`
}

// StrikesResponse represents available strikes for an option month
type StrikesResponse struct {
	Call []float64 `json:"call"`
	Put  []float64 `json:"put"`
}

// ContractInfo represents detailed option contract information
type ContractInfo struct {
	ConID           int     `json:"conid"`
	Symbol          string  `json:"symbol"`
	Strike          float64 `json:"strike"`
	Right           string  `json:"right"` // "C" or "P"
	MaturityDate    string  `json:"maturityDate"`
	Multiplier      string  `json:"multiplier"`
	TradingClass    string  `json:"tradingClass"`
	UnderlyingConID int     `json:"underlyingConid"`
}

// OptionPricing represents option pricing data including greeks
type OptionPricing struct {
	Bid             float64
	Ask             float64
	LastPrice       float64
	Delta           float64
	Gamma           float64
	Theta           float64
	Vega            float64
	ImpliedVol      float64
	UnderlyingPrice float64
}
