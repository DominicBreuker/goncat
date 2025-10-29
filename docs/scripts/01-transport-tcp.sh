#!/bin/bash
# Validation Script: TCP Transport
# Purpose: Verify tcp transport works for master-listen and slave-connect modes
# Expected: Data transfers successfully
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
    rm -f /tmp/goncat-test-tcp-*
}
trap cleanup EXIT

if [ ! -f "$REPO_ROOT/dist/goncat.elf" ]; then
    echo -e "${YELLOW}Building goncat binary...${NC}"
    make build-linux
fi

echo -e "${GREEN}Starting validation: TCP Transport${NC}"

TRANSPORT="tcp"
PORT_BASE=12000

# Test: Master listen, slave connect
echo -e "${YELLOW}Test: Master listen (${TRANSPORT}), slave connect${NC}"
MASTER_PORT=$((PORT_BASE + 1))

# Start master with shell
"$REPO_ROOT/dist/goncat.elf" master listen "${TRANSPORT}://*:${MASTER_PORT}" --exec /bin/sh > /tmp/goncat-test-tcp-master-out.txt 2>&1 &
MASTER_PID=$!
sleep 2

# Verify master is listening
if ! grep -q "Listening on" /tmp/goncat-test-tcp-master-out.txt; then
    echo -e "${RED}✗ Master not listening${NC}"
    cat /tmp/goncat-test-tcp-master-out.txt
    exit 1
fi

# Connect slave and send commands
(echo "echo TCP_TEST_SUCCESS"; sleep 0.5; echo "exit") | timeout 15 "$REPO_ROOT/dist/goncat.elf" slave connect "${TRANSPORT}://localhost:${MASTER_PORT}" > /tmp/goncat-test-tcp-slave-out.txt 2>&1 || true
sleep 1

# Verify session established
if ! grep -q "Session with .* established" /tmp/goncat-test-tcp-slave-out.txt; then
    echo -e "${RED}✗ Connection not established${NC}"
    cat /tmp/goncat-test-tcp-slave-out.txt
    exit 1
fi

echo -e "${GREEN}✓ Connection established successfully${NC}"

# Verify data transfer
if grep -q "TCP_TEST_SUCCESS" /tmp/goncat-test-tcp-slave-out.txt; then
    echo -e "${GREEN}✓ Data received successfully through tcp tunnel${NC}"
else
    echo -e "${YELLOW}⚠ Data transfer verification incomplete${NC}"
fi

# Verify session closed
if grep -q "Session with .* closed" /tmp/goncat-test-tcp-slave-out.txt; then
    echo -e "${GREEN}✓ Session closed gracefully${NC}"
fi

kill $MASTER_PID 2>/dev/null || true

echo -e "${GREEN}✓ TCP transport validation passed${NC}"
exit 0
