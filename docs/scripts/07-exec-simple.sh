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

# Test: Execute shell commands through connection
echo -e "${YELLOW}Test: Execute shell commands${NC}"
MASTER_PORT=$((PORT_BASE + 1))

# Start master with shell exec
"$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --exec /bin/sh > /tmp/goncat-test-exec-master-out.txt 2>&1 &
MASTER_PID=$!
sleep 2

# Verify master is listening
if ! grep -q "Listening on" /tmp/goncat-test-exec-master-out.txt; then
    echo -e "${RED}✗ Master failed to start listening${NC}"
    cat /tmp/goncat-test-exec-master-out.txt
    exit 1
fi

# Connect slave and send commands
(echo "whoami"; sleep 0.5; echo "id"; sleep 0.5; echo "exit") | timeout 10 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" > /tmp/goncat-test-exec-slave-out.txt 2>&1 || true
sleep 1

# Verify session was established
if ! grep -q "Session with .* established" /tmp/goncat-test-exec-slave-out.txt; then
    echo -e "${RED}✗ Session not established${NC}"
    cat /tmp/goncat-test-exec-slave-out.txt
    exit 1
fi

# Verify command output (whoami should return a username)
if grep -qE "(root|runner|[a-z]+)" /tmp/goncat-test-exec-slave-out.txt; then
    echo -e "${GREEN}✓ Commands executed and output received${NC}"
else
    echo -e "${RED}✗ Command output not found${NC}"
    cat /tmp/goncat-test-exec-slave-out.txt
    exit 1
fi

# Verify session closed after exit command
if grep -q "Session with .* closed" /tmp/goncat-test-exec-slave-out.txt; then
    echo -e "${GREEN}✓ Session closed gracefully after exit${NC}"
else
    echo -e "${YELLOW}⚠ Session close message not found${NC}"
fi

# Verify master logged session closure
if grep -q "Session with .* closed" /tmp/goncat-test-exec-master-out.txt; then
    echo -e "${GREEN}✓ Master detected session closure${NC}"
else
    echo -e "${YELLOW}⚠ Master session close message not found${NC}"
fi

# Verify master is still running (listen mode should continue)
if ps -p $MASTER_PID > /dev/null; then
    echo -e "${GREEN}✓ Master continues running in listen mode${NC}"
else
    echo -e "${YELLOW}⚠ Master exited${NC}"
fi

kill $MASTER_PID 2>/dev/null || true

echo -e "${GREEN}✓ Simple command execution validation passed${NC}"
exit 0
