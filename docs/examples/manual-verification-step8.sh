#!/bin/bash
# Manual Verification for Step 8
# Tests all 4 critical scenarios from the plan

set -e

FAILED=0
PASSED=0

# Cleanup function
cleanup() {
    pkill -9 goncat.elf 2>/dev/null || true
    rm -f /tmp/test*.log
}

trap cleanup EXIT

echo "========================================"
echo "Step 8: Manual Verification"
echo "========================================"
echo

# Test 1: Basic reverse shell (TCP)
echo "=== Test 1: Basic reverse shell (TCP) ==="
./dist/goncat.elf master listen 'tcp://*:12345' --exec /bin/sh > /tmp/test1_master.log 2>&1 &
MASTER_PID=$!
sleep 2

echo "echo 'REVERSE_SHELL_TEST' && exit" | timeout 5 ./dist/goncat.elf slave connect tcp://localhost:12345 > /tmp/test1_slave.log 2>&1 || true

kill $MASTER_PID 2>/dev/null || true
wait $MASTER_PID 2>/dev/null || true
sleep 1

if grep -q "REVERSE_SHELL_TEST" /tmp/test1_slave.log || grep -q "REVERSE_SHELL_TEST" /tmp/test1_master.log; then
    echo "✅ Test 1 PASSED: Reverse shell works"
    PASSED=$((PASSED + 1))
else
    echo "❌ Test 1 FAILED: Reverse shell not working"
    echo "Master log:"
    cat /tmp/test1_master.log
    echo "Slave log:"
    cat /tmp/test1_slave.log
    FAILED=$((FAILED + 1))
fi
echo

# Test 2: Basic bind shell (TCP)
echo "=== Test 2: Basic bind shell (TCP) ==="
./dist/goncat.elf slave listen 'tcp://*:12346' > /tmp/test2_slave.log 2>&1 &
SLAVE_PID=$!
sleep 2

echo "echo 'BIND_SHELL_TEST' && exit" | timeout 5 ./dist/goncat.elf master connect tcp://localhost:12346 --exec /bin/sh > /tmp/test2_master.log 2>&1 || true

kill $SLAVE_PID 2>/dev/null || true
wait $SLAVE_PID 2>/dev/null || true
sleep 1

if grep -q "BIND_SHELL_TEST" /tmp/test2_master.log || grep -q "BIND_SHELL_TEST" /tmp/test2_slave.log; then
    echo "✅ Test 2 PASSED: Bind shell works"
    PASSED=$((PASSED + 1))
else
    echo "❌ Test 2 FAILED: Bind shell not working"
    echo "Master log:"
    cat /tmp/test2_master.log
    echo "Slave log:"
    cat /tmp/test2_slave.log
    FAILED=$((FAILED + 1))
fi
echo

# Test 3: TLS encryption
echo "=== Test 3: TLS encryption ==="
./dist/goncat.elf master listen 'tcp://*:12347' --ssl --exec /bin/sh > /tmp/test3_master.log 2>&1 &
MASTER_PID=$!
sleep 2

echo "echo 'TLS_TEST' && exit" | timeout 5 ./dist/goncat.elf slave connect tcp://localhost:12347 --ssl > /tmp/test3_slave.log 2>&1 || true

kill $MASTER_PID 2>/dev/null || true
wait $MASTER_PID 2>/dev/null || true
sleep 1

if grep -q "TLS_TEST" /tmp/test3_slave.log || grep -q "TLS_TEST" /tmp/test3_master.log; then
    echo "✅ Test 3 PASSED: TLS encryption works"
    PASSED=$((PASSED + 1))
else
    echo "❌ Test 3 FAILED: TLS encryption not working"
    echo "Master log:"
    cat /tmp/test3_master.log
    echo "Slave log:"
    cat /tmp/test3_slave.log
    FAILED=$((FAILED + 1))
fi
echo

# Test 4: Mutual authentication
echo "=== Test 4: Mutual authentication ==="
./dist/goncat.elf master listen 'tcp://*:12348' --ssl --key testpass123 --exec /bin/sh > /tmp/test4_master.log 2>&1 &
MASTER_PID=$!
sleep 2

# Test with correct password
echo "Testing with correct password..."
echo "echo 'AUTH_SUCCESS' && exit" | timeout 5 ./dist/goncat.elf slave connect tcp://localhost:12348 --ssl --key testpass123 > /tmp/test4_slave_good.log 2>&1 || true
sleep 1

# Test with wrong password
echo "Testing with wrong password..."
echo "echo 'SHOULD_FAIL'" | timeout 5 ./dist/goncat.elf slave connect tcp://localhost:12348 --ssl --key wrongpass > /tmp/test4_slave_bad.log 2>&1 || true

kill $MASTER_PID 2>/dev/null || true
wait $MASTER_PID 2>/dev/null || true
sleep 1

AUTH_GOOD=false
AUTH_BAD=true

if grep -q "AUTH_SUCCESS" /tmp/test4_slave_good.log || grep -q "AUTH_SUCCESS" /tmp/test4_master.log; then
    AUTH_GOOD=true
fi

if grep -q "SHOULD_FAIL" /tmp/test4_slave_bad.log; then
    AUTH_BAD=false
fi

if [ "$AUTH_GOOD" = true ] && [ "$AUTH_BAD" = true ]; then
    echo "✅ Test 4 PASSED: Mutual authentication works (correct key accepted, wrong key rejected)"
    PASSED=$((PASSED + 1))
else
    echo "❌ Test 4 FAILED: Mutual authentication not working correctly"
    if [ "$AUTH_GOOD" = false ]; then
        echo "  - Correct password was rejected"
        echo "Good password log:"
        cat /tmp/test4_slave_good.log
    fi
    if [ "$AUTH_BAD" = false ]; then
        echo "  - Wrong password was accepted"
        echo "Bad password log:"
        cat /tmp/test4_slave_bad.log
    fi
    echo "Master log:"
    cat /tmp/test4_master.log
    FAILED=$((FAILED + 1))
fi
echo

# Summary
echo "========================================"
echo "Summary: $PASSED passed, $FAILED failed"
echo "========================================"

if [ $FAILED -gt 0 ]; then
    echo "❌ MANUAL VERIFICATION FAILED"
    echo "Some tests did not pass. Please investigate and fix before proceeding."
    exit 1
else
    echo "✅ ALL TESTS PASSED"
    echo "Manual verification successful. Ready to proceed to Step 9."
    exit 0
fi
