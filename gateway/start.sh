#!/bin/bash

# Start IBKR Client Portal Gateway
# This must be running for the IBKR API to work

echo "=== Starting IBKR Client Portal Gateway ==="
echo ""
echo "The gateway will start on https://localhost:5001"
echo "You'll need to login via the web UI at: https://localhost:5001"
echo ""
echo "Press Ctrl+C to stop the gateway"
echo ""

cd "$(dirname "$0")/clientportal.gw"
./bin/run.sh root/conf.yaml
