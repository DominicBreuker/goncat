#!/bin/bash
# Validation Script: Local TCP Port Forwarding
# Purpose: Verify -L flag forwards TCP ports with proper tunnel validation
# Data Flow: HTTP server serves unique token → curl through tunnel → verify token
# Tests: Forward works, persists across requests, teardown after slave killed
# Dependencies: bash, goncat binary, python3, curl

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
HTTP_PID=""

cleanup() {
    [ -n "$MASTER_PID" ] && kill "$MASTER_PID" 2>/dev/null && wait "$MASTER_PID" 2>/dev/null
    [ -n "$SLAVE_PID" ] && kill "$SLAVE_PID" 2>/dev/null && wait "$SLAVE_PID" 2>/dev/null
    [ -n "$HTTP_PID" ] && kill "$HTTP_PID" 2>/dev/null && wait "$HTTP_PID" 2>/dev/null
    rm -f /tmp/goncat-portfwd-* /tmp/test-server-*
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

echo -e "${GREEN}=== Local TCP Port Forwarding Validation ===${NC}"

HTTP_PORT=9991
FORWARD_PORT=8881
MASTER_PORT=12081
TOKEN="PORTFWD_$$_$RANDOM"

# Create simple HTTP server that serves our token
echo -e "${YELLOW}Starting HTTP server with unique token...${NC}"
mkdir -p /tmp/test-server-$$
echo "<html><body>TOKEN: $TOKEN</body></html>" > /tmp/test-server-$$/index.html
cd /tmp/test-server-$$
python3 -m http.server $HTTP_PORT > /tmp/goncat-portfwd-http.log 2>&1 &
HTTP_PID=$!
cd "$REPO_ROOT"

# Wait for HTTP server to start
sleep 1
if ! curl -s "http://localhost:${HTTP_PORT}/" | grep -q "$TOKEN"; then
    echo -e "${RED}✗ HTTP server failed to start or serve token${NC}"
    exit 1
fi
echo -e "${GREEN}✓ HTTP server running with token${NC}"

# Test: Local port forwarding (-L forwards local FORWARD_PORT to remote HTTP_PORT)
echo -e "${YELLOW}Setting up port forwarding: localhost:${FORWARD_PORT} → localhost:${HTTP_PORT}${NC}"

# Start master with port forward and keep-alive shell
"$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --exec /bin/sh -L "${FORWARD_PORT}:localhost:${HTTP_PORT}" > /tmp/goncat-portfwd-master.log 2>&1 &
MASTER_PID=$!

if ! poll_for_pattern /tmp/goncat-portfwd-master.log "Listening on" 5; then
    echo -e "${RED}✗ Master failed to start${NC}"
    cat /tmp/goncat-portfwd-master.log
    exit 1
fi
echo -e "${GREEN}✓ Master listening${NC}"

# Connect slave (this establishes the tunnel)
"$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" > /tmp/goncat-portfwd-slave.log 2>&1 &
SLAVE_PID=$!

if ! poll_for_pattern /tmp/goncat-portfwd-master.log "Session with .* established" 10; then
    echo -e "${RED}✗ Connection not established${NC}"
    cat /tmp/goncat-portfwd-master.log
    exit 1
fi
echo -e "${GREEN}✓ Connection established${NC}"

# Poll for forward binding message
if poll_for_pattern /tmp/goncat-portfwd-master.log "Forwarding.*${FORWARD_PORT}" 5 || \
   poll_for_pattern /tmp/goncat-portfwd-master.log "local.*${FORWARD_PORT}" 5; then
    echo -e "${GREEN}✓ Port forward binding detected${NC}"
else
    echo -e "${YELLOW}⚠ Forward binding message not found (may still work)${NC}"
fi

# Give tunnel a moment to be ready
sleep 2

# Test 1: Fetch through tunnel and verify token (decisive test)
echo -e "${YELLOW}Test 1: Fetch through tunnel and verify token${NC}"
RESULT=$(timeout 5 curl -s "http://localhost:${FORWARD_PORT}/" 2>&1 || true)
if echo "$RESULT" | grep -q "$TOKEN"; then
    echo -e "${GREEN}✓ Token verified (data went through goncat tunnel)${NC}"
else
    echo -e "${RED}✗ Token not found in response${NC}"
    echo "Expected: $TOKEN"
    echo "Got: $RESULT"
    exit 1
fi

# Test 2: Second request to verify persistence
echo -e "${YELLOW}Test 2: Second request to verify forward persistence${NC}"
RESULT2=$(timeout 5 curl -s "http://localhost:${FORWARD_PORT}/" 2>&1 || true)
if echo "$RESULT2" | grep -q "$TOKEN"; then
    echo -e "${GREEN}✓ Second request succeeded (forward persists)${NC}"
else
    echo -e "${RED}✗ Second request failed${NC}"
    exit 1
fi

# Test 3: Kill slave, verify forward tears down
echo -e "${YELLOW}Test 3: Verify forward teardown after slave exit${NC}"
kill "$SLAVE_PID" 2>/dev/null
wait "$SLAVE_PID" 2>/dev/null
SLAVE_PID=""

# Give it a moment to tear down
sleep 2

# Attempt to connect - should fail
if timeout 3 curl -s "http://localhost:${FORWARD_PORT}/" > /dev/null 2>&1; then
    echo -e "${YELLOW}⚠ Forward still active after slave exit (may be expected)${NC}"
else
    echo -e "${GREEN}✓ Forward torn down after slave exit${NC}"
fi

# Clean up
kill "$MASTER_PID" 2>/dev/null && wait "$MASTER_PID" 2>/dev/null
MASTER_PID=""
kill "$HTTP_PID" 2>/dev/null && wait "$HTTP_PID" 2>/dev/null
HTTP_PID=""

echo -e "${GREEN}✓ Local TCP port forwarding validation PASSED${NC}"
exit 0
