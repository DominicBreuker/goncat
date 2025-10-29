#!/bin/bash
# Validation Script: SSL/TLS Encryption
# Purpose: Verify --ssl flag with comprehensive test matrix
# Test Matrix:
#   1. --ssl both sides → success + data token verification
#   2. Mismatched --ssl (master only) → failure
#   3. Mismatched --ssl (slave only) → failure
#   4. --ssl --key matching → success
#   5. --ssl with wrong --key → failure
#   6. --key without --ssl → CLI error
# Dependencies: bash, goncat binary

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track PIDs for cleanup
MASTER_PID=""
SLAVE_PID=""

cleanup() {
    [ -n "$MASTER_PID" ] && kill "$MASTER_PID" 2>/dev/null && wait "$MASTER_PID" 2>/dev/null
    [ -n "$SLAVE_PID" ] && kill "$SLAVE_PID" 2>/dev/null && wait "$SLAVE_PID" 2>/dev/null
    rm -f /tmp/goncat-ssl-*
}
trap cleanup EXIT

# Helper function: Poll for pattern in file
poll_for_pattern() {
    local file="$1"
    local pattern="$2"
    local timeout="${3:-10}"
    local start=$(date +%s)
    while true; do
        if [ -f "$file" ] && grep -qE "$pattern" "$file" 2>/dev/null; then
            return 0
        fi
        local now=$(date +%s)
        if [ $((now - start)) -ge "$timeout" ]; then
            return 1
        fi
        sleep 0.1
    done
}

if [ ! -f "$REPO_ROOT/dist/goncat.elf" ]; then
    echo -e "${YELLOW}Building goncat binary...${NC}"
    make build-linux
fi

echo -e "${GREEN}=== SSL/TLS Encryption Validation ===${NC}"

TRANSPORT="tcp"
PORT_BASE=12020
TEST_KEY="test_password_123"

# Test 1: --ssl on both sides → success
echo -e "${YELLOW}Test 1: --ssl on both sides (should succeed)${NC}"
PORT=$((PORT_BASE + 1))
TOKEN="SSL_BOTH_$$_$RANDOM"

(echo "echo $TOKEN"; sleep 1; echo "exit") | "$REPO_ROOT/dist/goncat.elf" master listen "${TRANSPORT}://*:${PORT}" --exec /bin/sh --ssl > /tmp/goncat-ssl-t1-master.log 2>&1 &
MASTER_PID=$!

if ! poll_for_pattern /tmp/goncat-ssl-t1-master.log "Listening on" 5; then
    echo -e "${RED}✗ Test 1: Master failed to start${NC}"
    exit 1
fi

"$REPO_ROOT/dist/goncat.elf" slave connect "${TRANSPORT}://localhost:${PORT}" --ssl > /tmp/goncat-ssl-t1-slave.log 2>&1 &
SLAVE_PID=$!

if ! poll_for_pattern /tmp/goncat-ssl-t1-master.log "Session with .* established" 10; then
    echo -e "${RED}✗ Test 1: Connection not established${NC}"
    exit 1
fi

if ! poll_for_pattern /tmp/goncat-ssl-t1-master.log "$TOKEN" 10; then
    echo -e "${RED}✗ Test 1: Data token not found${NC}"
    exit 1
fi

if ! poll_for_pattern /tmp/goncat-ssl-t1-master.log "Session with .* closed" 5; then
    echo -e "${RED}✗ Test 1: Session not closed properly${NC}"
    exit 1
fi

wait "$SLAVE_PID" 2>/dev/null
kill "$MASTER_PID" 2>/dev/null && wait "$MASTER_PID" 2>/dev/null
MASTER_PID=""
SLAVE_PID=""
echo -e "${GREEN}✓ Test 1 PASSED: SSL encryption working${NC}"

# Test 2: Mismatched --ssl (master only) → failure
echo -e "${YELLOW}Test 2: --ssl on master only (should fail)${NC}"
PORT=$((PORT_BASE + 2))

"$REPO_ROOT/dist/goncat.elf" master listen "${TRANSPORT}://*:${PORT}" --exec /bin/sh --ssl > /tmp/goncat-ssl-t2-master.log 2>&1 &
MASTER_PID=$!

if ! poll_for_pattern /tmp/goncat-ssl-t2-master.log "Listening on" 5; then
    echo -e "${RED}✗ Test 2: Master failed to start${NC}"
    exit 1
fi

"$REPO_ROOT/dist/goncat.elf" slave connect "${TRANSPORT}://localhost:${PORT}" > /tmp/goncat-ssl-t2-slave.log 2>&1 &
SLAVE_PID=$!

# Should NOT establish connection (TLS handshake failure)
sleep 3
if grep -qE "Session with .* established" /tmp/goncat-ssl-t2-master.log; then
    echo -e "${RED}✗ Test 2: Connection established (should have failed)${NC}"
    exit 1
fi

wait "$SLAVE_PID" 2>/dev/null

# Check for TLS/connection error in logs (exit code may be 0)
if ! grep -qE "(TLS|handshake|EOF|Error)" /tmp/goncat-ssl-t2-slave.log && \
   ! grep -qE "(TLS|handshake|Error)" /tmp/goncat-ssl-t2-master.log; then
    echo -e "${RED}✗ Test 2: No error detected (should have TLS handshake failure)${NC}"
    exit 1
fi

kill "$MASTER_PID" 2>/dev/null && wait "$MASTER_PID" 2>/dev/null
MASTER_PID=""
SLAVE_PID=""
echo -e "${GREEN}✓ Test 2 PASSED: Mismatched SSL properly rejected${NC}"

# Test 3: Mismatched --ssl (slave only) → failure
echo -e "${YELLOW}Test 3: --ssl on slave only (should fail)${NC}"
PORT=$((PORT_BASE + 3))

"$REPO_ROOT/dist/goncat.elf" master listen "${TRANSPORT}://*:${PORT}" --exec /bin/sh > /tmp/goncat-ssl-t3-master.log 2>&1 &
MASTER_PID=$!

if ! poll_for_pattern /tmp/goncat-ssl-t3-master.log "Listening on" 5; then
    echo -e "${RED}✗ Test 3: Master failed to start${NC}"
    exit 1
fi

"$REPO_ROOT/dist/goncat.elf" slave connect "${TRANSPORT}://localhost:${PORT}" --ssl > /tmp/goncat-ssl-t3-slave.log 2>&1 &
SLAVE_PID=$!

# Should NOT establish connection (TLS expected but not provided)
sleep 3
if grep -qE "Session with .* established" /tmp/goncat-ssl-t3-slave.log; then
    echo -e "${RED}✗ Test 3: Connection established (should have failed)${NC}"
    exit 1
fi

wait "$SLAVE_PID" 2>/dev/null

# Check for connection error in logs
if ! grep -qE "(Error|handshake|EOF)" /tmp/goncat-ssl-t3-slave.log && \
   ! grep -qE "(Error|handshake)" /tmp/goncat-ssl-t3-master.log; then
    echo -e "${RED}✗ Test 3: No error detected (should have connection failure)${NC}"
    exit 1
fi

kill "$MASTER_PID" 2>/dev/null && wait "$MASTER_PID" 2>/dev/null
MASTER_PID=""
SLAVE_PID=""
echo -e "${GREEN}✓ Test 3 PASSED: Mismatched SSL properly rejected${NC}"

# Test 4: --ssl --key matching → success
echo -e "${YELLOW}Test 4: --ssl --key matching (should succeed)${NC}"
PORT=$((PORT_BASE + 4))
TOKEN="SSL_KEY_$$_$RANDOM"

(echo "echo $TOKEN"; sleep 1; echo "exit") | "$REPO_ROOT/dist/goncat.elf" master listen "${TRANSPORT}://*:${PORT}" --exec /bin/sh --ssl --key "$TEST_KEY" > /tmp/goncat-ssl-t4-master.log 2>&1 &
MASTER_PID=$!

if ! poll_for_pattern /tmp/goncat-ssl-t4-master.log "Listening on" 5; then
    echo -e "${RED}✗ Test 4: Master failed to start${NC}"
    exit 1
fi

"$REPO_ROOT/dist/goncat.elf" slave connect "${TRANSPORT}://localhost:${PORT}" --ssl --key "$TEST_KEY" > /tmp/goncat-ssl-t4-slave.log 2>&1 &
SLAVE_PID=$!

if ! poll_for_pattern /tmp/goncat-ssl-t4-master.log "Session with .* established" 10; then
    echo -e "${RED}✗ Test 4: Connection not established${NC}"
    exit 1
fi

if ! poll_for_pattern /tmp/goncat-ssl-t4-master.log "$TOKEN" 10; then
    echo -e "${RED}✗ Test 4: Data token not found${NC}"
    exit 1
fi

wait "$SLAVE_PID" 2>/dev/null
kill "$MASTER_PID" 2>/dev/null && wait "$MASTER_PID" 2>/dev/null
MASTER_PID=""
SLAVE_PID=""
echo -e "${GREEN}✓ Test 4 PASSED: SSL with matching key working${NC}"

# Test 5: --ssl with wrong --key → failure
echo -e "${YELLOW}Test 5: --ssl with mismatched keys (should fail)${NC}"
PORT=$((PORT_BASE + 5))

"$REPO_ROOT/dist/goncat.elf" master listen "${TRANSPORT}://*:${PORT}" --exec /bin/sh --ssl --key "$TEST_KEY" > /tmp/goncat-ssl-t5-master.log 2>&1 &
MASTER_PID=$!

if ! poll_for_pattern /tmp/goncat-ssl-t5-master.log "Listening on" 5; then
    echo -e "${RED}✗ Test 5: Master failed to start${NC}"
    exit 1
fi

"$REPO_ROOT/dist/goncat.elf" slave connect "${TRANSPORT}://localhost:${PORT}" --ssl --key "WRONG_KEY" > /tmp/goncat-ssl-t5-slave.log 2>&1 &
SLAVE_PID=$!

# Should NOT establish connection (mutual TLS auth failure)
sleep 3
if grep -qE "Session with .* established" /tmp/goncat-ssl-t5-master.log; then
    echo -e "${RED}✗ Test 5: Connection established (should have failed)${NC}"
    exit 1
fi

wait "$SLAVE_PID" 2>/dev/null

# Check for auth/TLS error in logs
if ! grep -qE "(Error|handshake|auth|certificate)" /tmp/goncat-ssl-t5-slave.log && \
   ! grep -qE "(Error|handshake|auth|certificate)" /tmp/goncat-ssl-t5-master.log; then
    echo -e "${RED}✗ Test 5: No error detected (should have auth failure)${NC}"
    exit 1
fi

kill "$MASTER_PID" 2>/dev/null && wait "$MASTER_PID" 2>/dev/null
MASTER_PID=""
SLAVE_PID=""
echo -e "${GREEN}✓ Test 5 PASSED: Mismatched keys properly rejected${NC}"

# Test 6: --key without --ssl → CLI error
echo -e "${YELLOW}Test 6: --key without --ssl (should fail immediately)${NC}"
PORT=$((PORT_BASE + 6))

"$REPO_ROOT/dist/goncat.elf" master listen "${TRANSPORT}://*:${PORT}" --exec /bin/sh --key "$TEST_KEY" > /tmp/goncat-ssl-t6-master.log 2>&1 &
MASTER_PID=$!

sleep 2

# Should fail immediately with error message (may exit 0 but with error logs)
if ! wait "$MASTER_PID" 2>/dev/null; then
    MASTER_EXIT=$?
fi

if grep -qE "(you must use '--ssl' to use '--key'|Argument validation)" /tmp/goncat-ssl-t6-master.log; then
    echo -e "${GREEN}✓ Test 6 PASSED: --key without --ssl properly rejected${NC}"
else
    echo -e "${RED}✗ Test 6: Expected validation error not found${NC}"
    cat /tmp/goncat-ssl-t6-master.log
    exit 1
fi

MASTER_PID=""

echo -e "${GREEN}✓ All SSL/TLS encryption tests PASSED${NC}"
exit 0
