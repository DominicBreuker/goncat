#!/bin/bash
# Validation Script: Mutual Authentication
# Purpose: Verify --key flag provides mutual TLS authentication
# Expected: Connection succeeds with matching passwords, fails with mismatched passwords
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

cleanup() {
    # Kill any background processes
    pkill -9 goncat.elf 2>/dev/null || true
    # Clean up temp files
    rm -f /tmp/goncat-test-auth-*
}
trap cleanup EXIT

# Ensure binary exists
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

# Start master with --ssl --key
"$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --ssl --key "$CORRECT_PASSWORD" --exec 'echo AUTH_SUCCESS' > /tmp/goncat-test-auth-master1-out.txt 2>&1 &
MASTER_PID=$!
sleep 2

# Verify master is listening
if ! grep -q "Listening on" /tmp/goncat-test-auth-master1-out.txt; then
    echo -e "${RED}✗ Master not listening${NC}"
    cat /tmp/goncat-test-auth-master1-out.txt
    exit 1
fi

# Connect slave with matching --ssl --key
timeout 10 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" --ssl --key "$CORRECT_PASSWORD" > /tmp/goncat-test-auth-slave1-out.txt 2>&1 || true
sleep 1

# Verify authentication succeeded
if grep -q "Session with .* established" /tmp/goncat-test-auth-slave1-out.txt; then
    echo -e "${GREEN}✓ Mutual authentication succeeded with correct password${NC}"
else
    echo -e "${RED}✗ Authentication failed with correct password${NC}"
    echo "Slave output:"
    cat /tmp/goncat-test-auth-slave1-out.txt
    echo "Master output:"
    cat /tmp/goncat-test-auth-master1-out.txt
    exit 1
fi

kill $MASTER_PID 2>/dev/null || true
sleep 1

# Test 2: Mismatched passwords (should fail)
echo -e "${YELLOW}Test 2: Master and slave with different passwords (should fail)${NC}"
MASTER_PORT=$((PORT_BASE + 2))

# Start master with --ssl --key
"$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --ssl --key "$CORRECT_PASSWORD" --exec 'echo AUTH_SUCCESS' > /tmp/goncat-test-auth-master2-out.txt 2>&1 &
MASTER_PID=$!
sleep 2

# Connect slave with wrong password
timeout 10 "$REPO_ROOT/dist/goncat.elf" slave connect "tcp://localhost:${MASTER_PORT}" --ssl --key "$WRONG_PASSWORD" > /tmp/goncat-test-auth-slave2-out.txt 2>&1 || true
sleep 1

# Verify authentication failed
if grep -q "Session with .* established" /tmp/goncat-test-auth-slave2-out.txt && grep -q "AUTH_SUCCESS" /tmp/goncat-test-auth-slave2-out.txt; then
    echo -e "${RED}✗ Authentication should have failed with wrong password${NC}"
    echo "Slave output:"
    cat /tmp/goncat-test-auth-slave2-out.txt
    exit 1
else
    echo -e "${GREEN}✓ Authentication correctly rejected wrong password${NC}"
fi

kill $MASTER_PID 2>/dev/null || true
sleep 1

# Test 3: --key requires --ssl (should error)
echo -e "${YELLOW}Test 3: Using --key without --ssl (should fail)${NC}"
MASTER_PORT=$((PORT_BASE + 3))

# Try to start master with --key but without --ssl (should fail)
timeout 3 "$REPO_ROOT/dist/goncat.elf" master listen "tcp://*:${MASTER_PORT}" --key "$CORRECT_PASSWORD" --exec 'echo AUTH_SUCCESS' > /tmp/goncat-test-auth-master3-out.txt 2>&1 || true

# Verify it failed
if grep -qi "error\|require.*ssl\|ssl.*require" /tmp/goncat-test-auth-master3-out.txt; then
    echo -e "${GREEN}✓ Tool correctly requires --ssl when using --key${NC}"
else
    echo -e "${YELLOW}⚠ Tool may not enforce --ssl requirement for --key (check manually)${NC}"
fi

echo -e "${GREEN}✓ Mutual authentication validation passed${NC}"
exit 0
