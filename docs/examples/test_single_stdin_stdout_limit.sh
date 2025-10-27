#!/bin/bash
# Test single stdin/stdout session limit
# This verifies that stdin/stdout piping (no --exec) is still limited to one connection
# for both master and slave listeners.

set -e

cd "$(dirname "$0")/../.."

echo "=== Step 14: Manual Verification - Single stdin/stdout Session Limit ==="
echo ""

# Cleanup function
cleanup() {
    echo "Cleaning up..."
    pkill -9 -f "goncat.elf" 2>/dev/null || true
    sleep 1
}

trap cleanup EXIT

echo "=== Test A: Slave listener with stdin/stdout piping ==="
echo ""

# Start slave listener (no --exec means it will pipe its stdin/stdout)
echo "1. Starting slave listener on port 12346..."
./dist/goncat.elf slave listen 'tcp://*:12346' &
SLAVE_PID=$!
sleep 2

# Verify slave is listening
echo "2. Verifying slave is listening..."
if ss -tlnp 2>/dev/null | grep -q 12346; then
    echo "   ✓ Slave is listening on port 12346"
else
    echo "   ✗ ERROR: Slave is not listening"
    exit 1
fi

# Connect first master (NO --exec, so it will pipe stdin/stdout of slave)
echo ""
echo "3. Connecting first master (stdin/stdout piping - NO --exec)..."
# Use a background process that keeps the connection open longer
(sleep 10; echo "exit") | ./dist/goncat.elf master connect tcp://localhost:12346 &
MASTER1_PID=$!
sleep 3  # Wait for first connection to be fully established

# Try to connect second master - should be rejected or timeout because semaphore is held
echo "4. Attempting second master connection while first is still active..."
START_TIME=$(date +%s)
(sleep 1; echo "exit") | timeout 5 ./dist/goncat.elf master connect tcp://localhost:12346 &
MASTER2_PID=$!
sleep 4  # Wait to see if second connection gets through

# Check if second connection is still running or exited
if kill -0 $MASTER2_PID 2>/dev/null; then
    echo "   ✓ Second connection is blocked/waiting (expected behavior)"
    kill -9 $MASTER2_PID 2>/dev/null || true
    RESULT_A="PASS"
else
    wait $MASTER2_PID
    EXIT2=$?
    if [ $EXIT2 -ne 0 ]; then
        echo "   ✓ Second connection failed/timed out (expected behavior)"
        RESULT_A="PASS"
    else
        echo "   ✗ Second connection succeeded (unexpected - should be blocked)"
        RESULT_A="FAIL"
    fi
fi

# Cleanup first test
kill -9 $MASTER1_PID 2>/dev/null || true
kill -9 $SLAVE_PID 2>/dev/null || true
sleep 2

echo ""
echo "=== Test B: Master listener ==="
echo ""

# Start master listener
echo "1. Starting master listener on port 12347..."
./dist/goncat.elf master listen 'tcp://*:12347' --exec /bin/sh &
MASTER_PID=$!
sleep 2

# Verify master is listening
echo "2. Verifying master is listening..."
if ss -tlnp 2>/dev/null | grep -q 12347; then
    echo "   ✓ Master is listening on port 12347"
else
    echo "   ✗ ERROR: Master is not listening"
    exit 1
fi

# Connect first slave
echo ""
echo "3. Connecting first slave..."
(sleep 10; echo "exit") | ./dist/goncat.elf slave connect tcp://localhost:12347 &
SLAVE1_PID=$!
sleep 3  # Wait for first connection to be fully established

# Try second slave - should be rejected
echo "4. Attempting second slave connection while first is still active..."
(sleep 1; echo "exit") | timeout 5 ./dist/goncat.elf slave connect tcp://localhost:12347 &
SLAVE2_PID=$!
sleep 4  # Wait to see if second connection gets through

# Check if second connection is still running or exited
if kill -0 $SLAVE2_PID 2>/dev/null; then
    echo "   ✓ Second connection is blocked/waiting (expected behavior)"
    kill -9 $SLAVE2_PID 2>/dev/null || true
    RESULT_B="PASS"
else
    wait $SLAVE2_PID
    EXIT2=$?
    if [ $EXIT2 -ne 0 ]; then
        echo "   ✓ Second connection failed/timed out (expected behavior)"
        RESULT_B="PASS"
    else
        echo "   ✗ Second connection succeeded (unexpected - should be blocked)"
        RESULT_B="FAIL"
    fi
fi

# Cleanup
kill -9 $SLAVE1_PID 2>/dev/null || true
kill -9 $MASTER_PID 2>/dev/null || true

# Overall result
echo ""
echo "=== RESULTS ==="
echo "Test A (Slave listener stdin/stdout): $RESULT_A"
echo "Test B (Master listener): $RESULT_B"
echo ""

if [ "$RESULT_A" = "PASS" ] && [ "$RESULT_B" = "PASS" ]; then
    echo "=== ✓ TEST PASSED ==="
    echo "Both tests confirm stdin/stdout piping is limited to one connection."
    exit 0
else
    echo "=== ✗ TEST FAILED ==="
    echo "One or more tests failed to limit connections properly."
    exit 1
fi
