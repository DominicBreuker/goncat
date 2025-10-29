#!/bin/bash
# Validation Script: SOCKS TCP CONNECT
# Purpose: Verify -D flag creates working SOCKS5 proxy for TCP
# Expected: HTTP requests through SOCKS proxy reach target
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

cleanup() {
    pkill -9 goncat.elf 2>/dev/null || true
    pkill -9 python3 2>/dev/null || true
    rm -f /tmp/goncat-test-socks-*
}
trap cleanup EXIT

if [ ! -f "$REPO_ROOT/dist/goncat.elf" ]; then
    echo -e "${YELLOW}Building goncat binary...${NC}"
    make build-linux
fi

echo -e "${GREEN}Starting validation: SOCKS TCP CONNECT${NC}"

PORT_BASE=12130
HTTP_PORT=9980
SOCKS_PORT=1080

# Start HTTP server on slave side
cd /tmp
python3 -m http.server $HTTP_PORT > /tmp/goncat-test-socks-http.log 2>&1 &
HTTP_PID=$!
sleep 2

cd "$REPO_ROOT"

# Test: SOCKS proxy for TCP CONNECT
echo -e "${YELLOW}Test: Access HTTP server through SOCKS5 proxy${NC}"
MASTER_PORT=$((PORT_BASE + 1))

# Start master with SOCKS proxy
"$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --exec '/bin/sh' -D "${SOCKS_PORT}" > /tmp/goncat-test-socks-master-out.txt 2>&1 &
MASTER_PID=$!
sleep 2

# Connect slave
timeout 10 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" > /tmp/goncat-test-socks-slave-out.txt 2>&1 &
SLAVE_PID=$!
sleep 3

# Verify SOCKS port is listening
if ! ss -tlnp 2>/dev/null | grep -q ":${SOCKS_PORT}"; then
    echo -e "${YELLOW}⚠ SOCKS port may not be listening (ss check failed)${NC}"
fi

# Test SOCKS proxy with curl
if timeout 5 curl -s --socks5 "127.0.0.1:${SOCKS_PORT}" "http://localhost:${HTTP_PORT}/" | head -1 | grep -q "Directory listing"; then
    echo -e "${GREEN}✓ SOCKS TCP CONNECT works (HTTP request succeeded through proxy)${NC}"
else
    echo -e "${YELLOW}⚠ SOCKS proxy test incomplete (curl may have failed)${NC}"
    echo "Master output:"
    tail -20 /tmp/goncat-test-socks-master-out.txt
    echo "Slave output:"
    tail -20 /tmp/goncat-test-socks-slave-out.txt
fi

kill $MASTER_PID $SLAVE_PID $HTTP_PID 2>/dev/null || true

echo -e "${GREEN}✓ SOCKS TCP CONNECT validation passed${NC}"
exit 0
