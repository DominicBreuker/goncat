#!/bin/bash
# Validation Script: TLS Encryption
# Purpose: Verify --ssl flag enables TLS encryption across transports
# Expected: TLS handshake succeeds, connection fails without matching --ssl flags
# Dependencies: bash, goncat binary

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

cleanup() {
    pkill -9 goncat.elf 2>/dev/null || true
    rm -f /tmp/goncat-test-ssl-*
}
trap cleanup EXIT

if [ ! -f "$REPO_ROOT/dist/goncat.elf" ]; then
    echo -e "${YELLOW}Building goncat binary...${NC}"
    make build-linux
fi

echo -e "${GREEN}Starting validation: TLS Encryption (--ssl)${NC}"

PORT_BASE=12040

# Test 1: TLS encryption with TCP
echo -e "${YELLOW}Test 1: TCP with --ssl on both sides (should succeed)${NC}"
MASTER_PORT=$((PORT_BASE + 1))

"$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --ssl --exec /bin/sh > /tmp/goncat-test-ssl-master1-out.txt 2>&1 &
MASTER_PID=$!
sleep 2

if ! grep -q "Listening on" /tmp/goncat-test-ssl-master1-out.txt; then
    echo -e "${RED}✗ Master not listening${NC}"
    cat /tmp/goncat-test-ssl-master1-out.txt
    exit 1
fi

echo "whoami" | timeout 10 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" --ssl > /tmp/goncat-test-ssl-slave1-out.txt 2>&1 || true
sleep 1

if grep -q "Session with .* established" /tmp/goncat-test-ssl-slave1-out.txt; then
    echo -e "${GREEN}✓ TLS connection established successfully${NC}"
else
    echo -e "${RED}✗ TLS connection failed${NC}"
    cat /tmp/goncat-test-ssl-slave1-out.txt
    exit 1
fi

kill $MASTER_PID 2>/dev/null || true
sleep 1

# Test 2: Mismatched SSL flags (should fail)
echo -e "${YELLOW}Test 2: Master with --ssl, slave without (should fail)${NC}"
MASTER_PORT=$((PORT_BASE + 2))

"$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --ssl --exec /bin/sh > /tmp/goncat-test-ssl-master2-out.txt 2>&1 &
MASTER_PID=$!
sleep 2

timeout 5 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" > /tmp/goncat-test-ssl-slave2-out.txt 2>&1 || true
sleep 1

if grep -q "Session with .* established" /tmp/goncat-test-ssl-slave2-out.txt && grep -qE "(whoami|root|runner)" /tmp/goncat-test-ssl-slave2-out.txt; then
    echo -e "${RED}✗ Connection should have failed with mismatched SSL${NC}"
    exit 1
else
    echo -e "${GREEN}✓ Mismatched SSL settings correctly rejected${NC}"
fi

kill $MASTER_PID 2>/dev/null || true

echo -e "${GREEN}✓ TLS encryption validation passed${NC}"
exit 0
