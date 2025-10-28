#!/bin/bash
# Test 1: Basic reverse shell (TCP)
# This tests master listen with slave connect

set -e

echo "=== Test 1: Basic reverse shell (TCP) ==="

# Start master listener in background
./dist/goncat.elf master listen 'tcp://*:12345' --exec /bin/sh &
MASTER_PID=$!
echo "Master listening (PID: $MASTER_PID)"

# Give it time to start
sleep 2

# Connect as slave and send command
echo "Connecting slave and sending test command..."
OUTPUT=$(echo "echo 'REVERSE_SHELL_TEST' && exit" | timeout 5 ./dist/goncat.elf slave connect tcp://localhost:12345 2>&1)

# Cleanup
echo "Cleaning up..."
kill $MASTER_PID 2>/dev/null || true
wait $MASTER_PID 2>/dev/null || true

# Verify output
if echo "$OUTPUT" | grep -q "REVERSE_SHELL_TEST"; then
    echo "✅ Test 1 PASSED: Reverse shell works correctly"
    echo "   Output: $OUTPUT"
    exit 0
else
    echo "❌ Test 1 FAILED: Expected 'REVERSE_SHELL_TEST' in output"
    echo "   Got: $OUTPUT"
    exit 1
fi
