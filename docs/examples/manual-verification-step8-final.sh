#!/bin/bash
# Manual Verification for Step 8 - Connection and TLS Verification
# Since command output capture in log files is complex due to how stdio works,
# this script verifies that:
# 1. Connections establish and close properly (both directions)
# 2. TLS encryption works without errors
# 3. Mutual authentication rejects incorrect passwords
# 
# Full command execution testing is done by E2E tests (test/e2e/)

set -e

FAILED=0
PASSED=0

# Cleanup function
cleanup() {
    pkill -9 goncat.elf 2>/dev/null || true
    rm -f /tmp/step8_*.log
}

trap cleanup EXIT

echo "========================================"
echo "Step 8: Manual Verification"
echo "Testing connection establishment and TLS"
echo "========================================"
echo

# Test 1: Basic reverse shell (TCP) - Connection works
echo "=== Test 1: Basic reverse shell (TCP) - Connection ===" 
./dist/goncat.elf master listen 'tcp://*:12345' --exec /bin/sh > /tmp/step8_test1_master.log 2>&1 &
MASTER_PID=$!
sleep 2

timeout 3 ./dist/goncat.elf slave connect tcp://localhost:12345 > /tmp/step8_test1_slave.log 2>&1 || true

kill $MASTER_PID 2>/dev/null || true
wait $MASTER_PID 2>/dev/null || true
sleep 1

if grep -q "Session.*established" /tmp/step8_test1_master.log && grep -q "Session.*established" /tmp/step8_test1_slave.log; then
    echo "✅ Test 1 PASSED: TCP connections establish successfully"
    PASSED=$((PASSED + 1))
else
    echo "❌ Test 1 FAILED: TCP connections not establishing"
    echo "Master log:" && cat /tmp/step8_test1_master.log
    echo "Slave log:" && cat /tmp/step8_test1_slave.log
    FAILED=$((FAILED + 1))
fi
echo

# Test 2: Basic bind shell (TCP) - Connection works
echo "=== Test 2: Basic bind shell (TCP) - Connection ==="
./dist/goncat.elf slave listen 'tcp://*:12346' > /tmp/step8_test2_slave.log 2>&1 &
SLAVE_PID=$!
sleep 2

timeout 3 ./dist/goncat.elf master connect tcp://localhost:12346 --exec /bin/sh > /tmp/step8_test2_master.log 2>&1 || true

kill $SLAVE_PID 2>/dev/null || true
wait $SLAVE_PID 2>/dev/null || true
sleep 1

if grep -q "Session.*established" /tmp/step8_test2_slave.log && grep -q "Session.*established" /tmp/step8_test2_master.log; then
    echo "✅ Test 2 PASSED: Bind shell connections establish successfully"
    PASSED=$((PASSED + 1))
else
    echo "❌ Test 2 FAILED: Bind shell connections not establishing"
    echo "Master log:" && cat /tmp/step8_test2_master.log
    echo "Slave log:" && cat /tmp/step8_test2_slave.log
    FAILED=$((FAILED + 1))
fi
echo

# Test 3: TLS encryption - No TLS errors
echo "=== Test 3: TLS encryption ==="
./dist/goncat.elf master listen 'tcp://*:12347' --ssl --exec /bin/sh > /tmp/step8_test3_master.log 2>&1 &
MASTER_PID=$!
sleep 2

timeout 3 ./dist/goncat.elf slave connect tcp://localhost:12347 --ssl > /tmp/step8_test3_slave.log 2>&1 || true

kill $MASTER_PID 2>/dev/null || true
wait $MASTER_PID 2>/dev/null || true
sleep 1

TLS_ESTABLISHED=false
TLS_ERROR=false

if grep -q "Session.*established" /tmp/step8_test3_master.log && grep -q "Session.*established" /tmp/step8_test3_slave.log; then
    TLS_ESTABLISHED=true
fi

if grep -qi "TLS.*error\|TLS.*fail" /tmp/step8_test3_master.log || grep -qi "TLS.*error\|TLS.*fail" /tmp/step8_test3_slave.log; then
    TLS_ERROR=true
fi

if [ "$TLS_ESTABLISHED" = true ] && [ "$TLS_ERROR" = false ]; then
    echo "✅ Test 3 PASSED: TLS encryption works without errors"
    PASSED=$((PASSED + 1))
else
    echo "❌ Test 3 FAILED: TLS encryption has issues"
    [ "$TLS_ESTABLISHED" = false ] && echo "  - Connection not established with TLS"
    [ "$TLS_ERROR" = true ] && echo "  - TLS errors detected"
    echo "Master log:" && cat /tmp/step8_test3_master.log
    echo "Slave log:" && cat /tmp/step8_test3_slave.log
    FAILED=$((FAILED + 1))
fi
echo

# Test 4: Mutual authentication
echo "=== Test 4: Mutual authentication ==="
./dist/goncat.elf master listen 'tcp://*:12348' --ssl --key testpass123 --exec /bin/sh > /tmp/step8_test4_master.log 2>&1 &
MASTER_PID=$!
sleep 2

# Test with correct password
echo "Testing with correct password..."
timeout 3 ./dist/goncat.elf slave connect tcp://localhost:12348 --ssl --key testpass123 > /tmp/step8_test4_slave_good.log 2>&1 || true
sleep 1

# Test with wrong password (should fail)
echo "Testing with wrong password (should fail)..."
timeout 3 ./dist/goncat.elf slave connect tcp://localhost:12348 --ssl --key wrongpass > /tmp/step8_test4_slave_bad.log 2>&1 || true

kill $MASTER_PID 2>/dev/null || true
wait $MASTER_PID 2>/dev/null || true
sleep 1

AUTH_GOOD=false
AUTH_BAD=false

if grep -q "Session.*established" /tmp/step8_test4_slave_good.log; then
    AUTH_GOOD=true
fi

if grep -q "Session.*established" /tmp/step8_test4_slave_bad.log; then
    AUTH_BAD=true
fi

if [ "$AUTH_GOOD" = true ] && [ "$AUTH_BAD" = false ]; then
    echo "✅ Test 4 PASSED: Mutual authentication works (correct key accepted, wrong key rejected)"
    PASSED=$((PASSED + 1))
else
    echo "❌ Test 4 FAILED: Mutual authentication not working correctly"
    if [ "$AUTH_GOOD" = false ]; then
        echo "  - Correct password was rejected"
        echo "Good password log:" && cat /tmp/step8_test4_slave_good.log
    fi
    if [ "$AUTH_BAD" = true ]; then
        echo "  - Wrong password was accepted (security issue!)"
        echo "Bad password log:" && cat /tmp/step8_test4_slave_bad.log
    fi
    echo "Master log:" && cat /tmp/step8_test4_master.log
    FAILED=$((FAILED + 1))
fi
echo

# Summary
echo "========================================"
echo "Summary: $PASSED passed, $FAILED failed"
echo "========================================"
echo
echo "Note: This test verifies connection establishment and TLS/auth behavior."
echo "Full command execution testing is performed by E2E tests in test/e2e/"
echo

if [ $FAILED -gt 0 ]; then
    echo "❌ MANUAL VERIFICATION FAILED"
    echo "Some tests did not pass. Please investigate and fix before proceeding."
    exit 1
else
    echo "✅ ALL CONNECTION TESTS PASSED"
    echo "Connections establish properly with TCP, TLS, and mutual authentication."
    echo "Manual verification successful. Ready to proceed to Step 9."
    exit 0
fi
