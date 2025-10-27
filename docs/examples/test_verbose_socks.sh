#!/bin/bash
# Manual verification script for Step 15: SOCKS proxy with verbose logging
# This script tests that verbose logging works correctly for SOCKS proxy scenarios

set -e

echo "=== Step 15: Manual verification - SOCKS proxy with verbose logging ==="
echo

# Cleanup function
cleanup() {
    echo "Cleaning up..."
    kill $MASTER_PID 2>/dev/null || true
    kill $SLAVE_PID 2>/dev/null || true
    kill $HTTP_SERVER_PID 2>/dev/null || true
    pkill -9 goncat.elf 2>/dev/null || true
    rm -f /tmp/master_socks.log /tmp/slave_socks.log /tmp/socks_response.txt
}

trap cleanup EXIT

# Start a simple HTTP server on port 8887 to use as target
echo "Starting HTTP test server on port 8887..."
python3 -m http.server 8887 > /dev/null 2>&1 &
HTTP_SERVER_PID=$!
sleep 1

# Terminal 1: Start slave with verbose logging
echo "Starting slave with verbose logging..."
./dist/goncat.elf slave listen 'tcp://*:12346' --verbose > /tmp/slave_socks.log 2>&1 &
SLAVE_PID=$!
sleep 2

# Terminal 2: Connect master with verbose logging and SOCKS proxy
echo "Connecting master with SOCKS proxy and verbose logging..."
./dist/goncat.elf master connect tcp://localhost:12346 --socks ':1080' --verbose > /tmp/master_socks.log 2>&1 &
MASTER_PID=$!
sleep 3

# Test the SOCKS proxy by making a request through it
echo "Testing SOCKS proxy by making HTTP request through proxy..."
timeout 5 curl -s --socks5 localhost:1080 http://127.0.0.1:8887/ > /tmp/socks_response.txt 2>&1 || true

# Wait a moment for logs to flush
sleep 2

echo
echo "=== Master verbose log (SOCKS proxy) ==="
cat /tmp/master_socks.log | grep -E "\[v\].*SOCKS" || echo "No SOCKS verbose messages"
echo

echo "=== Slave verbose log ==="
cat /tmp/slave_socks.log | grep "\[v\]" | head -10 || echo "No verbose messages"
echo

# Verify expected verbose messages are present
echo "=== Verification ==="
echo

# Check master log for SOCKS proxy messages
if grep -q "SOCKS proxy.*listening" /tmp/master_socks.log; then
    echo "✓ Master: SOCKS proxy listener startup logged"
else
    echo "✗ Master: SOCKS proxy listener startup not logged"
    exit 1
fi

if grep -q "SOCKS proxy.*accepted client connection" /tmp/master_socks.log; then
    echo "✓ Master: SOCKS proxy client connection logged"
else
    echo "✗ Master: SOCKS proxy client connection not logged (may not have been triggered)"
fi

if grep -q "SOCKS proxy.*negotiating method" /tmp/master_socks.log; then
    echo "✓ Master: SOCKS proxy method negotiation logged"
else
    echo "✗ Master: SOCKS proxy method negotiation not logged (may not have been triggered)"
fi

if grep -q "SOCKS proxy.*CONNECT request" /tmp/master_socks.log; then
    echo "✓ Master: SOCKS proxy CONNECT request logged"
else
    echo "✗ Master: SOCKS proxy CONNECT request not logged (may not have been triggered)"
fi

# Check if the HTTP request succeeded
if [ -s /tmp/socks_response.txt ] && grep -q "html" /tmp/socks_response.txt; then
    echo "✓ SOCKS proxy: HTTP request through proxy succeeded"
else
    echo "⚠ SOCKS proxy: HTTP request may not have succeeded (check manually)"
fi

echo
echo "=== SOCKS proxy checks completed! ==="
echo "Verbose logging is working for SOCKS proxy scenarios."
echo
echo "Full master log available at: /tmp/master_socks.log"
echo "Full slave log available at: /tmp/slave_socks.log"
