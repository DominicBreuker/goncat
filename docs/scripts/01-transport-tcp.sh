#!/bin/bash
# Validation Script: TCP Transport
# Purpose: Verify tcp transport with proper data flow validation
# Data Flow: Master stdin → slave executes in shell → output to master stdout
# Dependencies: bash, goncat binary

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Track PIDs for cleanup
MASTER_PID=""
SLAVE_PID=""

cleanup() {
    [ -n "$MASTER_PID" ] && kill "$MASTER_PID" 2>/dev/null && wait "$MASTER_PID" 2>/dev/null
    [ -n "$SLAVE_PID" ] && kill "$SLAVE_PID" 2>/dev/null && wait "$SLAVE_PID" 2>/dev/null
    rm -f /tmp/goncat-tcp-*
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

echo -e "${GREEN}=== TCP Transport Validation ===${NC}"

TRANSPORT="tcp"
PORT=12001
TOKEN="TCP_$$$RANDOM"

# Start master with --exec (shell will run on slave side)
(echo "echo $TOKEN"; sleep 1; echo "exit") | "$REPO_ROOT/dist/goncat.elf" master listen "${TRANSPORT}://*:${PORT}" --exec /bin/sh > /tmp/goncat-tcp-master.log 2>&1 &
MASTER_PID=$!

# Wait for master to start listening
if ! poll_for_pattern /tmp/goncat-tcp-master.log "Listening on" 5; then
    echo -e "${RED}✗ Master failed to start${NC}"
    cat /tmp/goncat-tcp-master.log
    exit 1
fi
echo -e "${GREEN}✓ Master listening${NC}"

# Connect slave
"$REPO_ROOT/dist/goncat.elf" slave connect "${TRANSPORT}://localhost:${PORT}" > /tmp/goncat-tcp-slave.log 2>&1 &
SLAVE_PID=$!

# Wait for connection establishment on both sides
if ! poll_for_pattern /tmp/goncat-tcp-master.log "Session with .* established" 10; then
    echo -e "${RED}✗ Connection not established on master${NC}"
    cat /tmp/goncat-tcp-master.log
    exit 1
fi

if ! poll_for_pattern /tmp/goncat-tcp-slave.log "Session with .* established" 10; then
    echo -e "${RED}✗ Connection not established on slave${NC}"
    cat /tmp/goncat-tcp-slave.log
    exit 1
fi
echo -e "${GREEN}✓ Connection established${NC}"

# Wait for command execution and data flow
# Token should appear in master stdout (slave executes, sends back)
if ! poll_for_pattern /tmp/goncat-tcp-master.log "$TOKEN" 10; then
    echo -e "${RED}✗ Data token not found in master output${NC}"
    echo "Expected: $TOKEN"
    echo "Master output:"
    cat /tmp/goncat-tcp-master.log
    exit 1
fi
echo -e "${GREEN}✓ Data token verified (data channel working)${NC}"

# Wait for master to close session
wait "$MASTER_PID"
MASTER_EXIT=$?

# Verify session closed on both sides
if ! grep -qE "Session with .* closed" /tmp/goncat-tcp-master.log; then
    echo -e "${RED}✗ Session close not logged on master${NC}"
    exit 1
fi

if ! grep -qE "Session with .* closed" /tmp/goncat-tcp-slave.log; then
    echo -e "${RED}✗ Session close not logged on slave${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Session closed on both sides${NC}"

# Master in listen mode should exit after processing the piped input (exit command)
if [ "$MASTER_EXIT" -ne 0 ]; then
    echo -e "${RED}✗ Master exit code: $MASTER_EXIT${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Master exited cleanly${NC}"

echo -e "${GREEN}✓ TCP transport validation PASSED${NC}"
exit 0
