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