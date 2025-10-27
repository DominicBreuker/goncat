#!/bin/bash
# Manual verification script for Step 16: TLS/SSL with verbose logging
# This script tests that verbose logging works correctly for TLS/SSL scenarios

set -e

echo "=== Step 16: Manual verification - TLS/SSL with verbose logging ==="
echo

# Cleanup function
cleanup() {
    echo "Cleaning up..."
    kill $MASTER_PID 2>/dev/null || true
    pkill -9 goncat.elf 2>/dev/null || true
    rm -f /tmp/master_tls.log /tmp/slave_tls.log
}

trap cleanup EXIT

# Terminal 1: Start master with TLS and verbose logging
echo "Starting master with TLS/SSL and verbose logging..."
./dist/goncat.elf master listen 'tcp://*:12348' --exec /bin/sh --ssl --key 'test-password' --verbose > /tmp/master_tls.log 2>&1 &
MASTER_PID=$!
sleep 2

# Terminal 2: Connect slave with TLS and verbose logging
echo "Connecting slave with TLS/SSL and verbose logging..."
echo "echo 'TLS_TEST' && exit" | timeout 5 ./dist/goncat.elf slave connect tcp://localhost:12348 --ssl --key 'test-password' --verbose > /tmp/slave_tls.log 2>&1 || true

# Wait a moment for logs to flush
sleep 1

echo
echo "=== Master verbose log (TLS/SSL) ==="
cat /tmp/master_tls.log | grep -E "\[v\].*[Tt][Ll][Ss]|[Cc]ertificate" || echo "No TLS verbose messages"
echo

echo "=== Slave verbose log (TLS/SSL) ==="
cat /tmp/slave_tls.log | grep -E "\[v\].*[Tt][Ll][Ss]|[Cc]ertificate" || echo "No TLS verbose messages"
echo

# Verify expected verbose messages are present
echo "=== Verification ==="
echo

# Check master log for TLS messages
if grep -q "Generating TLS certificates" /tmp/master_tls.log; then
    echo "✓ Master: TLS certificate generation logged"
else
    echo "✗ Master: TLS certificate generation not logged"
    exit 1
fi

if grep -q "TLS.*handshake.*with" /tmp/master_tls.log || grep -q "Starting TLS handshake" /tmp/master_tls.log; then
    echo "✓ Master: TLS handshake logged"
else
    echo "✗ Master: TLS handshake not logged"
    exit 1
fi

if grep -q "TLS mutual authentication enabled" /tmp/master_tls.log; then
    echo "✓ Master: TLS mutual authentication logged"
else
    echo "⚠ Master: TLS mutual authentication not logged (may be optional)"
fi

# Check slave log for TLS messages
if grep -q "Upgrading connection to TLS" /tmp/slave_tls.log; then
    echo "✓ Slave: TLS upgrade logged"
else
    echo "✗ Slave: TLS upgrade not logged"
    exit 1
fi

if grep -q "TLS.*handshake" /tmp/slave_tls.log; then
    echo "✓ Slave: TLS handshake logged"
else
    echo "✗ Slave: TLS handshake not logged"
    exit 1
fi

echo
echo "=== All TLS/SSL checks passed! ==="
echo "Verbose logging is working correctly for TLS/SSL scenarios."
