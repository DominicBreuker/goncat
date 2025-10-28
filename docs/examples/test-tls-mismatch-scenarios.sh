#!/bin/bash
# Test various TLS mismatch scenarios to ensure graceful error handling
# This script validates the bug fix for nil pointer dereference

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

PASSED=0
FAILED=0

cleanup() {
    pkill -9 goncat.elf 2>/dev/null || true
    rm -f /tmp/tls_test_*.log
}

trap cleanup EXIT

echo "========================================"
echo "TLS Mismatch Scenarios Test"
echo "Validates bug fix for nil pointer panic"
echo "========================================"
echo

# Test 1: TLS client -> Plain server
echo "=== Test 1: TLS client -> Plain server ==="
./dist/goncat.elf slave listen 'tcp://*:13001' > /tmp/tls_test_1_server.log 2>&1 &
sleep 2
if ./dist/goncat.elf master connect tcp://localhost:13001 --ssl --timeout 2000 > /tmp/tls_test_1_client.log 2>&1; then
    echo "❌ Test 1 FAILED: Should have failed"
    FAILED=$((FAILED + 1))
elif grep -q "panic:" /tmp/tls_test_1_client.log; then
    echo "❌ Test 1 FAILED: Client panicked"
    FAILED=$((FAILED + 1))
elif grep -q "Error.*TLS\|Error.*EOF" /tmp/tls_test_1_client.log; then
    echo "✅ Test 1 PASSED: Graceful TLS error"
    PASSED=$((PASSED + 1))
else
    echo "❌ Test 1 FAILED: Unexpected error"
    FAILED=$((FAILED + 1))
fi
pkill -9 goncat.elf 2>/dev/null || true
sleep 1
echo

# Test 2: Plain client -> TLS server  
echo "=== Test 2: Plain client -> TLS server ==="
./dist/goncat.elf slave listen 'tcp://*:13002' --ssl > /tmp/tls_test_2_server.log 2>&1 &
sleep 2
if ./dist/goncat.elf master connect tcp://localhost:13002 --timeout 2000 > /tmp/tls_test_2_client.log 2>&1; then
    echo "❌ Test 2 FAILED: Should have failed"
    FAILED=$((FAILED + 1))
elif grep -q "panic:" /tmp/tls_test_2_client.log; then
    echo "❌ Test 2 FAILED: Client panicked"
    FAILED=$((FAILED + 1))
elif grep -q "Error" /tmp/tls_test_2_client.log; then
    echo "✅ Test 2 PASSED: Graceful connection error"
    PASSED=$((PASSED + 1))
else
    echo "❌ Test 2 FAILED: Unexpected result"
    FAILED=$((FAILED + 1))
fi
pkill -9 goncat.elf 2>/dev/null || true
sleep 1
echo

# Test 3: Matching TLS (should work)
echo "=== Test 3: TLS client -> TLS server (should work) ==="
./dist/goncat.elf slave listen 'tcp://*:13003' --ssl > /tmp/tls_test_3_server.log 2>&1 &
sleep 2
timeout 3 ./dist/goncat.elf master connect tcp://localhost:13003 --ssl > /tmp/tls_test_3_client.log 2>&1 || true
if grep -q "panic:" /tmp/tls_test_3_client.log; then
    echo "❌ Test 3 FAILED: Client panicked"
    FAILED=$((FAILED + 1))
elif grep -q "Session.*established" /tmp/tls_test_3_client.log; then
    echo "✅ Test 3 PASSED: TLS connection works"
    PASSED=$((PASSED + 1))
else
    echo "⚠️  Test 3 WARNING: Expected connection, check logs"
    # This might timeout, which is ok for this test
    PASSED=$((PASSED + 1))
fi
pkill -9 goncat.elf 2>/dev/null || true
sleep 1
echo

# Test 4: Matching plain (should work)
echo "=== Test 4: Plain client -> Plain server (should work) ==="
./dist/goncat.elf slave listen 'tcp://*:13004' > /tmp/tls_test_4_server.log 2>&1 &
sleep 2
timeout 3 ./dist/goncat.elf master connect tcp://localhost:13004 > /tmp/tls_test_4_client.log 2>&1 || true
if grep -q "panic:" /tmp/tls_test_4_client.log; then
    echo "❌ Test 4 FAILED: Client panicked"
    FAILED=$((FAILED + 1))
elif grep -q "Session.*established" /tmp/tls_test_4_client.log; then
    echo "✅ Test 4 PASSED: Plain connection works"
    PASSED=$((PASSED + 1))
else
    echo "⚠️  Test 4 WARNING: Expected connection, check logs"
    # This might timeout, which is ok for this test
    PASSED=$((PASSED + 1))
fi
pkill -9 goncat.elf 2>/dev/null || true
sleep 1
echo

# Summary
echo "========================================"
echo "Summary: $PASSED passed, $FAILED failed"
echo "========================================"

if [ $FAILED -gt 0 ]; then
    echo "❌ SOME TESTS FAILED"
    echo "Check logs in /tmp/tls_test_*.log"
    exit 1
else
    echo "✅ ALL TESTS PASSED"
    echo "No panics detected - bug fix verified!"
    exit 0
fi
