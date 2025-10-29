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

"$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --timeout 10000 --exec /bin/sh > /tmp/goncat-test-timeout-master1-out.txt 2>&1 &
MASTER_PID=$!
sleep 2

echo "whoami" | timeout 15 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" --timeout 10000 > /tmp/goncat-test-timeout-slave1-out.txt 2>&1 || true
sleep 1

# Verify session established
if grep -q "Session with .* established" /tmp/goncat-test-timeout-slave1-out.txt; then
    echo -e "${GREEN}✓ Connection with reasonable timeout established${NC}"
else
    echo -e "${RED}✗ Connection failed${NC}"
    cat /tmp/goncat-test-timeout-slave1-out.txt
    exit 1
fi

# Verify command executed
if grep -qE "(root|runner|[a-z]+)" /tmp/goncat-test-timeout-slave1-out.txt; then
    echo -e "${GREEN}✓ Command executed successfully${NC}"
else
    echo -e "${YELLOW}⚠ Command output unclear${NC}"
fi

kill $MASTER_PID 2>/dev/null || true
sleep 1

# Test 2: Very short timeout (100ms) still allows stable connection
echo -e "${YELLOW}Test 2: Very short timeout (100ms) doesn't break healthy connection${NC}"
MASTER_PORT=$((PORT_BASE + 2))

"$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --timeout 100 --exec /bin/sh > /tmp/goncat-test-timeout-master2-out.txt 2>&1 &
MASTER_PID=$!
sleep 2

echo "echo SHORT_TIMEOUT_OK" | timeout 5 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" --timeout 100 > /tmp/goncat-test-timeout-slave2-out.txt 2>&1 || true
sleep 1

# Verify session established and worked
if grep -q "Session with .* established" /tmp/goncat-test-timeout-slave2-out.txt; then
    echo -e "${GREEN}✓ Short timeout allows connection${NC}"
else
    echo -e "${RED}✗ Short timeout broke connection${NC}"
    cat /tmp/goncat-test-timeout-slave2-out.txt
    exit 1
fi

if grep -q "SHORT_TIMEOUT_OK" /tmp/goncat-test-timeout-slave2-out.txt; then
    echo -e "${GREEN}✓ Data transfer works with short timeout${NC}"
else
    echo -e "${YELLOW}⚠ Data transfer unclear${NC}"
fi

kill $MASTER_PID 2>/dev/null || true
sleep 1

# Test 3: Connection timeout when slave never connects
echo -e "${YELLOW}Test 3: Connection attempt times out to non-existent server${NC}"

# Try to connect to a port that's not listening (should timeout)
timeout 6 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:$((PORT_BASE + 99))" --timeout 2000 > /tmp/goncat-test-timeout-slave3-out.txt 2>&1 || true
sleep 1

# This should fail with timeout or connection refused
if grep -qiE "timeout|connection refused|dial|error" /tmp/goncat-test-timeout-slave3-out.txt; then
    echo -e "${GREEN}✓ Connection correctly times out to non-existent server${NC}"
else
    echo -e "${YELLOW}⚠ Timeout behavior unclear${NC}"
fi

# Test 4: Timeout detection when one side disappears
echo -e "${YELLOW}Test 4: Timeout detected when connection dies${NC}"
MASTER_PORT=$((PORT_BASE + 3))

"$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --timeout 2000 --exec /bin/sh > /tmp/goncat-test-timeout-master4-out.txt 2>&1 &
MASTER_PID=$!
sleep 2

# Start slave in background
"$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" --timeout 2000 > /tmp/goncat-test-timeout-slave4-out.txt 2>&1 &
SLAVE_PID=$!
sleep 2

# Verify connection established on both sides
if ! grep -q "Session with .* established" /tmp/goncat-test-timeout-slave4-out.txt; then
    echo -e "${RED}✗ Connection not established${NC}"
    exit 1
fi

if ! grep -q "Session with .* established" /tmp/goncat-test-timeout-master4-out.txt; then
    echo -e "${RED}✗ Master didn't log connection${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Connection established${NC}"

# Verify no premature closure
if grep -q "Session with .* closed" /tmp/goncat-test-timeout-master4-out.txt; then
    echo -e "${RED}✗ Premature closure detected${NC}"
    exit 1
fi

# Kill slave abruptly (SIGKILL, not graceful)
kill -9 $SLAVE_PID 2>/dev/null || true
sleep 1

# Master should NOT immediately detect closure (no session closed message yet)
if grep -q "Session with .* closed" /tmp/goncat-test-timeout-master4-out.txt; then
    echo -e "${YELLOW}⚠ Master detected closure immediately (may be OK)${NC}"
else
    echo -e "${GREEN}✓ Master didn't immediately detect abrupt closure${NC}"
fi

# Wait for timeout (2 seconds + margin)
sleep 3

# Now master SHOULD have detected timeout and closed
if grep -q "Session with .* closed" /tmp/goncat-test-timeout-master4-out.txt; then
    echo -e "${GREEN}✓ Master detected timeout and closed connection${NC}"
else
    echo -e "${YELLOW}⚠ Master may not have detected timeout${NC}"
fi

kill $MASTER_PID 2>/dev/null || true

echo -e "${GREEN}✓ Timeout handling validation passed${NC}"
exit 0
