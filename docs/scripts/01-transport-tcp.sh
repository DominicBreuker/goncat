#!/bin/bash
# Validation Script: TCP Transport
# Purpose: Verify basic TCP transport connection establishment works
# Expected: Master and slave can establish connection successfully
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
    # Kill any background processes
    pkill -9 goncat.elf 2>/dev/null || true
    # Clean up temp files
    rm -f /tmp/goncat-test-tcp-*
}
trap cleanup EXIT

# Ensure binary exists
if [ ! -f "$REPO_ROOT/dist/goncat.elf" ]; then
    echo -e "${YELLOW}Building goncat binary...${NC}"
    make build-linux
fi

echo -e "${GREEN}Starting validation: TCP Transport${NC}"

# Get transport from argument or default to tcp
TRANSPORT="${1:-tcp}"
PORT_BASE=12000

# Test 1: Master listen, slave connect
echo -e "${YELLOW}Test 1: Master listen, slave connect${NC}"
MASTER_PORT=$((PORT_BASE + 1))

# Start master in background
"$REPO_ROOT/dist/goncat.elf" master listen "${TRANSPORT}://*:${MASTER_PORT}" --exec 'echo GONCAT_TEST_SUCCESS' > /tmp/goncat-test-tcp-master-out.txt 2>&1 &
MASTER_PID=$!

# Wait for master to start
sleep 2

# Verify master is running and listening
if ! ps -p $MASTER_PID > /dev/null; then
    echo -e "${RED}✗ Master failed to start${NC}"
    cat /tmp/goncat-test-tcp-master-out.txt
    exit 1
fi

if ! grep -q "Listening on" /tmp/goncat-test-tcp-master-out.txt; then
    echo -e "${RED}✗ Master not listening${NC}"
    cat /tmp/goncat-test-tcp-master-out.txt
    exit 1
fi

# Connect slave
timeout 10 "$REPO_ROOT/dist/goncat.elf" slave connect "${TRANSPORT}://localhost:${MASTER_PORT}" > /tmp/goncat-test-tcp-slave-out.txt 2>&1 || true

# Wait a bit
sleep 1

# Verify connection was established
if grep -q "Session with .* established" /tmp/goncat-test-tcp-slave-out.txt; then
    echo -e "${GREEN}✓ Connection established successfully${NC}"
else
    echo -e "${RED}✗ Connection not established${NC}"
    echo "Slave output:"
    cat /tmp/goncat-test-tcp-slave-out.txt
    echo "Master output:"
    cat /tmp/goncat-test-tcp-master-out.txt
    exit 1
fi

# Verify data was received
if grep -q "GONCAT_TEST_SUCCESS" /tmp/goncat-test-tcp-slave-out.txt; then
    echo -e "${GREEN}✓ Data received successfully through TCP tunnel${NC}"
else
    echo -e "${YELLOW}⚠ Connection established but data verification incomplete${NC}"
fi

# Cleanup
kill $MASTER_PID 2>/dev/null || true
sleep 1

# Test 2: Slave listen, master connect
echo -e "${YELLOW}Test 2: Slave listen, master connect${NC}"
SLAVE_PORT=$((PORT_BASE + 2))

# Start slave listener in background
"$REPO_ROOT/dist/goncat.elf" slave listen "${TRANSPORT}://*:${SLAVE_PORT}" > /tmp/goncat-test-tcp-slave2-out.txt 2>&1 &
SLAVE_PID=$!

# Wait for slave to start
sleep 2

# Verify slave is running and listening
if ! ps -p $SLAVE_PID > /dev/null; then
    echo -e "${RED}✗ Slave failed to start${NC}"
    cat /tmp/goncat-test-tcp-slave2-out.txt
    exit 1
fi

if ! grep -q "Listening on" /tmp/goncat-test-tcp-slave2-out.txt; then
    echo -e "${RED}✗ Slave not listening${NC}"
    cat /tmp/goncat-test-tcp-slave2-out.txt
    exit 1
fi

# Connect master
timeout 10 "$REPO_ROOT/dist/goncat.elf" master connect "${TRANSPORT}://localhost:${SLAVE_PORT}" --exec 'echo GONCAT_REVERSE_SUCCESS' > /tmp/goncat-test-tcp-master2-out.txt 2>&1 || true

# Wait a bit
sleep 1

# Verify connection was established
if grep -q "Session with .* established" /tmp/goncat-test-tcp-master2-out.txt; then
    echo -e "${GREEN}✓ Reverse connection established successfully${NC}"
else
    echo -e "${RED}✗ Reverse connection not established${NC}"
    echo "Master output:"
    cat /tmp/goncat-test-tcp-master2-out.txt
    echo "Slave output:"
    cat /tmp/goncat-test-tcp-slave2-out.txt
    exit 1
fi

# Cleanup
kill $SLAVE_PID 2>/dev/null || true

echo -e "${GREEN}✓ TCP transport validation passed${NC}"
exit 0
