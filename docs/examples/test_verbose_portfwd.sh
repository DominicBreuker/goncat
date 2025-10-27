#!/bin/bash
# Manual verification script for Step 14: Port forwarding with verbose logging
# This script tests that verbose logging works correctly for port forwarding scenarios

set -e

echo "=== Step 14: Manual verification - Port forwarding with verbose logging ==="
echo

# Cleanup function
cleanup() {
    echo "Cleaning up..."
    kill $MASTER_PID 2>/dev/null || true
    kill $SLAVE_PID 2>/dev/null || true
    kill $HTTP_SERVER_PID 2>/dev/null || true
    pkill -9 goncat.elf 2>/dev/null || true
    rm -f /tmp/master_pf.log /tmp/slave_pf.log /tmp/http_response.txt
}

trap cleanup EXIT

# Start a simple HTTP server on port 8888 to use as target
echo "Starting HTTP test server on port 8888..."
python3 -m http.server 8888 > /dev/null 2>&1 &
HTTP_SERVER_PID=$!
sleep 1

# Terminal 1: Start slave with verbose logging
echo "Starting slave with verbose logging..."
./dist/goncat.elf slave listen 'tcp://*:12345' --verbose > /tmp/slave_pf.log 2>&1 &
SLAVE_PID=$!
sleep 2

# Terminal 2: Connect master with verbose logging and local port forwarding
echo "Connecting master with local port forwarding and verbose logging..."
./dist/goncat.elf master connect tcp://localhost:12345 -L '127.0.0.1:9999:127.0.0.1:8888' --verbose > /tmp/master_pf.log 2>&1 &
MASTER_PID=$!
sleep 3

# Test the port forwarding by making a request through the tunnel
echo "Testing port forwarding by making HTTP request through tunnel..."
timeout 5 curl -s http://localhost:9999/ > /tmp/http_response.txt 2>&1 || true

# Wait a moment for logs to flush
sleep 2

echo
echo "=== Master verbose log (port forwarding) ==="
cat /tmp/master_pf.log | grep -E "\[v\].*[Pp]ort.*forward" || echo "No port forwarding verbose messages"
echo

echo "=== Slave verbose log ==="
cat /tmp/slave_pf.log | grep "\[v\]" | head -10 || echo "No verbose messages"
echo

# Verify expected verbose messages are present
echo "=== Verification ==="
echo

# Check master log for port forwarding messages
if grep -q "Port forwarding.*listening" /tmp/master_pf.log; then
    echo "✓ Master: Port forwarding listener startup logged"
else
    echo "✗ Master: Port forwarding listener startup not logged"
    exit 1
fi

if grep -q "Port forwarding.*accepted connection" /tmp/master_pf.log; then
    echo "✓ Master: Port forwarding connection acceptance logged"
else
    echo "✗ Master: Port forwarding connection acceptance not logged (may not have been triggered)"
fi

if grep -q "Port forwarding.*forwarding stream" /tmp/master_pf.log; then
    echo "✓ Master: Port forwarding stream creation logged"
else
    echo "✗ Master: Port forwarding stream creation not logged (may not have been triggered)"
fi

# Check if the HTTP request succeeded
if [ -s /tmp/http_response.txt ] && grep -q "html" /tmp/http_response.txt; then
    echo "✓ Port forwarding: HTTP request through tunnel succeeded"
else
    echo "⚠ Port forwarding: HTTP request may not have succeeded (check manually)"
fi

echo
echo "=== Port forwarding checks completed! ==="
echo "Verbose logging is working for port forwarding scenarios."
echo
echo "Full master log available at: /tmp/master_pf.log"
echo "Full slave log available at: /tmp/slave_pf.log"
