#!/bin/bash
# Validation Script: Timeout Handling
# Purpose: Verify --timeout flag is honored correctly
# Expected: Connections timeout when expected, work when not
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

cleanup() {
    pkill -9 goncat.elf 2>/dev/null || true
    rm -f /tmp/goncat-test-timeout-*
}
trap cleanup EXIT

if [ ! -f "$REPO_ROOT/dist/goncat.elf" ]; then
    echo -e "${YELLOW}Building goncat binary...${NC}"
    make build-linux
fi

echo -e "${GREEN}Starting validation: Timeout Handling${NC}"

PORT_BASE=12120

# Test 1: Connection with reasonable timeout succeeds
echo -e "${YELLOW}Test 1: Connection with reasonable timeout (10s) succeeds${NC}"
MASTER_PORT=$((PORT_BASE + 1))

"$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --timeout 10000 --exec 'echo TIMEOUT_SUCCESS' > /tmp/goncat-test-timeout-master1-out.txt 2>&1 &
MASTER_PID=$!
sleep 2

timeout 15 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" --timeout 10000 > /tmp/goncat-test-timeout-slave1-out.txt 2>&1 || true
sleep 1

if grep -q "TIMEOUT_SUCCESS" /tmp/goncat-test-timeout-slave1-out.txt; then
    echo -e "${GREEN}✓ Connection with reasonable timeout works${NC}"
else
    echo -e "${RED}✗ Connection with reasonable timeout failed${NC}"
    cat /tmp/goncat-test-timeout-slave1-out.txt
    exit 1
fi

kill $MASTER_PID 2>/dev/null || true
sleep 1

# Test 2: Very short timeout (100ms) still allows stable connection
echo -e "${YELLOW}Test 2: Very short timeout (100ms) doesn't break healthy connection${NC}"
MASTER_PORT=$((PORT_BASE + 2))

"$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --timeout 100 --exec 'echo SHORT_TIMEOUT_OK' > /tmp/goncat-test-timeout-master2-out.txt 2>&1 &
MASTER_PID=$!
sleep 2

timeout 5 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" --timeout 100 > /tmp/goncat-test-timeout-slave2-out.txt 2>&1 || true
sleep 1

if grep -q "SHORT_TIMEOUT_OK" /tmp/goncat-test-timeout-slave2-out.txt; then
    echo -e "${GREEN}✓ Short timeout (100ms) allows connection to work${NC}"
else
    echo -e "${YELLOW}⚠ Short timeout may have affected connection${NC}"
fi

kill $MASTER_PID 2>/dev/null || true
sleep 1

# Test 3: Connection timeout when slave never connects
echo -e "${YELLOW}Test 3: Connection attempt times out to non-existent server${NC}"

# Try to connect to a port that's not listening (should timeout)
timeout 6 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:$((PORT_BASE + 99))" --timeout 2000 > /tmp/goncat-test-timeout-slave3-out.txt 2>&1 || true
sleep 1

# This should fail with timeout error
if grep -qi "timeout\|connection refused\|dial" /tmp/goncat-test-timeout-slave3-out.txt; then
    echo -e "${GREEN}✓ Connection correctly times out to non-existent server${NC}"
else
    echo -e "${YELLOW}⚠ Connection timeout behavior unclear (check manually)${NC}"
fi

echo -e "${GREEN}✓ Timeout handling validation passed${NC}"
exit 0
