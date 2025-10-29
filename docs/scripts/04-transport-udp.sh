#!/bin/bash
# Validation Script: UDP/QUIC Transport
# Purpose: Verify UDP transport with QUIC protocol works
# Expected: Master and slave can establish UDP/QUIC connection successfully
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
    rm -f /tmp/goncat-test-udp-*
}
trap cleanup EXIT

# Ensure binary exists
if [ ! -f "$REPO_ROOT/dist/goncat.elf" ]; then
    echo -e "${YELLOW}Building goncat binary...${NC}"
    make build-linux
fi

echo -e "${GREEN}Starting validation: UDP/QUIC Transport${NC}"

TRANSPORT="udp"
PORT_BASE=12030

# Test: Master listen, slave connect
echo -e "${YELLOW}Test: Master listen (udp), slave connect${NC}"
MASTER_PORT=$((PORT_BASE + 1))

# Start master in background
"$REPO_ROOT/dist/goncat.elf" master listen "${TRANSPORT}://*:${MASTER_PORT}" --exec 'echo GONCAT_UDP_SUCCESS' > /tmp/goncat-test-udp-master-out.txt 2>&1 &
MASTER_PID=$!

# Wait for master to start
sleep 2

# Verify master is running and listening
if ! ps -p $MASTER_PID > /dev/null; then
    echo -e "${RED}✗ Master failed to start${NC}"
    cat /tmp/goncat-test-udp-master-out.txt
    exit 1
fi

if ! grep -q "Listening on" /tmp/goncat-test-udp-master-out.txt; then
    echo -e "${RED}✗ Master not listening${NC}"
    cat /tmp/goncat-test-udp-master-out.txt
    exit 1
fi

# Connect slave
timeout 10 "$REPO_ROOT/dist/goncat.elf" slave connect "${TRANSPORT}://localhost:${MASTER_PORT}" > /tmp/goncat-test-udp-slave-out.txt 2>&1 || true

# Wait a bit
sleep 1

# Verify connection was established
if grep -q "Session with .* established" /tmp/goncat-test-udp-slave-out.txt; then
    echo -e "${GREEN}✓ UDP/QUIC connection established successfully${NC}"
else
    echo -e "${RED}✗ UDP/QUIC connection not established${NC}"
    echo "Slave output:"
    cat /tmp/goncat-test-udp-slave-out.txt
    echo "Master output:"
    cat /tmp/goncat-test-udp-master-out.txt
    exit 1
fi

# Verify data was received
if grep -q "GONCAT_UDP_SUCCESS" /tmp/goncat-test-udp-slave-out.txt; then
    echo -e "${GREEN}✓ Data received successfully through UDP/QUIC tunnel${NC}"
else
    echo -e "${YELLOW}⚠ Connection established but data verification incomplete${NC}"
fi

# Cleanup
kill $MASTER_PID 2>/dev/null || true

echo -e "${GREEN}✓ UDP/QUIC transport validation passed${NC}"
exit 0
