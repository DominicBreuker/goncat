#!/bin/bash
# Validation Script: Connection Close Behavior
# Purpose: Verify connection lifecycle behaviors
# Expected: Connect mode exits on close, listen mode continues
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
    rm -f /tmp/goncat-test-behavior-*
}
trap cleanup EXIT

if [ ! -f "$REPO_ROOT/dist/goncat.elf" ]; then
    echo -e "${YELLOW}Building goncat binary...${NC}"
    make build-linux
fi

echo -e "${GREEN}Starting validation: Connection Close Behavior${NC}"

PORT_BASE=12090

# Test 1: Listen mode continues after connection closes
echo -e "${YELLOW}Test 1: Listen mode continues after connection closes${NC}"
MASTER_PORT=$((PORT_BASE + 1))

# Start master with shell
"$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --exec /bin/sh > /tmp/goncat-test-behavior-master-out.txt 2>&1 &
MASTER_PID=$!
sleep 2

# Verify master is listening
if ! grep -q "Listening on" /tmp/goncat-test-behavior-master-out.txt; then
    echo -e "${RED}✗ Master not listening${NC}"
    cat /tmp/goncat-test-behavior-master-out.txt
    exit 1
fi

# Connect slave and send exit command to close connection
echo "exit" | timeout 5 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" > /tmp/goncat-test-behavior-slave1-out.txt 2>&1 || true
sleep 1

# Verify session was established and closed on slave side
if ! grep -q "Session with .* established" /tmp/goncat-test-behavior-slave1-out.txt; then
    echo -e "${RED}✗ First connection not established${NC}"
    cat /tmp/goncat-test-behavior-slave1-out.txt
    exit 1
fi

if ! grep -q "Session with .* closed" /tmp/goncat-test-behavior-slave1-out.txt; then
    echo -e "${YELLOW}⚠ Session close message not found on slave${NC}"
fi

# Verify master logged the connection and closure
if ! grep -q "Session with .* established" /tmp/goncat-test-behavior-master-out.txt; then
    echo -e "${RED}✗ Master didn't log session establishment${NC}"
    exit 1
fi

if ! grep -q "Session with .* closed" /tmp/goncat-test-behavior-master-out.txt; then
    echo -e "${RED}✗ Master didn't log session closure${NC}"
    exit 1
fi

# Check master is still running
if ! ps -p $MASTER_PID > /dev/null; then
    echo -e "${RED}✗ Listen mode should continue after connection closes${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Listen mode continues after connection closes${NC}"

# Test 2: Connect again to verify it still works
echo -e "${YELLOW}Test 2: Second connection works${NC}"
echo "whoami" | timeout 5 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" > /tmp/goncat-test-behavior-slave2-out.txt 2>&1 || true
sleep 1

# Verify second connection established
if grep -q "Session with .* established" /tmp/goncat-test-behavior-slave2-out.txt; then
    echo -e "${GREEN}✓ Listen mode accepted second connection${NC}"
else
    echo -e "${RED}✗ Second connection failed${NC}"
    cat /tmp/goncat-test-behavior-slave2-out.txt
    exit 1
fi

# Test 3: Slave shutdown via SIGINT
echo -e "${YELLOW}Test 3: Slave graceful shutdown${NC}"

# Start a new slave connection in background
"$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" > /tmp/goncat-test-behavior-slave3-out.txt 2>&1 &
SLAVE_PID=$!
sleep 2

# Verify connection established
if ! grep -q "Session with .* established" /tmp/goncat-test-behavior-slave3-out.txt; then
    echo -e "${RED}✗ Third connection not established${NC}"
    exit 1
fi

# Send SIGINT to slave
kill -INT $SLAVE_PID 2>/dev/null || true
sleep 2

# Verify slave exited
if ps -p $SLAVE_PID > /dev/null 2>&1; then
    echo -e "${RED}✗ Slave should have exited after SIGINT${NC}"
    kill -9 $SLAVE_PID 2>/dev/null || true
    exit 1
fi

echo -e "${GREEN}✓ Slave shut down gracefully${NC}"

# Verify master detected the closure
sleep 1
if tail -5 /tmp/goncat-test-behavior-master-out.txt | grep -q "Session with .* closed"; then
    echo -e "${GREEN}✓ Master detected slave shutdown${NC}"
else
    echo -e "${YELLOW}⚠ Master may not have logged the closure${NC}"
fi

kill $MASTER_PID 2>/dev/null || true

echo -e "${GREEN}✓ Connection close behavior validation passed${NC}"
exit 0
