#!/bin/bash
# Test 2: Basic bind shell (TCP)
# This tests slave listen with master connect

set -e

echo "=== Test 2: Basic bind shell (TCP) ==="

# Start slave listener in background
./dist/goncat.elf slave listen 'tcp://*:12346' &
SLAVE_PID=$!
echo "Slave listening (PID: $SLAVE_PID)"

# Give it time to start
sleep 2

# Connect as master and send command
echo "Connecting master and sending test command..."
OUTPUT=$(echo "echo 'BIND_SHELL_TEST' && exit" | timeout 5 ./dist/goncat.elf master connect tcp://localhost:12346 --exec /bin/sh 2>&1)

# Cleanup
echo "Cleaning up..."
kill $SLAVE_PID 2>/dev/null || true
wait $SLAVE_PID 2>/dev/null || true

# Verify output
if echo "$OUTPUT" | grep -q "BIND_SHELL_TEST"; then
    echo "✅ Test 2 PASSED: Bind shell works correctly"
    echo "   Output: $OUTPUT"
    exit 0
else
    echo "❌ Test 2 FAILED: Expected 'BIND_SHELL_TEST' in output"
    echo "   Got: $OUTPUT"
    exit 1
fi
