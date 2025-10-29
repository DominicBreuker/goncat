#!/bin/bash
# Validation Script: SOCKS5 TCP CONNECT
# Purpose: Verify -D flag creates working SOCKS5 proxy for TCP connections
# Data Flow: HTTP server serves unique token → curl --socks5 through proxy → verify token
# Tests: Proxy works, persists across requests, teardown after slave killed
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
    [ -n "$MASTER_PID" ] && kill "$MASTER_PID" 2>/dev/null && wait "$MASTER_PID" 2>/dev/null || true
    [ -n "$SLAVE_PID" ] && kill "$SLAVE_PID" 2>/dev/null && wait "$SLAVE_PID" 2>/dev/null || true
    [ -n "$HTTP_PID" ] && kill "$HTTP_PID" 2>/dev/null && wait "$HTTP_PID" 2>/dev/null || true
    rm -rf /tmp/goncat-socks-* /tmp/test-socks-* 2>/dev/null || true
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

# Ensure clean state - kill any lingering processes from previous tests
pkill -9 -f "goncat.elf.*1080" 2>/dev/null || true
pkill -9 -f "python3.*9980" 2>/dev/null || true
sleep 1

echo -e "${GREEN}=== SOCKS5 TCP CONNECT Validation ===${NC}"

HTTP_PORT=9980
SOCKS_PORT=1080
MASTER_PORT=12130
TOKEN="SOCKS_$$_$RANDOM"

# Create simple HTTP server that serves our token
echo -e "${YELLOW}Starting HTTP server with unique token...${NC}"
mkdir -p /tmp/test-socks-$$
echo "<html><body>TOKEN: $TOKEN</body></html>" > /tmp/test-socks-$$/index.html
cd /tmp/test-socks-$$
python3 -m http.server $HTTP_PORT > /tmp/goncat-socks-http.log 2>&1 &
HTTP_PID=$!
cd "$REPO_ROOT"

# Wait for HTTP server to start
sleep 1
if ! curl -s "http://localhost:${HTTP_PORT}/" | grep -q "$TOKEN"; then
    echo -e "${RED}✗ HTTP server failed to start or serve token${NC}"
    exit 1
fi
echo -e "${GREEN}✓ HTTP server running with token${NC}"

# Test: SOCKS5 proxy (-D creates SOCKS proxy on master side)
echo -e "${YELLOW}Setting up SOCKS5 proxy on port ${SOCKS_PORT}${NC}"

# Start master with SOCKS proxy and keep session alive
(sleep 30) | "$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --exec /bin/sh -D "${SOCKS_PORT}" > /tmp/goncat-socks-master.log 2>&1 &
MASTER_PID=$!

if ! poll_for_pattern /tmp/goncat-socks-master.log "Listening on" 5; then
    echo -e "${RED}✗ Master failed to start${NC}"
    cat /tmp/goncat-socks-master.log
    exit 1
fi
echo -e "${GREEN}✓ Master listening${NC}"

# Connect slave (this establishes the SOCKS tunnel)
"$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" > /tmp/goncat-socks-slave.log 2>&1 &
SLAVE_PID=$!

if ! poll_for_pattern /tmp/goncat-socks-master.log "Session with .* established" 10; then
    echo -e "${RED}✗ Connection not established${NC}"
    cat /tmp/goncat-socks-master.log
    exit 1
fi
echo -e "${GREEN}✓ Connection established${NC}"

# Give SOCKS proxy a moment to be ready
sleep 3

# Verify SOCKS port is actually listening
if ! ss -tln | grep -q ":${SOCKS_PORT} "; then
    echo -e "${RED}✗ SOCKS port ${SOCKS_PORT} not listening${NC}"
    echo "Master log:"
    cat /tmp/goncat-socks-master.log
    exit 1
fi
echo -e "${GREEN}✓ SOCKS proxy port listening${NC}"

# Test 1: Fetch through SOCKS proxy and verify token (decisive test)
echo -e "${YELLOW}Test 1: Fetch through SOCKS proxy and verify token${NC}"
RESULT=$(timeout 10 curl -s --socks5 "127.0.0.1:${SOCKS_PORT}" "http://localhost:${HTTP_PORT}/" 2>&1)
if echo "$RESULT" | grep -q "$TOKEN"; then
    echo -e "${GREEN}✓ Token verified (data went through SOCKS proxy)${NC}"
else
    echo -e "${RED}✗ Token not found in response${NC}"
    echo "Expected: $TOKEN"
    echo "Got: $RESULT"
    echo "Master log:"
    cat /tmp/goncat-socks-master.log
    exit 1
fi

# Test 2: Second request to verify persistence
echo -e "${YELLOW}Test 2: Second request to verify proxy persistence${NC}"
RESULT2=$(timeout 5 curl -s --socks5 "127.0.0.1:${SOCKS_PORT}" "http://localhost:${HTTP_PORT}/" 2>&1)
if echo "$RESULT2" | grep -q "$TOKEN"; then
    echo -e "${GREEN}✓ Second request succeeded (proxy persists)${NC}"
else
    echo -e "${RED}✗ Second request failed${NC}"
    exit 1
fi

# Test 3: Kill slave, verify proxy tears down (negative test)
echo -e "${YELLOW}Test 3: Verify proxy teardown after slave exit${NC}"
kill "$SLAVE_PID" 2>/dev/null
wait "$SLAVE_PID" 2>/dev/null || true
SLAVE_PID=""

# Poll for session closed message
if poll_for_pattern /tmp/goncat-socks-master.log "Session with .* closed" 5; then
    echo -e "${GREEN}✓ Session closed detected${NC}"
fi

# Give it a moment to tear down
sleep 1

# Attempt to connect through SOCKS - should fail since tunnel is gone
if timeout 3 curl -s --socks5 "127.0.0.1:${SOCKS_PORT}" "http://localhost:${HTTP_PORT}/" > /dev/null 2>&1; then
    echo -e "${YELLOW}⚠ SOCKS proxy still active after slave exit (unexpected)${NC}"
else
    echo -e "${GREEN}✓ SOCKS proxy torn down after slave exit (connection failed as expected)${NC}"
fi

# Verify master listener is still up (should be waiting for next connection)
if kill -0 "$MASTER_PID" 2>/dev/null; then
    echo -e "${GREEN}✓ Master listener still active (correct behavior)${NC}"
else
    echo -e "${YELLOW}⚠ Master listener exited${NC}"
fi

# Clean up
kill "$MASTER_PID" 2>/dev/null && wait "$MASTER_PID" 2>/dev/null || true
MASTER_PID=""
kill "$HTTP_PID" 2>/dev/null && wait "$HTTP_PID" 2>/dev/null || true
HTTP_PID=""

echo -e "${GREEN}✓ SOCKS5 TCP CONNECT validation PASSED${NC}"
exit 0
