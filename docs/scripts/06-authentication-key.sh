#!/bin/bash
# Validation Script: Mutual Authentication with --key
# Purpose: Verify --key flag with mutual TLS authentication
# Test Matrix:
#   1. --ssl --key matching → success + data token verification
#   2. --ssl with wrong --key → auth failure
#   3. --key without --ssl → CLI error
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
    rm -f /tmp/goncat-auth-*
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

echo -e "${GREEN}=== Mutual Authentication Validation ===${NC}"

TRANSPORT="tcp"
PORT_BASE=12030
TEST_KEY="auth_password_456"

# Test 1: --ssl --key matching → success + data verification
echo -e "${YELLOW}Test 1: --ssl --key matching (should succeed)${NC}"
PORT=$((PORT_BASE + 1))
TOKEN="AUTH_MATCH_$$_$RANDOM"

(echo "echo $TOKEN"; sleep 1; echo "exit") | "$REPO_ROOT/dist/goncat.elf" master listen "${TRANSPORT}://*:${PORT}" --exec /bin/sh --ssl --key "$TEST_KEY" > /tmp/goncat-auth-t1-master.log 2>&1 &
MASTER_PID=$!

if ! poll_for_pattern /tmp/goncat-auth-t1-master.log "Listening on" 5; then
    echo -e "${RED}✗ Test 1: Master failed to start${NC}"
    exit 1
fi

"$REPO_ROOT/dist/goncat.elf" slave connect "${TRANSPORT}://localhost:${PORT}" --ssl --key "$TEST_KEY" > /tmp/goncat-auth-t1-slave.log 2>&1 &
SLAVE_PID=$!

if ! poll_for_pattern /tmp/goncat-auth-t1-master.log "Session with .* established" 10; then
    echo -e "${RED}✗ Test 1: Connection not established${NC}"
    cat /tmp/goncat-auth-t1-master.log
    exit 1
fi

# Verify data token in master output
if ! poll_for_pattern /tmp/goncat-auth-t1-master.log "$TOKEN" 10; then
    echo -e "${RED}✗ Test 1: Data token not found (mTLS may not be protecting data channel)${NC}"
    exit 1
fi

if ! poll_for_pattern /tmp/goncat-auth-t1-master.log "Session with .* closed" 5; then
    echo -e "${RED}✗ Test 1: Session not closed properly${NC}"
    exit 1
fi

wait "$SLAVE_PID" 2>/dev/null
kill "$MASTER_PID" 2>/dev/null && wait "$MASTER_PID" 2>/dev/null
MASTER_PID=""
SLAVE_PID=""
echo -e "${GREEN}✓ Test 1 PASSED: Mutual TLS authentication working${NC}"

# Test 2: --ssl with mismatched --key → auth failure
echo -e "${YELLOW}Test 2: --ssl with mismatched keys (should fail)${NC}"
PORT=$((PORT_BASE + 2))

"$REPO_ROOT/dist/goncat.elf" master listen "${TRANSPORT}://*:${PORT}" --exec /bin/sh --ssl --key "$TEST_KEY" > /tmp/goncat-auth-t2-master.log 2>&1 &
MASTER_PID=$!

if ! poll_for_pattern /tmp/goncat-auth-t2-master.log "Listening on" 5; then
    echo -e "${RED}✗ Test 2: Master failed to start${NC}"
    exit 1
fi

"$REPO_ROOT/dist/goncat.elf" slave connect "${TRANSPORT}://localhost:${PORT}" --ssl --key "WRONG_PASSWORD" > /tmp/goncat-auth-t2-slave.log 2>&1 &
SLAVE_PID=$!

# Should NOT establish connection (mutual TLS auth failure)
sleep 3
if grep -qE "Session with .* established" /tmp/goncat-auth-t2-master.log; then
    echo -e "${RED}✗ Test 2: Connection established (should have failed auth)${NC}"
    exit 1
fi

wait "$SLAVE_PID" 2>/dev/null

# Check for auth/TLS error in logs
if ! grep -qE "(Error|handshake|auth|certificate)" /tmp/goncat-auth-t2-slave.log && \
   ! grep -qE "(Error|handshake|auth|certificate)" /tmp/goncat-auth-t2-master.log; then
    echo -e "${RED}✗ Test 2: No error detected (should have auth failure)${NC}"
    exit 1
fi

kill "$MASTER_PID" 2>/dev/null && wait "$MASTER_PID" 2>/dev/null
MASTER_PID=""
SLAVE_PID=""
echo -e "${GREEN}✓ Test 2 PASSED: Mismatched keys properly rejected${NC}"

# Test 3: --key without --ssl → CLI error
echo -e "${YELLOW}Test 3: --key without --ssl (should fail immediately)${NC}"
PORT=$((PORT_BASE + 3))

"$REPO_ROOT/dist/goncat.elf" master listen "${TRANSPORT}://*:${PORT}" --exec /bin/sh --key "$TEST_KEY" > /tmp/goncat-auth-t3-master.log 2>&1 &
MASTER_PID=$!

sleep 2

# Should fail immediately with validation error
wait "$MASTER_PID" 2>/dev/null

if grep -qE "(you must use '--ssl' to use '--key'|Argument validation)" /tmp/goncat-auth-t3-master.log; then
    echo -e "${GREEN}✓ Test 3 PASSED: --key without --ssl properly rejected${NC}"
else
    echo -e "${RED}✗ Test 3: Expected validation error not found${NC}"
    cat /tmp/goncat-auth-t3-master.log
    exit 1
fi

MASTER_PID=""

echo -e "${GREEN}✓ All mutual authentication tests PASSED${NC}"
exit 0
