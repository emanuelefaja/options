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
}