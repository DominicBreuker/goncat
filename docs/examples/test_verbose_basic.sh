#!/bin/bash
# Manual verification script for Step 13: Basic reverse shell with verbose logging
# This script tests that verbose logging works correctly for basic connection scenarios

set -e

echo "=== Step 13: Manual verification - Basic reverse shell with verbose logging ==="
echo

# Cleanup function
cleanup() {
    echo "Cleaning up..."
    kill $MASTER_PID 2>/dev/null || true
    pkill -9 goncat.elf 2>/dev/null || true
    rm -f /tmp/master_verbose.log /tmp/slave_verbose.log
}

trap cleanup EXIT

# Terminal 1: Start master with verbose logging
echo "Starting master with verbose logging..."
./dist/goncat.elf master listen 'tcp://*:12345' --exec /bin/sh --verbose > /tmp/master_verbose.log 2>&1 &
MASTER_PID=$!
sleep 2

# Terminal 2: Connect slave with verbose logging
echo "Connecting slave with verbose logging..."
echo "echo 'VERBOSE_TEST' && exit" | timeout 5 ./dist/goncat.elf slave connect tcp://localhost:12345 --verbose > /tmp/slave_verbose.log 2>&1 || true

# Wait a moment for logs to flush
sleep 1

echo
echo "=== Master verbose log ==="
cat /tmp/master_verbose.log
echo

echo "=== Slave verbose log ==="
cat /tmp/slave_verbose.log
echo

# Verify expected verbose messages are present
echo "=== Verification ==="
echo

# Check master log for expected messages
if grep -q "\[v\]" /tmp/master_verbose.log; then
    echo "✓ Master: Verbose messages found"
else
    echo "✗ Master: No verbose messages found"
    exit 1
fi

if grep -q "Creating listener" /tmp/master_verbose.log; then
    echo "✓ Master: Listener creation logged"
else
    echo "✗ Master: Listener creation not logged"
    exit 1
fi

if grep -q "Opening yamux session" /tmp/master_verbose.log; then
    echo "✓ Master: Yamux session creation logged"
else
    echo "✗ Master: Yamux session creation not logged"
    exit 1
fi

if grep -q "Sending Hello message" /tmp/master_verbose.log; then
    echo "✓ Master: Handshake logged"
else
    echo "✗ Master: Handshake not logged"
    exit 1
fi

# Check slave log for expected messages
if grep -q "\[v\]" /tmp/slave_verbose.log; then
    echo "✓ Slave: Verbose messages found"
else
    echo "✗ Slave: No verbose messages found"
    exit 1
fi

if grep -q "Dialing.*using protocol" /tmp/slave_verbose.log; then
    echo "✓ Slave: Connection attempt logged"
else
    echo "✗ Slave: Connection attempt not logged"
    exit 1
fi

if grep -q "Connection established" /tmp/slave_verbose.log; then
    echo "✓ Slave: Connection establishment logged"
else
    echo "✗ Slave: Connection establishment not logged"
    exit 1
fi

if grep -q "Accepting yamux session" /tmp/slave_verbose.log; then
    echo "✓ Slave: Yamux session acceptance logged"
else
    echo "✗ Slave: Yamux session acceptance not logged"
    exit 1
fi

echo
echo "=== All checks passed! ==="
echo "Verbose logging is working correctly for basic reverse shell scenario."
