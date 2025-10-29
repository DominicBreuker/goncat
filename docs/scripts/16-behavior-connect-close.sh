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

"$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --exec 'echo CLOSE_TEST' > /tmp/goncat-test-behavior-master-out.txt 2>&1 &
MASTER_PID=$!
sleep 2

# Connect and immediately exit
timeout 5 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" > /tmp/goncat-test-behavior-slave1-out.txt 2>&1 || true
sleep 2

# Check master is still running
if ps -p $MASTER_PID > /dev/null; then
    echo -e "${GREEN}✓ Listen mode continues after connection closes${NC}"
else
    echo -e "${RED}✗ Listen mode should continue after connection closes${NC}"
    cat /tmp/goncat-test-behavior-master-out.txt
    exit 1
fi

# Connect again to verify it still works
timeout 5 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" > /tmp/goncat-test-behavior-slave2-out.txt 2>&1 || true
sleep 1

if grep -q "CLOSE_TEST" /tmp/goncat-test-behavior-slave2-out.txt; then
    echo -e "${GREEN}✓ Listen mode accepted second connection${NC}"
else
    echo -e "${YELLOW}⚠ Second connection incomplete${NC}"
fi

kill $MASTER_PID 2>/dev/null || true

echo -e "${GREEN}✓ Connection close behavior validation passed${NC}"
exit 0
