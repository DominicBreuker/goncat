#!/bin/bash
# Manual Verification Tests for Transport API Refactoring
# These tests verify that the refactored transport layer works correctly with real binaries

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BINARY="$REPO_ROOT/dist/goncat.elf"

echo "=========================================="
echo "Manual Verification Tests"
echo "=========================================="
echo ""

# Ensure binary exists
if [ ! -f "$BINARY" ]; then
    echo "ERROR: Binary not found at $BINARY"
    echo "Please run: make build-linux"
    exit 1
fi

echo "Using binary: $BINARY"
echo ""

# Test 1: Version Check
echo "Test 1: Version Check"
echo "----------------------"
VERSION=$("$BINARY" version)
if [ "$VERSION" = "0.0.1" ]; then
    echo "✅ PASS: Version is correct ($VERSION)"
else
    echo "❌ FAIL: Expected version 0.0.1, got: $VERSION"
    exit 1
fi
echo ""

# Test 2: Help Commands
echo "Test 2: Help Commands"
echo "---------------------"
if "$BINARY" --help | grep -q "goncat"; then
    echo "✅ PASS: Main help works"
else
    echo "❌ FAIL: Main help failed"
    exit 1
fi

if "$BINARY" master --help | grep -q "master"; then
    echo "✅ PASS: Master help works"
else
    echo "❌ FAIL: Master help failed"
    exit 1
fi

if "$BINARY" slave --help | grep -q "slave"; then
    echo "✅ PASS: Slave help works"
else
    echo "❌ FAIL: Slave help failed"
    exit 1
fi
echo ""

# Test 3: Basic TCP Reverse Shell (Master Listen, Slave Connect)
echo "Test 3: TCP Reverse Shell (Master Listen, Slave Connect)"
echo "----------------------------------------------------------"

# Cleanup any existing processes
pkill -9 goncat.elf 2>/dev/null || true
sleep 1

# Start master listener in background
"$BINARY" master listen 'tcp://*:12345' --exec /bin/sh > /tmp/master.log 2>&1 &
MASTER_PID=$!
echo "Started master listener (PID: $MASTER_PID)"

# Wait for master to start
sleep 2

# Check if master is listening
if ss -tlnp 2>/dev/null | grep -q 12345 || netstat -tlnp 2>/dev/null | grep -q 12345; then
    echo "✅ PASS: Master is listening on port 12345"
else
    echo "❌ FAIL: Master is not listening on port 12345"
    kill $MASTER_PID 2>/dev/null || true
    cat /tmp/master.log
    exit 1
fi

# Test slave connection with simple command
echo "Testing slave connection..."
TEST_OUTPUT=$(echo "echo 'TRANSPORT_API_TEST'; exit" | timeout 5 "$BINARY" slave connect tcp://localhost:12345 2>&1 || true)

# Check if connection was established (which means transport layer works)
if echo "$TEST_OUTPUT" | grep -q "Session with.*established"; then
    echo "✅ PASS: Slave connected successfully (transport layer working)"
else
    echo "❌ FAIL: Slave connection failed"
    echo "Output: $TEST_OUTPUT"
    kill $MASTER_PID 2>/dev/null || true
    cat /tmp/master.log
    exit 1
fi

# Cleanup
kill $MASTER_PID 2>/dev/null || true
sleep 1
echo ""

# Test 4: Basic TCP Bind Shell (Slave Listen, Master Connect)
echo "Test 4: TCP Bind Shell (Slave Listen, Master Connect)"
echo "------------------------------------------------------"

# Cleanup any existing processes
pkill -9 goncat.elf 2>/dev/null || true
sleep 1

# Start slave listener in background
"$BINARY" slave listen 'tcp://*:12346' > /tmp/slave.log 2>&1 &
SLAVE_PID=$!
echo "Started slave listener (PID: $SLAVE_PID)"

# Wait for slave to start
sleep 2

# Check if slave is listening
if ss -tlnp 2>/dev/null | grep -q 12346 || netstat -tlnp 2>/dev/null | grep -q 12346; then
    echo "✅ PASS: Slave is listening on port 12346"
else
    echo "❌ FAIL: Slave is not listening on port 12346"
    kill $SLAVE_PID 2>/dev/null || true
    cat /tmp/slave.log
    exit 1
fi

# Test master connection with simple command
echo "Testing master connection..."
TEST_OUTPUT=$(echo "echo 'BIND_SHELL_TEST'; exit" | timeout 5 "$BINARY" master connect tcp://localhost:12346 --exec /bin/sh 2>&1 || true)

# Check if connection was established (which means transport layer works)
if echo "$TEST_OUTPUT" | grep -q "Session with.*established"; then
    echo "✅ PASS: Master connected successfully (transport layer working)"
else
    echo "❌ FAIL: Master connection failed"
    echo "Output: $TEST_OUTPUT"
    kill $SLAVE_PID 2>/dev/null || true
    cat /tmp/slave.log
    exit 1
fi

# Cleanup
kill $SLAVE_PID 2>/dev/null || true
sleep 1
echo ""

# Final cleanup
pkill -9 goncat.elf 2>/dev/null || true
rm -f /tmp/master.log /tmp/slave.log

echo "=========================================="
echo "All Manual Verification Tests PASSED ✅"
echo "=========================================="
echo ""
echo "Summary:"
echo "  ✅ Version check"
echo "  ✅ Help commands"
echo "  ✅ TCP Reverse Shell (Master Listen, Slave Connect)"
echo "  ✅ TCP Bind Shell (Slave Listen, Master Connect)"
echo ""
echo "The refactored transport layer works correctly!"
