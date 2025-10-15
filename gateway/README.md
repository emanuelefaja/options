# IBKR Client Portal Gateway

## What is this?
The IBKR Client Portal Gateway is a local API proxy that allows programmatic access to your Interactive Brokers account. It must be running for the `ibkr` package to work.

## Requirements
- Java Runtime Environment (JRE) 8 or higher
- Active Interactive Brokers account

## Installation

**Note**: The gateway binary is NOT included in this repository. Download it separately:

1. Download from IBKR: https://www.interactivebrokers.com/en/trading/ib-api.php
2. Look for "Client Portal API Gateway"
3. Extract the ZIP file contents into `gateway/clientportal.gw/`
4. Verify structure: You should have `gateway/clientportal.gw/bin/run.sh`

## How to Start

```bash
cd gateway
./start.sh
```

Or manually:
```bash
cd gateway/clientportal.gw
./bin/run.sh root/conf.yaml
```

## Accessing the Gateway

Once started:
- **Web UI**: https://localhost:5001
- **API Base URL**: https://localhost:5001/v1/api

## First Time Setup

1. Start the gateway (see above)
2. Open https://localhost:5001 in your browser
3. Accept the self-signed certificate warning
4. Login with your IBKR credentials
5. Complete 2FA if required
6. You should see the API portal dashboard

## API Authentication

The gateway maintains an authenticated session after you login via the web UI. Your Go code will use this session to make API calls.

**Important**: The session expires after ~24 hours of inactivity. You'll need to re-login via the web UI.

## Checking Authentication Status

```bash
curl -sk https://localhost:5001/v1/api/iserver/auth/status
```

Should return: `{"authenticated": true, ...}`

## Common Issues

### Gateway won't start
- Check Java is installed: `java -version`
- Check port 5001 is available: `lsof -i :5001`

### Can't access web UI
- Accept the self-signed certificate in your browser
- Try different browsers (Chrome/Firefox/Safari)

### Authentication keeps expiring
- This is normal IBKR behavior (security)
- You'll need to re-login periodically
- Consider keeping the web UI tab open

## Stopping the Gateway

Press `Ctrl+C` in the terminal where it's running.

## Logs

Logs are stored in: `clientportal.gw/root/logs/`
