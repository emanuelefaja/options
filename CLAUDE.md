# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Go-based web application for tracking and analyzing portfolio performance including options trading (covered calls, cash secured puts) and stock positions. Built with standard library HTTP server, HTML templates, and Alpine.js/Tailwind CSS for frontend.

## Commands

### Development
- `go run main.go` - Run the server directly (port 8080)
- `air` - Run with hot reload using Air (.air.toml configured)
- `go build -o mnmlsm` - Build the binary

### Analytics & Tools
- `go run cmd/stats/main.go` - CLI tool to view portfolio stats without running web server
  - Shows: Portfolio Overview, Analytics Metrics, Risk Metrics, Sector Exposure, Position Details
  - Useful for quick portfolio checks and understanding current state

- `go run cmd/ibkr-quote/main.go --symbol TSLA` - Get real-time stock quotes from IBKR
  - Requires IBKR Client Portal Gateway running on https://localhost:5001
  - Supports JSON output: `--format json`
  - Premium scanning: `--premium-scan --right P --min-return 100 --max-dte 4`

- `go run cmd/update-universe/main.go` - Update universe.csv with live prices from IBKR
  - Uses 5 concurrent goroutines for fast parallel updates
  - Updates ~56 stocks in ~12 seconds
  - Requires IBKR Client Portal Gateway running

### IBKR Gateway
- `cd gateway && ./start.sh` - Start IBKR Client Portal Gateway
  - Gateway runs on https://localhost:5001
  - Must authenticate via web browser on first start
  - Required for live market data and options scanning

### Testing & Build
- `go build` - Compile check
- No test framework currently configured

## File Structure

```
.
├── analysis              # Options scanning engine
│   ├── scanner.go       # Premium opportunity scanner
│   └── types.go         # Scanner data structures
├── CLAUDE.md
├── cmd                  # CLI tools
│   ├── ibkr-quote       # Real-time stock quotes
│   │   └── main.go
│   ├── stats            # Portfolio statistics CLI
│   │   └── main.go
│   └── update-universe  # Update universe.csv with live prices
│       └── main.go
├── components           # HTML partials
│   ├── header.html
│   └── sidebar.html
├── data                 # CSV data files (database)
│   ├── options_transactions.csv   # Options trade history
│   ├── stocks_transactions.csv    # Stock buy/sell transactions
│   ├── transactions.csv           # Portfolio deposits/withdrawals
│   ├── universe.csv               # Stock universe with prices & sectors
│   ├── vix.csv                    # VIX historical data
│   └── wise.csv                   # Savings account balance history
├── gateway              # IBKR Client Portal Gateway
│   ├── clientportal.gw  # Gateway installation
│   ├── README.md
│   └── start.sh         # Gateway startup script
├── go.mod
├── ibkr                 # IBKR API client
│   ├── client.go        # HTTP client & base API calls
│   ├── options.go       # Options-specific API calls
│   ├── quotes.go        # Stock quotes API
│   └── types.go         # API data structures
├── layouts              # HTML layout templates
│   └── main.html        # Base template with Alpine.js
├── main.go              # HTTP server & routes
├── mnmlsm               # Compiled binary
├── pages                # HTML page templates
│   ├── analytics.html   # Portfolio analytics dashboard
│   ├── home.html        # Options trades overview
│   ├── options.html     # Options positions table
│   ├── risk.html        # Risk management dashboard
│   ├── rules.html       # Trading rules & strategy
│   └── stocks
│       ├── detail.html  # Individual stock detail page
│       └── index.html   # Stock positions overview
├── plan.md
├── scripts
├── static
└── web                  # Backend logic
    ├── analytics.go            # Portfolio metrics & calculations
    ├── handlers.go             # HTTP route handlers
    ├── options_transactions.go # Options transaction processing
    ├── options.go              # Options display logic (legacy)
    ├── stocks.go               # Stock position tracking (FIFO)
    ├── symbol_analysis.go      # Stock detail page logic
    ├── transactions.go         # Portfolio deposits/withdrawals
    ├── types.go                # Core data structures
    └── weekly_performance.go   # Weekly performance calculations
```

## Architecture

### Core Structure
- **main.go**: HTTP server setup and route handlers
  - `/` - Home page with options trades
  - `/stocks` - Stock positions (open and closed)
  - `/stocks/:symbol` - Individual stock detail page
  - `/analytics` - Portfolio analytics dashboard
  - `/risk` - Risk management dashboard
  - `/rules` - Trading rules & strategy reference

### Data Layer (`web/` package)
- **handlers.go**: HTTP route handlers for all pages
- **types.go**: Core data structures (PageData, shared types)
- **options.go**: Options trading logic (legacy, Trade struct)
- **options_transactions.go**: Options transaction processing
  - Groups transactions into positions
  - Calculates net premiums, returns, and Greeks
- **stocks.go**: Stock position management with FIFO lot tracking
  - Handles both open and closed positions
  - Calculates realized & unrealized P&L
- **analytics.go**: Portfolio metrics and calculations
  - Time-weighted return (TWR)
  - Sector exposure analysis
  - Position risk calculations
  - Daily returns aggregation
- **weekly_performance.go**: Weekly return tracking
- **symbol_analysis.go**: Individual stock analysis page logic
- **transactions.go**: Portfolio deposits/withdrawals
- **Formatting utilities**: Currency and percentage formatting

### IBKR Integration (`ibkr/` package)
- **client.go**: Base HTTP client with TLS configuration
  - Handles authentication and API communication
  - Market data snapshot requests
- **quotes.go**: Stock quote fetching
  - GetQuote() - Single stock quote
  - GetQuotes() - Batch quote fetching
- **options.go**: Options chain and pricing
  - SearchUnderlying() - Find option contract IDs
  - GetStrikes() - Available strike prices
  - GetContractInfo() - Option contract details
  - GetOptionPricing() - Pricing & Greeks
- **types.go**: API data structures

### Options Scanner (`analysis/` package)
- **scanner.go**: Premium opportunity scanner
  - Scans options chains for high-yield trades
  - Filters by annualized return, DTE, strike range
  - Returns sorted list of qualified contracts
- **types.go**: Scanner result structures

### Data Files (`data/`)
- **options_transactions.csv** - Options trade history (sell/buy transactions)
- **stocks_transactions.csv** - Stock buy/sell transactions for FIFO tracking
- **transactions.csv** - Portfolio deposits/withdrawals
- **universe.csv** - Stock universe with live prices & sector mappings (Ticker,Name,Price,Sector)
- **vix.csv** - VIX historical data for volatility tracking
- **wise.csv** - Monthly savings account balance history

### Templates
- **layouts/main.html** - Base template with Alpine.js, Tailwind CSS, Chart.js
- **components/** - Reusable partials (sidebar, header)
- **pages/** - Page-specific templates
  - home.html - Options trades overview
  - options.html - Options positions table
  - stocks/index.html - Stock positions
  - stocks/detail.html - Individual stock analysis
  - analytics.html - Portfolio analytics dashboard
  - risk.html - Risk management dashboard
  - rules.html - Trading rules reference

## Key Implementation Details

### Trading System
- **FIFO lot tracking**: Stock positions track cost basis using first-in-first-out
- **Options position grouping**: Transactions grouped into positions (open/expired/rolled)
- **Net premium calculation**: Accounts for collected premiums, buybacks, and commissions
- **Sector diversification**: universe.csv maps stocks to sectors for risk tracking

### IBKR Integration
- **Client Portal Gateway**: REST API proxy to Interactive Brokers
- **Real-time quotes**: Live market data via GetQuote()
- **Options scanning**: Search chains for premium opportunities
- **Concurrent updates**: 5 goroutines update universe.csv in parallel (~12s for 56 stocks)
- **Field mappings**: API returns fields at root level (not nested)
  - Field 31 = Last Price
  - Field 84 = Bid
  - Field 86 = Ask (NOT 85!)
  - Field 87_raw = Volume

### Data Architecture
- **CSV as database**: No external DB, all data in CSV files
- **Transaction-based**: All trades stored as transactions, positions calculated on load
- **Immutable history**: CSVs are append-only, never edit historical data
- **Daily snapshots**: wise.csv tracks monthly net worth snapshots

### Frontend
- **Alpine.js**: Reactive UI components and state management
- **Chart.js**: Portfolio performance charts and visualizations
- **Tailwind CSS**: Utility-first styling
- **Server-side rendering**: Go templates with JSON data injection
- **Real-time status indicators**: Animated ping dots for risk compliance

### Performance
- All monetary values stored as floats, formatted for display
- Template functions handle conditional styling (positive/negative values)
- Portfolio totals calculated on page load (no caching)
- Sector exposure and risk metrics computed from transaction history


# Trade Strategy: Premium Harvesting via Covered Calls

- Sell 4-day to 0-day OTM or slightly ATM calls with >100% annualized returns on stocks I would happily own, then let math and time work for me. 
- Aiming for 52% capital growth per year overall, so 1% per week.
- This is a positive expectancy rate strategy. 
- No problem with getting assigned, we will just buy back the stock and sell again.
- I am Panama permanent foreign resident, so I pay 0% tax on foreign sourced income including capital gains and option premiums. US does NOT tax me on this.

## Current Portfolio Statistics (as of Day 49)

**Portfolio Overview:**
- Total Portfolio Value: $113,328
- Total Deposits: $96,009
- Total Profit: $17,320
- Portfolio Return: **18.04%**
- Time-Weighted Return: 18.84% (Ann: **245.73%**)
- Days Active: 49

**Capital Deployment:**
- At Risk Capital: $92,068 (58.3% of total capital)
- Available Cash (Dry Powder): $21,261
- Wise Balance: $44,629

**Profit Breakdown:**
- Total Premiums Collected: $9,581
- Total Stock Profit: $7,739
- Unrealized P&L: -$2,119 (-2.2% of at risk capital) ✓ Risk Compliant

**Performance Metrics:**
- Premium Per Day: $275
- Daily Theta: $321
- Avg Return Per Option: 1.84%
- Weekly Return Rate: 11.91% (+$10,961) ✓ On Track

**Trade Statistics:**
- Number of Option Trades: 64
- Number of Stock Trades: 29
- Total Number of Trades: 93
- Largest Premium: $1,821
- Smallest Premium: $12
- Average Premium: $150

**Risk Metrics:**
- VIX: 21.66
- Largest Position: SHOP Put at $15,000 (9.5%) ✓ Compliant
- Top 3 Positions: SHOP, HOOD, HIMS (all <10%) ✓ Compliant

**Sector Exposure:**
- Technology: **20.5%** (improved diversification)
- Financial Services: 16.2%
- Healthcare: 12.3%
- Industrials: 3.9%
- Energy: 3.0%
- Consumer Cyclical: 2.3%

**Strategy Status:**
- Testing period complete (50+ trades)
- Now scaling and optimizing strategy
- TWR annualized at 246% - exceeding 52% annual target
- Tech sector concentration improved to 20.5% (well under 40% limit)
- All positions now compliant with 10% single-position limit

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