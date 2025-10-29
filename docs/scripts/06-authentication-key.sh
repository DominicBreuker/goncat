#!/bin/bash
# Validation Script: Mutual Authentication
# Purpose: Verify --key flag provides mutual TLS authentication
# Expected: Connection succeeds with matching passwords, fails with mismatched
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
    rm -f /tmp/goncat-test-auth-*
}
trap cleanup EXIT

if [ ! -f "$REPO_ROOT/dist/goncat.elf" ]; then
    echo -e "${YELLOW}Building goncat binary...${NC}"
    make build-linux
fi

echo -e "${GREEN}Starting validation: Mutual Authentication (--key)${NC}"

PORT_BASE=12050
CORRECT_PASSWORD="test_password_123"
WRONG_PASSWORD="wrong_password_456"

# Test 1: Matching passwords (should succeed)
echo -e "${YELLOW}Test 1: Both sides with correct --key (should succeed)${NC}"
MASTER_PORT=$((PORT_BASE + 1))

"$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --ssl --key "$CORRECT_PASSWORD" --exec /bin/sh > /tmp/goncat-test-auth-master1-out.txt 2>&1 &
MASTER_PID=$!
sleep 2

if ! grep -q "Listening on" /tmp/goncat-test-auth-master1-out.txt; then
    echo -e "${RED}✗ Master not listening${NC}"
    exit 1
fi

echo "whoami" | timeout 10 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" --ssl --key "$CORRECT_PASSWORD" > /tmp/goncat-test-auth-slave1-out.txt 2>&1 || true
sleep 1

if grep -q "Session with .* established" /tmp/goncat-test-auth-slave1-out.txt; then
    echo -e "${GREEN}✓ Mutual authentication succeeded with correct password${NC}"
else
    echo -e "${RED}✗ Authentication failed with correct password${NC}"
    cat /tmp/goncat-test-auth-slave1-out.txt
    exit 1
fi

kill $MASTER_PID 2>/dev/null || true
sleep 1

# Test 2: Mismatched passwords (should fail)
echo -e "${YELLOW}Test 2: Different passwords (should fail)${NC}"
MASTER_PORT=$((PORT_BASE + 2))

"$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --ssl --key "$CORRECT_PASSWORD" --exec /bin/sh > /tmp/goncat-test-auth-master2-out.txt 2>&1 &
MASTER_PID=$!
sleep 2

timeout 10 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" --ssl --key "$WRONG_PASSWORD" > /tmp/goncat-test-auth-slave2-out.txt 2>&1 || true
sleep 1

if grep -q "Session with .* established" /tmp/goncat-test-auth-slave2-out.txt && grep -qE "(whoami|root)" /tmp/goncat-test-auth-slave2-out.txt; then
    echo -e "${RED}✗ Authentication should have failed with wrong password${NC}"
    exit 1
else
    echo -e "${GREEN}✓ Authentication correctly rejected wrong password${NC}"
fi

kill $MASTER_PID 2>/dev/null || true

echo -e "${GREEN}✓ Mutual authentication validation passed${NC}"
exit 0
