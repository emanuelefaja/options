package analysis

// ScanParams defines parameters for premium scanning
type ScanParams struct {
	Symbol      string  // Stock symbol to scan
	Exchange    string  // Exchange (e.g., "NASDAQ", "NYSE")
	Right       string  // "C" for calls, "P" for puts
	StrikeRange float64 // Strike price range around current price
	MinReturn   float64 // Minimum annualized return percentage (e.g., 100 for 100%)
	MaxDTE      int     // Maximum days to expiration
}

// OptionContract represents an option contract with calculated metrics
type OptionContract struct {
	// Contract details
	Symbol          string
	Strike          float64
	Right           string // "C" or "P"
	MaturityDate    string
	ConID           int
	UnderlyingConID int

	// Pricing
	Bid             float64
	Ask             float64
	MidPrice        float64
	UnderlyingPrice float64

	// Greeks
	Delta      float64
	Gamma      float64
	Theta      float64
	Vega       float64
	ImpliedVol float64

	// Calculated metrics
	DTE              int     // Days to expiration
	Premium          float64 // Dollar premium (total)
	IntrinsicValue   float64 // Intrinsic value (ITM amount)
	ExtrinsicValue   float64 // Extrinsic value (time premium)
	PremiumPercent   float64 // Premium as % of strike (based on extrinsic)
	AnnualizedReturn float64 // Annualized return % (based on extrinsic)
	CapitalRequired  float64 // Capital required for cash-secured put/covered call
	POP              float64 // Probability of Profit (1 - |Delta|) as percentage
	Efficiency       float64 // Risk-adjusted return: AnnualizedReturn / (1 - POP)
	IsITM            bool    // Whether option is in-the-money
}
