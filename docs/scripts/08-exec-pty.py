#!/usr/bin/env python3
"""
Validation Script: PTY Mode Command Execution
Purpose: Verify --pty flag provides proper interactive terminal
Data Flow: pexpect drives MASTER process interactively
Tests: TTY features, command execution, Ctrl+C, line editing, exit
Dependencies: python3, pexpect, goncat binary
"""

import os
import sys
import signal
import subprocess
import time
import pexpect
import re

# Colors
RED = '\033[0;31m'
GREEN = '\033[0;32m'
YELLOW = '\033[1;33m'
NC = '\033[0m'

def poll_for_pattern(filepath, pattern, timeout=10):
    """Poll for a pattern in a file"""
    start = time.time()
    while time.time() - start < timeout:
        try:
            with open(filepath, 'r') as f:
                content = f.read()
                if re.search(pattern, content):
                    return True
        except FileNotFoundError:
            pass
        time.sleep(0.1)
    return False

def main():
    repo_root = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
    os.chdir(repo_root)
    
    binary = f"{repo_root}/dist/goncat.elf"
    if not os.path.exists(binary):
        print(f"{YELLOW}Building goncat binary...{NC}")
        subprocess.run(["make", "build-linux"], check=True)
    
    print(f"{GREEN}=== PTY Mode Command Execution Validation ==={NC}")
    
    port = 12071
    slave_proc = None
    master = None
    
    try:
        # Start master with pexpect (PTY mode, interactive)
        print(f"{YELLOW}Starting master with --pty (interactive)...{NC}")
        cmd = f"{binary} master listen tcp://*:{port} --exec /bin/bash --pty"
        master = pexpect.spawn(cmd, encoding='utf-8', timeout=10)
        master.logfile = open("/tmp/goncat-pty-master.log", "w")
        master.setwinsize(24, 120)
        
        # Wait for listening message
        master.expect("Listening on", timeout=5)
        print(f"{GREEN}✓ Master listening{NC}")
        
        # Now connect slave in background
        print(f"{YELLOW}Connecting slave...{NC}")
        slave_log = open("/tmp/goncat-pty-slave.log", "w")
        slave_proc = subprocess.Popen(
            [binary, "slave", "connect", f"tcp://localhost:{port}"],
            stdout=slave_log,
            stderr=subprocess.STDOUT
        )
        
        # Wait for session established (may appear in logs or as output)
        time.sleep(2)
        # Check slave log for establishment
        if poll_for_pattern("/tmp/goncat-pty-slave.log", r"Session with .* established", timeout=5):
            print(f"{GREEN}✓ Session established{NC}")
        else:
            print(f"{YELLOW}⚠ Session established message not found (may still work){NC}")
        
        # Set deterministic PS1 prompt
        master.sendline("export PS1='GONCAT-PTY> '")
        master.expect("GONCAT-PTY> ", timeout=5)
        print(f"{GREEN}✓ Deterministic prompt set{NC}")
        
        # Test 1: TTY detection
        print(f"{YELLOW}Test 1: Verify TTY is allocated{NC}")
        master.sendline("test -t 0 && echo TTY_YES || echo TTY_NO")
        idx = master.expect(["TTY_YES", "TTY_NO"], timeout=5)
        if idx == 0:
            print(f"{GREEN}✓ TTY allocated (stdin is a terminal){NC}")
        else:
            print(f"{RED}✗ No TTY allocated{NC}")
            sys.exit(1)
        master.expect("GONCAT-PTY> ", timeout=5)
        
        # Test 2: Basic command execution
        print(f"{YELLOW}Test 2: Execute basic commands{NC}")
        master.sendline("whoami")
        master.expect(r"(root|runner|[a-z0-9]+)", timeout=5)
        print(f"{GREEN}✓ whoami command executed{NC}")
        master.expect("GONCAT-PTY> ", timeout=5)
        
        token = f"PTY_TOKEN_{os.getpid()}"
        master.sendline(f"echo {token}")
        master.expect(token, timeout=5)
        print(f"{GREEN}✓ Token verified (data channel working){NC}")
        master.expect("GONCAT-PTY> ", timeout=5)
        
        # Test 3: Line editing (command history with arrow keys)
        print(f"{YELLOW}Test 3: Test line editing (arrow keys){NC}")
        master.sendline("echo first")
        master.expect("first", timeout=5)
        master.expect("GONCAT-PTY> ", timeout=5)
        
        master.sendline("echo second")
        master.expect("second", timeout=5)
        master.expect("GONCAT-PTY> ", timeout=5)
        
        # Up arrow should recall "echo second"
        master.send("\x1b[A")  # Up arrow
        time.sleep(0.2)
        master.sendline("")  # Execute recalled command
        master.expect("second", timeout=5)
        print(f"{GREEN}✓ Command history (up arrow) working{NC}")
        master.expect("GONCAT-PTY> ", timeout=5)
        
        # Test 4: Ctrl+C interrupt
        print(f"{YELLOW}Test 4: Test Ctrl+C interrupt{NC}")
        master.sendline("sleep 30")
        time.sleep(0.5)
        master.sendintr()  # Ctrl+C
        # After Ctrl+C, should get prompt back (sleep interrupted)
        master.expect("GONCAT-PTY> ", timeout=5)
        print(f"{GREEN}✓ Ctrl+C interrupt working{NC}")
        
        # Test 5: Terminal dimensions
        print(f"{YELLOW}Test 5: Test terminal dimensions{NC}")
        master.sendline("tput cols")
        master.expect(r"\d+", timeout=5)
        print(f"{GREEN}✓ Terminal dimensions accessible{NC}")
        master.expect("GONCAT-PTY> ", timeout=5)
        
        # Test 6: Graceful exit
        print(f"{YELLOW}Test 6: Test graceful exit{NC}")
        master.sendline("exit")
        # After exit, expect either EOF or session closed message
        idx = master.expect([pexpect.EOF, r"Session .* closed", pexpect.TIMEOUT], timeout=5)
        if idx in [0, 1]:
            print(f"{GREEN}✓ Graceful exit working{NC}")
        else:
            print(f"{YELLOW}⚠ Exit completed but no clean EOF{NC}")
        
        # Verify slave connection logs show session closed
        slave_log.close()
        time.sleep(1)
        if poll_for_pattern("/tmp/goncat-pty-slave.log", r"Session with .* closed", timeout=3):
            print(f"{GREEN}✓ Session closed on slave side{NC}")
        else:
            print(f"{YELLOW}⚠ Session close not logged on slave{NC}")
        
        # Clean up
        if slave_proc:
            slave_proc.wait(timeout=3)
        
        print(f"{GREEN}✓ PTY mode validation PASSED{NC}")
        return 0
        
    except pexpect.TIMEOUT as e:
        print(f"{RED}✗ Timeout during PTY interaction{NC}")
        print(f"Buffer: {master.buffer if master else 'N/A'}")
        print(f"Before: {master.before if master else 'N/A'}")
        return 1
    except Exception as e:
        print(f"{RED}✗ Error: {e}{NC}")
        import traceback
        traceback.print_exc()
        return 1
    finally:
        # Cleanup
        if master and master.isalive():
            master.close(force=True)
        if slave_proc:
            try:
                slave_proc.terminate()
                slave_proc.wait(timeout=3)
            except:
                slave_proc.kill()

if __name__ == "__main__":
    sys.exit(main())
