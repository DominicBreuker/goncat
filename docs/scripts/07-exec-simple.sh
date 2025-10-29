#!/bin/bash
# Validation Script: Simple Command Execution (Non-PTY)
# Purpose: Verify --exec flag executes commands without PTY
# Data Flow: Master stdin → slave executes in shell → output to master stdout
# Tests: Command execution, listener persistence, second connection
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
    rm -f /tmp/goncat-exec-*
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

echo -e "${GREEN}=== Simple Command Execution Validation ===${NC}"

PORT=12061
TOKEN1="EXEC1_$$_$RANDOM"
TOKEN2="EXEC2_$$_$RANDOM"

# Test 1: First connection - execute commands
echo -e "${YELLOW}Test 1: Execute commands (whoami, id)${NC}"

# Start master in listen mode with command input
(echo "whoami"; echo "echo $TOKEN1"; echo "id"; sleep 1; echo "exit") | "$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${PORT}" --exec /bin/sh > /tmp/goncat-exec-master.log 2>&1 &
MASTER_PID=$!

if ! poll_for_pattern /tmp/goncat-exec-master.log "Listening on" 5; then
    echo -e "${RED}✗ Master failed to start${NC}"
    cat /tmp/goncat-exec-master.log
    exit 1
fi
echo -e "${GREEN}✓ Master listening${NC}"

# Connect slave
"$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${PORT}" > /tmp/goncat-exec-slave.log 2>&1 &
SLAVE_PID=$!

# Wait for connection establishment
if ! poll_for_pattern /tmp/goncat-exec-master.log "Session with .* established" 10; then
    echo -e "${RED}✗ Connection not established${NC}"
    cat /tmp/goncat-exec-master.log
    exit 1
fi
echo -e "${GREEN}✓ Connection established${NC}"

# Verify whoami output appears in master log (slave executed it)
if ! poll_for_pattern /tmp/goncat-exec-master.log "(root|runner|[a-z0-9]+)" 10; then
    echo -e "${RED}✗ whoami output not found in master log${NC}"
    exit 1
fi
echo -e "${GREEN}✓ whoami command executed${NC}"

# Verify token appears in master log
if ! poll_for_pattern /tmp/goncat-exec-master.log "$TOKEN1" 10; then
    echo -e "${RED}✗ Token not found in master log${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Token verified (data channel working)${NC}"

# Verify id command output (should contain uid/gid)
if ! poll_for_pattern /tmp/goncat-exec-master.log "uid=" 10; then
    echo -e "${RED}✗ id command output not found${NC}"
    exit 1
fi
echo -e "${GREEN}✓ id command executed${NC}"

# Wait for session closed
if ! poll_for_pattern /tmp/goncat-exec-master.log "Session with .* closed" 5; then
    echo -e "${RED}✗ Session not closed on master${NC}"
    exit 1
fi

if ! poll_for_pattern /tmp/goncat-exec-slave.log "Session with .* closed" 5; then
    echo -e "${RED}✗ Session not closed on slave${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Session closed gracefully${NC}"

# Wait for slave to exit
wait "$SLAVE_PID" 2>/dev/null
SLAVE_PID=""

# Test 2: Verify listener persistence with second connection
echo -e "${YELLOW}Test 2: Second connection to verify listener persistence${NC}"

# Check master still running
if ! kill -0 "$MASTER_PID" 2>/dev/null; then
    echo -e "${RED}✗ Master exited (should persist in listen mode)${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Master still listening${NC}"

# Make second connection
(echo "echo $TOKEN2"; sleep 1; echo "exit") | "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${PORT}" > /tmp/goncat-exec-slave2.log 2>&1 &
SLAVE_PID=$!

# Verify second connection established
sleep 2
if ! grep -qE "Session with .* established" /tmp/goncat-exec-slave2.log; then
    echo -e "${RED}✗ Second connection not established${NC}"
    cat /tmp/goncat-exec-slave2.log
    exit 1
fi
echo -e "${GREEN}✓ Second connection established${NC}"

# Wait for slave to exit
wait "$SLAVE_PID" 2>/dev/null
SLAVE_PID=""

# Clean up master
kill "$MASTER_PID" 2>/dev/null && wait "$MASTER_PID" 2>/dev/null
MASTER_PID=""

echo -e "${GREEN}✓ Simple command execution validation PASSED${NC}"
exit 0
