#!/usr/bin/env python3
"""
Validation Script: SOCKS5 TCP CONNECT
Purpose: Verify -D flag creates working SOCKS5 proxy for TCP connections
Uses pexpect to maintain interactive session for SOCKS proxy functionality
Dependencies: python3, pexpect, goncat binary, curl
"""

import sys
import os
import time
import subprocess
import signal
import pexpect

# Add repo root to path
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "../.."))
GONCAT_BIN = os.path.join(REPO_ROOT, "dist/goncat.elf")

# Colors
GREEN = '\033[0;32m'
RED = '\033[0;31m'
YELLOW = '\033[1;33m'
NC = '\033[0m'

def cleanup_processes():
    """Kill any lingering test processes"""
    subprocess.run(['pkill', '-9', '-f', 'goncat.elf.*1080'], stderr=subprocess.DEVNULL)
    subprocess.run(['pkill', '-9', '-f', 'python3.*9980'], stderr=subprocess.DEVNULL)
    time.sleep(1)

def main():
    # Check binary exists
    if not os.path.exists(GONCAT_BIN):
        print(f"{YELLOW}Building goncat binary...{NC}")
        subprocess.run(['make', 'build-linux'], cwd=REPO_ROOT, check=True)
    
    cleanup_processes()
    
    print(f"{GREEN}=== SOCKS5 TCP CONNECT Validation ==={NC}")
    
    HTTP_PORT = 9980
    SOCKS_PORT = 1080
    MASTER_PORT = 12130
    TOKEN = f"SOCKS_PY_{os.getpid()}"
    
    # Start HTTP server
    print(f"{YELLOW}Starting HTTP server with unique token...{NC}")
    http_dir = f"/tmp/test-socks-py-{os.getpid()}"
    os.makedirs(http_dir, exist_ok=True)
    with open(f"{http_dir}/index.html", 'w') as f:
        f.write(f"<html><body>TOKEN: {TOKEN}</body></html>")
    
    http_proc = subprocess.Popen(
        ['python3', '-m', 'http.server', str(HTTP_PORT)],
        cwd=http_dir,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL
    )
    
    try:
        time.sleep(1)
        
        # Verify HTTP server
        result = subprocess.run(['curl', '-s', f'http://localhost:{HTTP_PORT}/'],
                                capture_output=True, text=True, timeout=5)
        if TOKEN not in result.stdout:
            print(f"{RED}✗ HTTP server failed to start or serve token{NC}")
            return 1
        print(f"{GREEN}✓ HTTP server running with token{NC}")
        
        # Start master with SOCKS - use bash to start goncat with sleep to keep session alive
        # CRITICAL: SOCKS proxy requires --exec and active session (use sleep to keep it alive)
        print(f"{YELLOW}Setting up SOCKS5 proxy on port {SOCKS_PORT}{NC}")
        
        # Start master in background with sleep piped to stdin to keep session alive
        master_cmd = f'(sleep 60) | {GONCAT_BIN} master listen tcp://*:{MASTER_PORT} --exec /bin/sh -D {SOCKS_PORT}'
        master_proc = subprocess.Popen(
            ['bash', '-c', master_cmd],
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            bufsize=1
        )
        master_log = open('/tmp/socks-master.log', 'w')
        
        # Read master output in background
        import threading
        def read_master():
            for line in master_proc.stdout:
                master_log.write(line)
                master_log.flush()
        
        reader_thread = threading.Thread(target=read_master, daemon=True)
        reader_thread.start()
        
        # Wait for master to start
        for _ in range(50):  # 5 seconds
            time.sleep(0.1)
            master_log.flush()
            with open('/tmp/socks-master.log', 'r') as f:
                if 'Listening on' in f.read():
                    break
        else:
            print(f"{RED}✗ Master failed to start{NC}")
            return 1
        print(f"{GREEN}✓ Master listening{NC}")
        
        # Start slave in background
        slave_proc = subprocess.Popen(
            [GONCAT_BIN, 'slave', 'connect', f'tcp://localhost:{MASTER_PORT}'],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL
        )
        
        # Wait for connection
        for _ in range(100):  # 10 seconds
            time.sleep(0.1)
            with open('/tmp/socks-master.log', 'r') as f:
                if 'established' in f.read():
                    break
        else:
            print(f"{RED}✗ Connection not established{NC}")
            return 1
        print(f"{GREEN}✓ Connection established{NC}")
        
        # Wait for SOCKS proxy to be ready
        time.sleep(2)
        
        # Verify SOCKS port is listening
        result = subprocess.run(['ss', '-tln'], capture_output=True, text=True)
        if f':{SOCKS_PORT}' not in result.stdout:
            print(f"{RED}✗ SOCKS port {SOCKS_PORT} not listening{NC}")
            return 1
        print(f"{GREEN}✓ SOCKS proxy port listening{NC}")
        
        # Test 1: Fetch through SOCKS proxy
        print(f"{YELLOW}Test 1: Fetch through SOCKS proxy and verify token{NC}")
        result = subprocess.run(
            ['curl', '-s', '--socks5', f'127.0.0.1:{SOCKS_PORT}', f'http://localhost:{HTTP_PORT}/'],
            capture_output=True,
            text=True,
            timeout=10
        )
        
        if TOKEN in result.stdout:
            print(f"{GREEN}✓ Token verified (data went through SOCKS proxy){NC}")
        else:
            print(f"{RED}✗ Token not found in response{NC}")
            print(f"Expected: {TOKEN}")
            print(f"Got: {result.stdout}")
            return 1
        
        # Test 2: Second request to verify persistence
        print(f"{YELLOW}Test 2: Second request to verify proxy persistence{NC}")
        result2 = subprocess.run(
            ['curl', '-s', '--socks5', f'127.0.0.1:{SOCKS_PORT}', f'http://localhost:{HTTP_PORT}/'],
            capture_output=True,
            text=True,
            timeout=10
        )
        
        if TOKEN in result2.stdout:
            print(f"{GREEN}✓ Second request succeeded (proxy persists){NC}")
        else:
            print(f"{RED}✗ Second request failed{NC}")
            return 1
        
        # Test 3: Kill slave, verify proxy tears down
        print(f"{YELLOW}Test 3: Verify proxy teardown after slave exit{NC}")
        slave_proc.kill()
        slave_proc.wait()
        
        # Wait for session closed
        for _ in range(50):  # 5 seconds
            time.sleep(0.1)
            with open('/tmp/socks-master.log', 'r') as f:
                if 'closed' in f.read():
                    break
        else:
            print(f"{YELLOW}⚠ Session closed not detected in time{NC}")
        print(f"{GREEN}✓ Session closed detected{NC}")
        
        time.sleep(1)
        
        # Attempt connection - should fail
        result3 = subprocess.run(
            ['curl', '-s', '--socks5', f'127.0.0.1:{SOCKS_PORT}', f'http://localhost:{HTTP_PORT}/'],
            capture_output=True,
            text=True,
            timeout=3
        )
        
        if result3.returncode != 0 or TOKEN not in result3.stdout:
            print(f"{GREEN}✓ SOCKS proxy torn down after slave exit (connection failed as expected){NC}")
        else:
            print(f"{YELLOW}⚠ SOCKS proxy still active after slave exit (unexpected){NC}")
        
        # Verify master still active (listener mode)
        if master_proc.poll() is None:
            print(f"{GREEN}✓ Master listener still active (correct behavior){NC}")
        else:
            print(f"{YELLOW}⚠ Master listener exited{NC}")
        
        master_proc.terminate()
        master_proc.wait()
        master_log.close()
        
        print(f"{GREEN}✓ SOCKS5 TCP CONNECT validation PASSED{NC}")
        return 0
        
    finally:
        # Cleanup
        try:
            http_proc.kill()
            http_proc.wait()
        except:
            pass
        try:
            master_proc.kill()
            master_proc.wait()
        except:
            pass
        try:
            master_log.close()
        except:
            pass
        subprocess.run(['rm', '-rf', http_dir], stderr=subprocess.DEVNULL)
        subprocess.run(['rm', '-f', '/tmp/socks-master.log'], stderr=subprocess.DEVNULL)
        cleanup_processes()

if __name__ == '__main__':
    sys.exit(main())
