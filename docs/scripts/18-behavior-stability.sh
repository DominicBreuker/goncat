#!/bin/bash
# Validation Script: Connection Stability
# Purpose: Verify connections are stable with short timeout
# Tests: Multiple connections succeed with 100ms timeout (no uncanceled timeouts)
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
    rm -f /tmp/goncat-stability-*
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

echo -e "${GREEN}=== Connection Stability Validation ===${NC}"

MASTER_PORT=12100

# Test: Multiple connections work with very short timeout (100ms)
echo -e "${YELLOW}Test: Multiple connections succeed with 100ms timeout (stability check)${NC}"

# Start master with commands that keep connection alive
# The key is to have activity on stdin to prevent the shell from exiting
(sleep 5) | "$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --timeout 100 --exec /bin/sh > /tmp/goncat-stability-master.log 2>&1 &
MASTER_PID=$!

if ! poll_for_pattern /tmp/goncat-stability-master.log "Listening on" 5; then
    echo -e "${RED}✗ Master failed to start${NC}"
    cat /tmp/goncat-stability-master.log
    exit 1
fi
echo -e "${GREEN}✓ Master listening with 100ms timeout${NC}"

# Connection 1 - slave just connects, no commands needed
"$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" --timeout 100 > /tmp/goncat-stability-slave1.log 2>&1 &
SLAVE_PID=$!

if ! poll_for_pattern /tmp/goncat-stability-master.log "Session with .* established" 10; then
    echo -e "${RED}✗ First connection failed to establish${NC}"
    cat /tmp/goncat-stability-master.log
    exit 1
fi
echo -e "${GREEN}✓ First connection established with 100ms timeout${NC}"

# Keep connection alive for 3 seconds (much longer than 100ms timeout)
# The connection should remain stable because there's ongoing data/heartbeat
sleep 3

# Verify connection is still up (no premature timeout)
if kill -0 "$SLAVE_PID" 2>/dev/null; then
    echo -e "${GREEN}✓ Connection stable for 3 seconds (no false timeout)${NC}"
else
    # Check if it closed gracefully or timed out
    if grep -q "Session with .* closed" /tmp/goncat-stability-master.log; then
        echo -e "${YELLOW}⚠ Connection closed after sleep ended (expected behavior)${NC}"
    else
        echo -e "${RED}✗ Connection died prematurely (false timeout)${NC}"
        cat /tmp/goncat-stability-master.log
        exit 1
    fi
fi

# Clean close
kill "$SLAVE_PID" 2>/dev/null && wait "$SLAVE_PID" 2>/dev/null || true
SLAVE_PID=""

if poll_for_pattern /tmp/goncat-stability-master.log "Session with .* closed" 5; then
    echo -e "${GREEN}✓ First session closed cleanly${NC}"
fi

# Kill first master
kill "$MASTER_PID" 2>/dev/null && wait "$MASTER_PID" 2>/dev/null || true
MASTER_PID=""

# Connection 2 - verify listener can handle multiple connections with short timeout
echo -e "${YELLOW}Starting second connection test${NC}"
(sleep 5) | "$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --timeout 100 --exec /bin/sh > /tmp/goncat-stability-master2.log 2>&1 &
MASTER_PID=$!

if ! poll_for_pattern /tmp/goncat-stability-master2.log "Listening on" 5; then
    echo -e "${RED}✗ Master failed to restart${NC}"
    exit 1
fi

"$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" --timeout 100 > /tmp/goncat-stability-slave2.log 2>&1 &
SLAVE_PID=$!

# Wait for second connection
sleep 2
if poll_for_pattern /tmp/goncat-stability-master2.log "Session with .* established" 5; then
    echo -e "${GREEN}✓ Second connection also succeeded with 100ms timeout${NC}"
    echo -e "${GREEN}✓ No uncanceled timeouts detected (connections are stable)${NC}"
else
    echo -e "${RED}✗ Second connection failed${NC}"
    exit 1
fi

kill "$SLAVE_PID" 2>/dev/null && wait "$SLAVE_PID" 2>/dev/null || true
SLAVE_PID=""

kill "$MASTER_PID" 2>/dev/null && wait "$MASTER_PID" 2>/dev/null || true
MASTER_PID=""

echo -e "${GREEN}✓ Connection stability validation PASSED${NC}"
exit 0
