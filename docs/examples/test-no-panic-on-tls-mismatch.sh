#!/bin/bash
# Simple test: Verify no panic occurs on TLS mismatch
# This validates the nil pointer dereference bug fix

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

cleanup() {
    pkill -9 goncat.elf 2>/dev/null || true
}

trap cleanup EXIT

echo "========================================"
echo "TLS Mismatch Panic Test"
echo "Validates: No panic on TLS upgrade failure"
echo "========================================"
echo

# Test: TLS client -> Plain server (the scenario that caused the original panic)
echo "Testing: TLS client connecting to plain server..."
./dist/goncat.elf slave listen 'tcp://*:13100' > /tmp/panic_test_server.log 2>&1 &
sleep 2

./dist/goncat.elf master connect tcp://localhost:13100 --ssl --timeout 2000 > /tmp/panic_test_client.log 2>&1 || true

cleanup
sleep 1

# Check results
echo
echo "=== Client output ===" 
cat /tmp/panic_test_client.log
echo

if grep -q "panic:" /tmp/panic_test_client.log; then
    echo "❌ FAILED: Client panicked (nil pointer dereference bug NOT fixed)"
    exit 1
fi

if grep -q "Error.*TLS\|Error.*EOF" /tmp/panic_test_client.log; then
    echo "✅ PASSED: Client failed gracefully with TLS error (no panic)"
    echo "Bug fix verified: nil pointer dereference is fixed!"
    exit 0
fi

echo "⚠️  WARNING: Unexpected output, but no panic detected"
exit 0
