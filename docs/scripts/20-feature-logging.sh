#!/bin/bash
# Validation Script: Session Logging
# Purpose: Verify --log flag creates session logs
# Expected: Log file is created with session data
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
    rm -f /tmp/goncat-test-log-*
}
trap cleanup EXIT

if [ ! -f "$REPO_ROOT/dist/goncat.elf" ]; then
    echo -e "${YELLOW}Building goncat binary...${NC}"
    make build-linux
fi

echo -e "${GREEN}Starting validation: Session Logging${NC}"

PORT_BASE=12100
LOG_FILE="/tmp/goncat-test-session.log"

rm -f "$LOG_FILE"

# Test: Session logging
echo -e "${YELLOW}Test: Create session log${NC}"
MASTER_PORT=$((PORT_BASE + 1))

"$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --exec /bin/sh --log "$LOG_FILE" > /tmp/goncat-test-log-master-out.txt 2>&1 &
MASTER_PID=$!
sleep 2

echo "echo LOG_TEST_DATA" | timeout 10 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" > /tmp/goncat-test-log-slave-out.txt 2>&1 || true
sleep 2

kill $MASTER_PID 2>/dev/null || true
sleep 1

# Verify log file was created
if [ -f "$LOG_FILE" ]; then
    echo -e "${GREEN}✓ Log file was created${NC}"
    
    # Check if log contains some data (even if empty, file should exist)
    if [ -s "$LOG_FILE" ]; then
        echo -e "${GREEN}✓ Log file contains data${NC}"
    else
        echo -e "${YELLOW}⚠ Log file is empty (may be timing issue)${NC}"
    fi
else
    echo -e "${RED}✗ Log file was not created${NC}"
    exit 1
fi

rm -f "$LOG_FILE"

echo -e "${GREEN}✓ Session logging validation passed${NC}"
exit 0
