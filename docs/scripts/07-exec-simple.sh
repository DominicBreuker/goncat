#!/bin/bash
# Validation Script: Simple Command Execution
# Purpose: Verify --exec flag executes commands without PTY
# Expected: Commands execute and output is received correctly
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
    rm -f /tmp/goncat-test-exec-*
}
trap cleanup EXIT

if [ ! -f "$REPO_ROOT/dist/goncat.elf" ]; then
    echo -e "${YELLOW}Building goncat binary...${NC}"
    make build-linux
fi

echo -e "${GREEN}Starting validation: Simple Command Execution${NC}"

PORT_BASE=12060

# Test: Execute simple echo command
echo -e "${YELLOW}Test: Execute echo command${NC}"
MASTER_PORT=$((PORT_BASE + 1))

"$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --exec 'echo EXEC_TEST_SUCCESS' > /tmp/goncat-test-exec-master-out.txt 2>&1 &
MASTER_PID=$!
sleep 2

timeout 10 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" > /tmp/goncat-test-exec-slave-out.txt 2>&1 || true
sleep 1

if grep -q "EXEC_TEST_SUCCESS" /tmp/goncat-test-exec-slave-out.txt; then
    echo -e "${GREEN}✓ Command execution works${NC}"
else
    echo -e "${RED}✗ Command execution failed${NC}"
    cat /tmp/goncat-test-exec-slave-out.txt
    exit 1
fi

kill $MASTER_PID 2>/dev/null || true

echo -e "${GREEN}✓ Simple command execution validation passed${NC}"
exit 0
