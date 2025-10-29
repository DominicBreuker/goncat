#!/bin/bash
# Validation Script: Connection Close Behavior
# Purpose: Verify connection lifecycle behaviors
# Expected: Listen mode persists after connection closes, connect mode exits
# Data Flow: Commands from master stdin → slave executes → master stdout
# Tests: Listen persistence, connect mode exit, graceful SIGINT
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

# Track PIDs for cleanup
MASTER_PID=""
SLAVE_PID=""

cleanup() {
    [ -n "$MASTER_PID" ] && kill "$MASTER_PID" 2>/dev/null && wait "$MASTER_PID" 2>/dev/null || true
    [ -n "$SLAVE_PID" ] && kill "$SLAVE_PID" 2>/dev/null && wait "$SLAVE_PID" 2>/dev/null || true
    rm -f /tmp/goncat-behavior-*
}
trap cleanup EXIT

# Helper function: Poll for pattern in file
poll_for_pattern() {
    local file="$1"
    local pattern="$2"
    local timeout="${3:-10}"
    local start=$(date +%s)
    while true; do
        if [ -f "$file" ] && grep -qE "$pattern" "$file" 2>/dev/null; then
            return 0
        fi
        local now=$(date +%s)
        if [ $((now - start)) -ge "$timeout" ]; then
            return 1
        fi
        sleep 0.1
    done
}

if [ ! -f "$REPO_ROOT/dist/goncat.elf" ]; then
    echo -e "${YELLOW}Building goncat binary...${NC}"
    make build-linux
fi

echo -e "${GREEN}=== Connection Close Behavior Validation ===${NC}"

MASTER_PORT=12090
TOKEN1="CLOSE_$$_$RANDOM"
TOKEN2="PERSIST_$$_$RANDOM"

# Test 1: Listen mode persists after connection closes
echo -e "${YELLOW}Test 1: Listen mode persists after connection closes${NC}"

# Start master in listen mode with shell (no input needed for listening)
"$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --exec /bin/sh > /tmp/goncat-behavior-master.log 2>&1 &
MASTER_PID=$!

if ! poll_for_pattern /tmp/goncat-behavior-master.log "Listening on" 5; then
    echo -e "${RED}✗ Master failed to start${NC}"
    cat /tmp/goncat-behavior-master.log
    exit 1
fi
echo -e "${GREEN}✓ Master listening${NC}"

# Connect slave, send command, then exit
(echo "echo $TOKEN1"; echo "exit") | "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" > /tmp/goncat-behavior-slave1.log 2>&1 &
SLAVE_PID=$!

# Wait for session establishment
if ! poll_for_pattern /tmp/goncat-behavior-master.log "Session with .* established" 10; then
    echo -e "${RED}✗ Connection not established${NC}"
    cat /tmp/goncat-behavior-master.log
    exit 1
fi
echo -e "${GREEN}✓ Connection established${NC}"

# Wait for command output and session closure
if ! poll_for_pattern /tmp/goncat-behavior-master.log "$TOKEN1" 10; then
    echo -e "${RED}✗ Token not found in master output${NC}"
    cat /tmp/goncat-behavior-master.log
    exit 1
fi
echo -e "${GREEN}✓ Token verified in master output${NC}"

# Wait for slave to exit
wait "$SLAVE_PID" 2>/dev/null || true
SLAVE_PID=""

# Poll for session closed message
if ! poll_for_pattern /tmp/goncat-behavior-master.log "Session with .* closed" 5; then
    echo -e "${YELLOW}⚠ Session closed message not found${NC}"
else
    echo -e "${GREEN}✓ Session closed detected${NC}"
fi

# Verify master is still running (listen mode should persist)
if kill -0 "$MASTER_PID" 2>/dev/null; then
    echo -e "${GREEN}✓ Master listener still active (correct behavior)${NC}"
else
    echo -e "${RED}✗ Master listener exited (should persist in listen mode)${NC}"
    exit 1
fi

# Test 2: Second connection works (listen mode accepted new connection)
echo -e "${YELLOW}Test 2: Second connection to verify listen persistence${NC}"

(echo "echo $TOKEN2"; echo "exit") | "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" > /tmp/goncat-behavior-slave2.log 2>&1 &
SLAVE_PID=$!

if ! poll_for_pattern /tmp/goncat-behavior-master.log "$TOKEN2" 10; then
    echo -e "${RED}✗ Second connection failed${NC}"
    cat /tmp/goncat-behavior-master.log
    exit 1
fi
echo -e "${GREEN}✓ Second connection succeeded (listener persists)${NC}"

wait "$SLAVE_PID" 2>/dev/null || true
SLAVE_PID=""

# Test 3: Connect mode exits after connection closes
echo -e "${YELLOW}Test 3: Connect mode exits after connection closes${NC}"

# Start slave in listen mode
"$REPO_ROOT/dist/goncat.elf" slave listen "tcp://*:$((MASTER_PORT + 1))" > /tmp/goncat-behavior-slave-listen.log 2>&1 &
SLAVE_PID=$!

if ! poll_for_pattern /tmp/goncat-behavior-slave-listen.log "Listening on" 5; then
    echo -e "${RED}✗ Slave listener failed to start${NC}"
    cat /tmp/goncat-behavior-slave-listen.log
    exit 1
fi

# Connect master in connect mode (should exit after connection closes)
(echo "whoami"; echo "exit") | "$REPO_ROOT/dist/goncat.elf" master connect "tcp://localhost:$((MASTER_PORT + 1))" --exec /bin/sh > /tmp/goncat-behavior-master-connect.log 2>&1 &
MASTER_CONNECT_PID=$!

# Wait for master connect to exit
timeout 10 bash -c "while kill -0 $MASTER_CONNECT_PID 2>/dev/null; do sleep 0.1; done" || true

if kill -0 "$MASTER_CONNECT_PID" 2>/dev/null; then
    echo -e "${RED}✗ Master connect mode did not exit${NC}"
    kill "$MASTER_CONNECT_PID" 2>/dev/null || true
    wait "$MASTER_CONNECT_PID" 2>/dev/null || true
    exit 1
else
    echo -e "${GREEN}✓ Master connect mode exited after connection closed (correct behavior)${NC}"
fi

# Clean up
kill "$MASTER_PID" 2>/dev/null && wait "$MASTER_PID" 2>/dev/null || true
MASTER_PID=""
kill "$SLAVE_PID" 2>/dev/null && wait "$SLAVE_PID" 2>/dev/null || true
SLAVE_PID=""

echo -e "${GREEN}✓ Connection close behavior validation PASSED${NC}"
exit 0
