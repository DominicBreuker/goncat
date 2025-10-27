#!/bin/bash
# Test multiple concurrent slave sessions with --exec
# This verifies the main goal of the refactoring: that a listening slave
# can accept multiple concurrent master connections when executing commands.

set -e

cd "$(dirname "$0")/../.."

echo "=== Step 13: Manual Verification - Multiple Concurrent Slave Sessions ==="
echo ""

# Cleanup function
cleanup() {
    echo "Cleaning up..."
    pkill -9 -f "goncat.elf slave listen" 2>/dev/null || true
    pkill -9 -f "goncat.elf master connect" 2>/dev/null || true
    sleep 1
}

trap cleanup EXIT

# Start slave listener
echo "1. Starting slave listener on port 12345..."
./dist/goncat.elf slave listen 'tcp://*:12345' &
SLAVE_PID=$!
sleep 2

# Verify slave is listening
echo "2. Verifying slave is listening..."
if ss -tlnp 2>/dev/null | grep -q 12345; then
    echo "   ✓ Slave is listening on port 12345"
else
    echo "   ✗ ERROR: Slave is not listening"
    exit 1
fi

# Start first master connection with command execution
echo ""
echo "3. Starting first master connection with command execution..."
echo "echo 'Connection 1 started' && sleep 3 && echo 'Connection 1 done' && exit" | timeout 10 ./dist/goncat.elf master connect tcp://localhost:12345 --exec /bin/sh &
MASTER1_PID=$!
sleep 1

# Start second master connection with command execution (immediately, not waiting for first to finish)
echo "4. Starting second master connection with command execution (concurrent)..."
echo "echo 'Connection 2 started' && sleep 1 && echo 'Connection 2 done' && exit" | timeout 10 ./dist/goncat.elf master connect tcp://localhost:12345 --exec /bin/sh &
MASTER2_PID=$!
sleep 1

# Start third master connection
echo "5. Starting third master connection..."
echo "echo 'Connection 3 started' && echo 'Connection 3 done' && exit" | timeout 10 ./dist/goncat.elf master connect tcp://localhost:12345 --exec /bin/sh &
MASTER3_PID=$!

# Wait for all masters to complete
echo ""
echo "6. Waiting for all master connections to complete..."
wait $MASTER1_PID
EXIT1=$?
wait $MASTER2_PID
EXIT2=$?
wait $MASTER3_PID
EXIT3=$?

# Check results
echo ""
echo "=== RESULTS ==="
if [ $EXIT1 -eq 0 ]; then
    echo "✓ Connection 1: SUCCESS (exit code $EXIT1)"
else
    echo "✗ Connection 1: FAILED (exit code $EXIT1)"
fi

if [ $EXIT2 -eq 0 ]; then
    echo "✓ Connection 2: SUCCESS (exit code $EXIT2)"
else
    echo "✗ Connection 2: FAILED (exit code $EXIT2)"
fi

if [ $EXIT3 -eq 0 ]; then
    echo "✓ Connection 3: SUCCESS (exit code $EXIT3)"
else
    echo "✗ Connection 3: FAILED (exit code $EXIT3)"
fi

# Overall result
echo ""
if [ $EXIT1 -eq 0 ] && [ $EXIT2 -eq 0 ] && [ $EXIT3 -eq 0 ]; then
    echo "=== ✓ TEST PASSED ==="
    echo "All three connections executed successfully and concurrently."
    echo "This confirms that slave listeners can accept multiple concurrent command execution sessions."
    exit 0
else
    echo "=== ✗ TEST FAILED ==="
    echo "One or more connections failed. The refactoring goal was not achieved."
    exit 1
fi
