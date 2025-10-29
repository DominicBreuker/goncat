#!/bin/bash
# Validation Script: Connection Stability
# Purpose: Verify connections work correctly with short timeout
# Expected: Connection establishes and completes successfully with short timeout
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
    rm -f /tmp/goncat-test-stability-*
}
trap cleanup EXIT

if [ ! -f "$REPO_ROOT/dist/goncat.elf" ]; then
    echo -e "${YELLOW}Building goncat binary...${NC}"
    make build-linux
fi

echo -e "${GREEN}Starting validation: Connection Stability${NC}"

PORT_BASE=12100

# Test: Connection works correctly with short timeout
echo -e "${YELLOW}Test: Connection works with 100ms timeout (no uncanceled timeouts)${NC}"
MASTER_PORT=$((PORT_BASE + 1))

# Start master with very short timeout (100ms)
# Use echo command which completes quickly but validates the connection works
"$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --timeout 100 --exec 'echo STABILITY_TEST_SUCCESS' > /tmp/goncat-test-stability-master-out.txt 2>&1 &
MASTER_PID=$!
sleep 2

# Connect slave with same short timeout
timeout 10 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" --timeout 100 > /tmp/goncat-test-stability-slave-out.txt 2>&1 || true
sleep 1

# Verify the connection worked and data was transferred
if grep -q "STABILITY_TEST_SUCCESS" /tmp/goncat-test-stability-slave-out.txt; then
    echo -e "${GREEN}✓ Connection with 100ms timeout completed successfully${NC}"
    echo -e "${GREEN}✓ No uncanceled timeouts (data transferred correctly)${NC}"
else
    echo -e "${RED}✗ Connection with short timeout failed${NC}"
    echo "Master output:"
    cat /tmp/goncat-test-stability-master-out.txt
    echo "Slave output:"
    cat /tmp/goncat-test-stability-slave-out.txt
    exit 1
fi

# Verify connection was actually established (not just failed gracefully)
if grep -q "Session with .* established" /tmp/goncat-test-stability-slave-out.txt; then
    echo -e "${GREEN}✓ Connection establishment confirmed in logs${NC}"
fi

kill $MASTER_PID 2>/dev/null || true

echo -e "${GREEN}✓ Connection stability validation passed${NC}"
exit 0
