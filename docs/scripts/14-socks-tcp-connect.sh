#!/usr/bin/env bash
#
# Validation Script 14: SOCKS5 TCP CONNECT proxy
#
# Tests SOCKS5 proxy functionality (-D flag) by:
# 1. Starting an echo server (target service)
# 2. Starting master with SOCKS proxy (-D flag)
# 3. Starting slave to connect
# 4. Using curl through SOCKS to reach echo server
# 5. Verifying data flows through the tunnel
#
# This validates that SOCKS proxy correctly tunnels TCP connections
# from master side through the slave to target services.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GONCAT="${GONCAT:-${SCRIPT_DIR}/../../dist/goncat.elf}"
HELPER_DIR="${SCRIPT_DIR}/helpers"

# Polling helper function
poll_for_pattern() {
    local file="$1"
    local pattern="$2"
    local timeout="${3:-10}"
    local interval="${4:-0.1}"
    
    local start=$(date +%s)
    while true; do
        if [[ -f "$file" ]] && grep -q "$pattern" "$file" 2>/dev/null; then
            return 0
        fi
        
        local now=$(date +%s)
        if (( now - start >= timeout )); then
            return 1
        fi
        
        sleep "$interval"
    done
}

# Test configuration
MASTER_PORT=23140
SOCKS_PORT=23141
ECHO_PORT=23142
MASTER_LOG="/tmp/goncat-socks-master-$$.log"
SLAVE_LOG="/tmp/goncat-socks-slave-$$.log"
ECHO_LOG="/tmp/goncat-socks-echo-$$.log"
TEST_TOKEN="SOCKS_TEST_$(date +%s%N)"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Cleanup function
cleanup() {
    echo "Cleaning up..."
    
    # Close FIFO file descriptor
    [[ -n "${FIFO:-}" ]] && exec 3>&- 2>/dev/null || true
    
    # Kill processes by PID
    [[ -n "${ECHO_PID:-}" ]] && kill "$ECHO_PID" 2>/dev/null || true
    [[ -n "${MASTER_PID:-}" ]] && kill "$MASTER_PID" 2>/dev/null || true
    [[ -n "${SLAVE_PID:-}" ]] && kill "$SLAVE_PID" 2>/dev/null || true
    
    # Wait for processes to exit
    [[ -n "${ECHO_PID:-}" ]] && wait "$ECHO_PID" 2>/dev/null || true
    [[ -n "${MASTER_PID:-}" ]] && wait "$MASTER_PID" 2>/dev/null || true
    [[ -n "${SLAVE_PID:-}" ]] && wait "$SLAVE_PID" 2>/dev/null || true
    
    # Clean up FIFO and log files
    [[ -n "${FIFO:-}" ]] && rm -f "$FIFO"
    rm -f "$MASTER_LOG" "$SLAVE_LOG" "$ECHO_LOG"
}

trap cleanup EXIT

echo "========================================" 
echo "SOCKS5 TCP CONNECT Proxy Validation"
echo "========================================"
echo ""

# Verify goncat binary exists
if [[ ! -x "$GONCAT" ]]; then
    echo -e "${RED}✗ FAIL${NC}: goncat binary not found at $GONCAT"
    exit 1
fi

echo "Test: SOCKS5 proxy tunnels connections through slave"
echo ""

# Step 1: Start echo server (simulates target service accessible from slave)
echo "Starting echo server on port $ECHO_PORT..."
python3 -c "
import socket
import sys

def echo_server(port):
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    server.bind(('127.0.0.1', port))
    server.listen(1)
    print(f'Echo server listening on port {port}', flush=True)
    sys.stdout.flush()
    
    while True:
        try:
            conn, addr = server.accept()
            data = conn.recv(1024)
            if data:
                response = f'ECHO: {data.decode().strip()}\n'
                conn.sendall(response.encode())
            conn.close()
        except Exception as e:
            print(f'Error: {e}', flush=True)
            break

echo_server($ECHO_PORT)
" > "$ECHO_LOG" 2>&1 &
ECHO_PID=$!

# Wait for echo server to be ready
if ! poll_for_pattern "$ECHO_LOG" "Echo server listening" 5; then
    echo -e "${RED}✗ FAIL${NC}: Echo server failed to start"
    exit 1
fi
echo -e "${GREEN}✓${NC} Echo server ready"

# Step 2: Create a FIFO for master stdin
FIFO="/tmp/goncat-socks-fifo-$$"
mkfifo "$FIFO"

# Start master with SOCKS proxy, reading from FIFO to keep stdin open
echo "Starting master with SOCKS proxy on port $SOCKS_PORT..."
"$GONCAT" master listen "tcp://*:$MASTER_PORT" -D "127.0.0.1:$SOCKS_PORT" --exec /bin/sh --verbose < "$FIFO" > "$MASTER_LOG" 2>&1 &
MASTER_PID=$!

# Keep FIFO open to prevent EOF
exec 3>"$FIFO"

# Wait for master to be listening
if ! poll_for_pattern "$MASTER_LOG" "Listening on" 5; then
    echo -e "${RED}✗ FAIL${NC}: Master failed to start listening"
    cat "$MASTER_LOG"
    exit 1
fi
echo -e "${GREEN}✓${NC} Master listening"

# Step 3: Start slave to connect
echo "Starting slave..."
"$GONCAT" slave connect "tcp://127.0.0.1:$MASTER_PORT" --verbose > "$SLAVE_LOG" 2>&1 &
SLAVE_PID=$!

# Wait for connection to be established
if ! poll_for_pattern "$MASTER_LOG" "Session with .* established" 5; then
    echo -e "${RED}✗ FAIL${NC}: Session not established"
    echo "Master log:"
    cat "$MASTER_LOG"
    echo "Slave log:"
    cat "$SLAVE_LOG"
    exit 1
fi
echo -e "${GREEN}✓${NC} Session established"

# Wait for SOCKS proxy to be ready (appears with --verbose)
if ! poll_for_pattern "$MASTER_LOG" "SOCKS proxy: listening on 127.0.0.1:$SOCKS_PORT" 5; then
    echo -e "${RED}✗ FAIL${NC}: SOCKS proxy not ready"
    cat "$MASTER_LOG"
    exit 1
fi
echo -e "${GREEN}✓${NC} SOCKS proxy ready"

# Step 4: Send a sleep command to keep the shell session alive
echo "Keeping session alive..."
echo "sleep 30" >&3

# Small delay to ensure SOCKS proxy is fully ready
sleep 1

# Step 5: Test SOCKS connectivity using Python SOCKS client
echo "Testing SOCKS proxy with unique token..."
RESPONSE=$(timeout 5 python3 -c "
import socket
import struct

# Connect to SOCKS proxy
sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
sock.connect(('127.0.0.1', $SOCKS_PORT))

# SOCKS5 greeting (no authentication)
sock.sendall(b'\x05\x01\x00')
response = sock.recv(2)

# SOCKS5 CONNECT request
request = b'\x05\x01\x00\x01'  # Version 5, CONNECT, reserved, IPv4
request += socket.inet_aton('127.0.0.1')
request += struct.pack('>H', $ECHO_PORT)
sock.sendall(request)

# Read SOCKS response
response = sock.recv(10)

# Send test token
sock.sendall(b'$TEST_TOKEN\n')

# Read echo response
result = sock.recv(1024).decode().strip()
print(result)

sock.close()
" 2>&1)

# Step 6: Verify response
if echo "$RESPONSE" | grep -q "ECHO: $TEST_TOKEN"; then
    echo -e "${GREEN}✓ PASS${NC}: Token received through SOCKS proxy"
    echo "  Response: $RESPONSE"
else
    echo -e "${RED}✗ FAIL${NC}: Token not found in response"
    echo "  Expected: ECHO: $TEST_TOKEN"
    echo "  Got: $RESPONSE"
    echo ""
    echo "Master log:"
    cat "$MASTER_LOG"
    echo ""
    echo "Slave log:"
    cat "$SLAVE_LOG"
    exit 1
fi

# Step 7: Verify second connection (proxy persists)
echo "Testing second connection through SOCKS..."
TEST_TOKEN2="SOCKS_TEST2_$(date +%s%N)"
RESPONSE2=$(timeout 5 python3 -c "
import socket
import struct

sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
sock.connect(('127.0.0.1', $SOCKS_PORT))

# SOCKS5 greeting
sock.sendall(b'\x05\x01\x00')
sock.recv(2)

# SOCKS5 CONNECT
request = b'\x05\x01\x00\x01'
request += socket.inet_aton('127.0.0.1')
request += struct.pack('>H', $ECHO_PORT)
sock.sendall(request)
sock.recv(10)

# Send token
sock.sendall(b'$TEST_TOKEN2\n')
result = sock.recv(1024).decode().strip()
print(result)
sock.close()
" 2>&1)

if echo "$RESPONSE2" | grep -q "ECHO: $TEST_TOKEN2"; then
    echo -e "${GREEN}✓ PASS${NC}: Second connection successful (proxy persists)"
else
    echo -e "${RED}✗ FAIL${NC}: Second connection failed"
    echo "  Got: $RESPONSE2"
    exit 1
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✓ ALL SOCKS5 TESTS PASSED${NC}"
echo -e "${GREEN}========================================${NC}"
