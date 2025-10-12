package web

import (
	"fmt"
	"time"
)

type WeeklyPerformance struct {
	WeeklyReturnPercent   float64
	WeeklyReturnFormatted string
	WeeklyPL              float64
	WeeklyPLFormatted     string
	DaysRemainingInWeek   int
	WeeklyReturnStatus    string // "compliant", "warning", "violation"
	WeekStartDate         string
	TargetWeeklyReturn    float64
}

// CalculateWeeklyPerformance calculates the weekly P&L and return metrics
func CalculateWeeklyPerformance(portfolioValue float64) WeeklyPerformance {
	now := time.Now()

	// Calculate current week boundaries (Monday to Sunday)
	weekStart := getWeekStart(now)
	weekEnd := getWeekEnd(weekStart)

	// Calculate days remaining in week
	daysRemaining := int(weekEnd.Sub(now).Hours() / 24)
	if daysRemaining < 0 {
		daysRemaining = 0
	}

	// Load and calculate weekly P&L from closed trades
	weeklyPL := calculateWeeklyPL(weekStart, weekEnd)

	// Calculate weekly return percentage
	weeklyReturnPercent := 0.0
	if portfolioValue > 0 {
		weeklyReturnPercent = (weeklyPL / portfolioValue) * 100
	}

	// Determine status based on thresholds
	status := "compliant"
	if weeklyReturnPercent < 0.5 {
		status = "violation"
	} else if weeklyReturnPercent < 1.0 {
		status = "warning"
	}

	return WeeklyPerformance{
		WeeklyReturnPercent:   weeklyReturnPercent,
		WeeklyReturnFormatted: fmt.Sprintf("%.2f", weeklyReturnPercent),
		WeeklyPL:              weeklyPL,
		WeeklyPLFormatted:     FormatCurrency(weeklyPL),
		DaysRemainingInWeek:   daysRemaining,
		WeeklyReturnStatus:    status,
		WeekStartDate:         weekStart.Format("2006-01-02"),
		TargetWeeklyReturn:    1.0,
	}
}

// getWeekStart returns the most recent Monday at 00:00
func getWeekStart(t time.Time) time.Time {
	// Get the weekday (0 = Sunday, 1 = Monday, etc.)
	weekday := int(t.Weekday())

	// Calculate days since last Monday
	daysSinceMonday := (weekday + 6) % 7

	// Subtract days to get to Monday
	weekStart := t.AddDate(0, 0, -daysSinceMonday)

	// Set to start of day (00:00:00)
	return time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())
}

// getWeekEnd returns the Sunday at 23:59:59 for the given week start
func getWeekEnd(weekStart time.Time) time.Time {
	weekEnd := weekStart.AddDate(0, 0, 6)
	return time.Date(weekEnd.Year(), weekEnd.Month(), weekEnd.Day(), 23, 59, 59, 0, weekEnd.Location())
}

// calculateWeeklyPL sums up P&L from all trades within the current week
// Uses the same calculation as analytics page for consistency
// For options: premiums collected from positions OPENED this week (sell to open)
// For stocks: realized P&L from positions CLOSED this week
func calculateWeeklyPL(weekStart, weekEnd time.Time) float64 {
	// Use the same daily returns calculation as analytics for consistency
	optionTransactions := LoadOptionTransactions("data/options_transactions.csv")
	stockTransactions := LoadStockTransactions("data/stocks_transactions.csv")
	optionPositions := CalculateOptionPositions(optionTransactions)
	dailyReturns := CalculateDailyReturnsNew(optionPositions, stockTransactions)

	// Sum up all returns that fall within the current week
	weeklyPL := 0.0
	for _, dr := range dailyReturns {
		date, err := time.Parse("2006-01-02", dr.Date)
		if err != nil {
			continue
		}

		// Check if date is within current week
		if (date.Equal(weekStart) || date.After(weekStart)) &&
		   (date.Before(weekEnd) || date.Equal(weekEnd)) {
			weeklyPL += dr.TotalReturns
		}
	}

	return weeklyPL
}
