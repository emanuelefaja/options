# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Go-based web application for tracking and analyzing portfolio performance including options trading (covered calls, cash secured puts) and stock positions. Built with standard library HTTP server, HTML templates, and Alpine.js/Tailwind CSS for frontend.

## Commands

### Development
- `go run main.go` - Run the server directly (port 8080)
- `air` - Run with hot reload using Air (.air.toml configured)
- `go build -o mnmlsm` - Build the binary

### Testing & Build
- `go build` - Compile check
- No test framework currently configured

## Architecture

### Core Structure
- **main.go**: HTTP server setup and route handlers
  - `/` - Home page with options trades
  - `/stocks` - Stock positions (open and closed)
  - `/analytics` - Portfolio analytics and metrics

### Data Layer (`web/` package)
- **types.go**: Core data structures (PageData, shared types)
- **options.go**: Options trading logic (Trade struct, CSV loading)
- **stocks.go**: Stock position management with FIFO lot tracking
  - Handles both open and closed positions
  - Calculates P&L based on stock_transactions.csv
- **analytics.go**: Portfolio metrics calculations
- **transactions.go**: Portfolio funding transactions
- **Formatting utilities**: Currency and percentage formatting

### Data Files (`data/`)
- `options.csv` - Options trades history
- `stocks.csv` - Stock positions (legacy)
- `stocks_transactions.csv` - Buy/sell transactions for lot tracking
- `transactions.csv` - Portfolio deposits/withdrawals

### Templates
- **layouts/main.html** - Base template with Alpine.js setup
- **components/** - Reusable partials (sidebar, header)
- **pages/** - Page-specific templates (index, stocks, analytics)

## Key Implementation Details

- Stock positions use FIFO lot tracking for accurate P&L
- All monetary values stored as floats, formatted for display
- CSV files serve as the database (no external DB dependencies)
- Template functions handle conditional styling (positive/negative values)
- Portfolio totals calculated across options premiums and stock positions


# Trade Strategy: Premium Harvesting via Covered Calls

- Sell 4-day to 0-day OTM or slightly ATM calls with >100% annualized returns on stocks I would happily own, then let math and time work for me. 
- Aiming for 52% capital growth per year overall, so 1% per week.
- This is a positive expectancy rate strategy. 
- No problem with getting assigned, we will just buy back the stock and sell again.
- I am Panama permanent foreign resident, so I pay 0% tax on foreign sourced income including capital gains and option premiums. US does NOT tax me on this.

## Current Portfolio Statistics (as of Day 45)

**Portfolio Overview:**
- Total Portfolio Value: $110,387
- Total Deposits: $96,009
- Total Profit: $14,379
- Portfolio Return: **14.98%**
- Days Active: 45
- IBKR Value: $108,362

**Capital Deployment:**
- Total Active Capital: $64,218
- Cumulative Capital Deployed: $434,608

**Profit Breakdown:**
- Total Premiums Collected: $6,640
- Total Stock Profit: $7,739
- Unrealized P&L: -$2,026

**Performance Metrics:**
- Premium Per Day: $217
- Daily Theta: $154
- Avg Return Per Option: 1.71%

**Trade Statistics:**
- Number of Option Trades: 50
- Number of Stock Trades: 28
- Total Number of Trades: 78
- Largest Premium: $459
- Smallest Premium: $12
- Average Premium: $133

**Testing Status:**
- Currently at 50 option trades
- Target: 50 trades to refine strategy (testing period complete - now evaluating results)

## Trade Decision Framework

```
| **Criteria** | **Requirement** | **Status** |
|-------------|----------------|------------|
| Annualized Return | > 100% | ✅/❌ |
| Events Check | No earnings/dividend in period | ✅/❌ |
| Liquidity | Tight bid/ask spread | ✅/❌ |
| Position Size | < 10% of portfolio | ✅/❌ |
```

**IMPORTANT: Always create this table when evaluating any trade**

**Decision:**
- ✅ **ALL GREEN = EXECUTE**
- ❌ **ANY RED = SKIP**


**Quick Math:**
- Annual Return = (Premium % / Days) × 365
- Example: 1.2% in 2 days = (1.2/2) × 365 = 219% annualized ✅

**Stock Priority (when multiple pass):**
1. Highest annualized return
2. Better liquidity (tighter spread)
3. Lower correlation to existing positions

**Event Check Requirements:**
- **Check earnings date** - Avoid if earnings within 5 days
- **Check ex-dividend date** - Avoid if ex-div during holding period
- **Check company events** - Product launches, FDA approvals, court cases
- **Check Fed meetings** - FOMC can move entire market
- **Check sector events** - OPEC for oil stocks, CPI for rate-sensitive


## Risk Management

- No margin
- Only sell covered calls or cash secured puts
- No buying back calls to "defend" positions
- All positions should be small enough that I do not lose sleep over them.
- No single position is more than 10% of portfolio. So a 20% drop in a stock is a 2% drop in the portfolio.
- Avoid crazy meme stock (i.e. GME, AMC, BBBY) where stock price has nothing to do with fundamentals.

## Stock Categories & Selection

**Tier 1: Juice Machines (300-400% annualized)**
- Core picks: SOFI, MARA, RIOT, CVNA, HOOD, DKNG
- Alternatives: COIN, UPST, AFRM, RBLX, BYND, OPEN
- *Allocate 2 positions max, avoid correlation*

**Tier 2: Moderate Volatility (150-250% annualized)**
- Core picks: TSLA, AMD, NVDA
- Alternatives: META, NFLX, SQ, PYPL, ROKU, NIO, ARKK, , UBER, MRNA, SNAP
- *Allocate 2 positions - tech/growth but less wild*

**Tier 3: Boring Anchors (100-150% annualized)**
- Core picks: XOM, OXY, BAC, TGT, F, AA
- Alternatives: WFC, CVX, WMT, KO, PFE, T, VZ, GE
- *Allocate 1-2 positions for stability*

**Correlation Groups to AVOID Stacking:**
- Crypto proxies: MARA, RIOT, COIN, CLSK
- Fintech: SOFI, UPST, AFRM, SQ, HOOD
- EV plays: TSLA, NIO, RIVN, LCID
- Meme stocks: GME, AMC, BBBY
- AI plays: AI, BBAI, C3.AI
- High-growth tech: AMD, NVDA, SMCI, MU
- **Risk-on/Risk-off warning:** SOFI, AMD, TSLA, COIN all crash together when market fears rise

**Optimal 5-Position Spread Example:**
1. One growth/tech (SOFI, or AMD - pick ONE)
2. One financials/banks (BAC, WFC, or JPM)
3. One energy/commodities (XOM, OXY, or CVX)
4. One consumer/defensive (TGT, WMT, or KO)
5. One wildcard/other sector (F, DKNG, or healthcare)

**Maximum 40% exposure to growth/tech stocks** - they all crash together

## Strategy when underwater (when cost basis is higher than stock price)
  - When holding stock below cost basis, accept >50% annualized returns (vs >100% for normal trades)
  - Extend duration to 1-4 weeks to capture more premium while waiting for recovery
  - Set strike at or near cost basis to avoid locking in losses (unless you want to exit)
  - This is defensive income generation, not aggressive premium harvesting - don't confuse the two strategies