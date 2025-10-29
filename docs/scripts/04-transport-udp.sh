#!/bin/bash
# Validation Script: UDP/QUIC Transport
# Purpose: Verify udp transport (QUIC) with proper data flow validation
# Data Flow: Master stdin → slave executes in shell → output to master stdout
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
    [ -n "$MASTER_PID" ] && kill "$MASTER_PID" 2>/dev/null && wait "$MASTER_PID" 2>/dev/null
    [ -n "$SLAVE_PID" ] && kill "$SLAVE_PID" 2>/dev/null && wait "$SLAVE_PID" 2>/dev/null
    rm -f /tmp/goncat-udp-*
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

echo -e "${GREEN}=== UDP/QUIC Transport Validation ===${NC}"

TRANSPORT="udp"
PORT=12013
# Multi-line payload to test UDP segmentation/ordering
TOKEN="UDP_$$_$RANDOM"
MULTILINE="Line1_$TOKEN
Line2_$TOKEN
Line3_$TOKEN"

# Start master with --exec (shell will run on slave side)
(echo "echo '$MULTILINE'"; sleep 1; echo "exit") | "$REPO_ROOT/dist/goncat.elf" master listen "${TRANSPORT}://*:${PORT}" --exec /bin/sh > /tmp/goncat-udp-master.log 2>&1 &
MASTER_PID=$!

# Wait for master to start listening
if ! poll_for_pattern /tmp/goncat-udp-master.log "Listening on" 5; then
    echo -e "${RED}✗ Master failed to start${NC}"
    cat /tmp/goncat-udp-master.log
    exit 1
fi
echo -e "${GREEN}✓ Master listening${NC}"

# Connect slave
"$REPO_ROOT/dist/goncat.elf" slave connect "${TRANSPORT}://localhost:${PORT}" > /tmp/goncat-udp-slave.log 2>&1 &
SLAVE_PID=$!

# Wait for connection establishment on both sides
if ! poll_for_pattern /tmp/goncat-udp-master.log "Session with .* established" 10; then
    echo -e "${RED}✗ Connection not established on master${NC}"
    cat /tmp/goncat-udp-master.log
    exit 1
fi

if ! poll_for_pattern /tmp/goncat-udp-slave.log "Session with .* established" 10; then
    echo -e "${RED}✗ Connection not established on slave${NC}"
    cat /tmp/goncat-udp-slave.log
    exit 1
fi
echo -e "${GREEN}✓ Connection established${NC}"

# Wait for command execution and multi-line data flow
# All lines should appear in master stdout (slave executes, sends back)
if ! poll_for_pattern /tmp/goncat-udp-master.log "Line1_$TOKEN" 10; then
    echo -e "${RED}✗ Line 1 not found in master output${NC}"
    echo "Master output:"
    cat /tmp/goncat-udp-master.log
    exit 1
fi

if ! poll_for_pattern /tmp/goncat-udp-master.log "Line2_$TOKEN" 10; then
    echo -e "${RED}✗ Line 2 not found in master output${NC}"
    exit 1
fi

if ! poll_for_pattern /tmp/goncat-udp-master.log "Line3_$TOKEN" 10; then
    echo -e "${RED}✗ Line 3 not found in master output${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Multi-line data verified (UDP/QUIC data channel working)${NC}"

# Poll for session closed
if ! poll_for_pattern /tmp/goncat-udp-master.log "Session with .* closed" 5; then
    echo -e "${RED}✗ Session close not logged on master${NC}"
    exit 1
fi

if ! poll_for_pattern /tmp/goncat-udp-slave.log "Session with .* closed" 5; then
    echo -e "${RED}✗ Session close not logged on slave${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Session closed on both sides${NC}"

# Wait for slave to exit
if wait "$SLAVE_PID" 2>/dev/null; then
    SLAVE_EXIT=$?
    if [ "$SLAVE_EXIT" -ne 0 ]; then
        echo -e "${RED}✗ Slave exit code: $SLAVE_EXIT${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Slave exited cleanly${NC}"
fi

# Master in listen mode may persist or exit
if timeout 3 bash -c "wait $MASTER_PID 2>/dev/null"; then
    echo -e "${GREEN}✓ Master exited cleanly${NC}"
else
    if kill -0 "$MASTER_PID" 2>/dev/null; then
        echo -e "${GREEN}✓ Master still listening (persistence working)${NC}"
        kill "$MASTER_PID" 2>/dev/null
        wait "$MASTER_PID" 2>/dev/null
    fi
fi

echo -e "${GREEN}✓ UDP/QUIC transport validation PASSED${NC}"
exit 0
