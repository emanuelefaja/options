package web

type PageData struct {
	Title              string
	CurrentPage        string
	Trades             []Trade
	OptionPositions    []OptionPosition
	Stocks             []Stock
	ClosedStocks       []Stock
	TotalPremiums      float64
	TotalCapital       float64
	TotalActiveCapital float64
	PremiumPerDay      float64
	AvgReturnPerTrade  float64
	LargestPremium     float64
	SmallestPremium    float64
	AveragePremium     float64
	OptionTradesCount  int
	StockTradesCount   int
	TotalTradesCount   int
	// Options page specific
	OpenOptionsCount     int
	ClosedOptionsCount   int
	OptionsActiveCapital float64
	CollectedPremiums    float64
	OptionsActiveCapitalFormatted string
	CollectedPremiumsFormatted    string
	TotalPremiumsFormatted      string
	TotalCapitalFormatted       string
	TotalActiveCapitalFormatted string
	PremiumPerDayFormatted      string
	AvgReturnPerTradeFormatted  string
	LargestPremiumFormatted     string
	SmallestPremiumFormatted    string
	AveragePremiumFormatted     string
	// Portfolio fields
	TotalPortfolioValue          float64
	TotalPortfolioProfit         float64
	TotalPortfolioProfitPercentage float64
	TotalStockProfit             float64
	TotalPortfolioValueFormatted  string
	TotalPortfolioProfitFormatted string
	TotalPortfolioProfitPercentageFormatted string
	TotalStockProfitFormatted    string
	TotalDepositsFormatted       string
	// Daily returns data
	DailyReturns     []DailyReturn
	DailyReturnsJSON string
	// Symbol-specific data
	Symbol          string            // Current symbol for detail pages
	SymbolSummaries []SymbolSummary   // For stocks index table
	SymbolDetails   SymbolDetails     // For individual stock detail page
	SymbolStocks    []Stock           // Filtered stocks for this symbol
	SymbolOptions   []OptionPosition  // Filtered options for this symbol
	// Stock performance data
	StockPerformance StockPerformance
	// Net worth data
	NetWorthData     []NetWorthMonth
	NetWorthDataJSON string
}

type NetWorthMonth struct {
	Month            string  `json:"month"`
	SavingsBalance   float64 `json:"savingsBalance"`
	BrokerageBalance float64 `json:"brokerageBalance"`
	TotalNetWorth    float64 `json:"totalNetWorth"`
}

type SymbolSummary struct {
	Symbol            string
	TotalPL           float64 // Premium + Stock P/L
	PremiumsCollected float64 // Sum of option net premiums
	StockPL           float64 // Realized stock P/L
	TotalCapital      float64 // Peak or current capital deployed
	TotalPLFormatted  string
	PremiumsFormatted string
	StockPLFormatted  string
	CapitalFormatted  string
}

type SymbolDetails struct {
	Symbol                         string
	TotalPremiumCollected          float64
	TotalStockPL                   float64
	NumberOfOptionsTrades          int
	CurrentCapital                 float64
	AverageDTE                     float64
	AvgOptionReturn                float64
	TotalPL                        float64
	PercentOfOverallPL             float64
	TotalPremiumCollectedFormatted string
	TotalStockPLFormatted          string
	CurrentCapitalFormatted        string
	TotalPLFormatted               string
	PercentOfOverallPLFormatted    string
	AverageDTEFormatted            string
	AvgOptionReturnFormatted       string
	NumberOfOptionsTradesFormatted string
}