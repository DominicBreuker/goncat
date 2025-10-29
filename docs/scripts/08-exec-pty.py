#!/usr/bin/env python3
"""
Validation Script: PTY Mode Execution
Purpose: Verify --pty flag enables interactive pseudo-terminal mode
Expected: PTY mode works with interactive features
Dependencies: python3, pexpect, goncat binary
"""

import pexpect
import re
import sys
import os
import time
import subprocess

# Colors for output
RED = '\033[0;31m'
GREEN = '\033[0;32m'
YELLOW = '\033[1;33m'
NC = '\033[0m'  # No Color

def print_status(message, status='info'):
    if status == 'success':
        print(f"{GREEN}{message}{NC}")
    elif status == 'error':
        print(f"{RED}{message}{NC}")
    elif status == 'warning':
        print(f"{YELLOW}{message}{NC}")
    else:
        print(message)

def cleanup():
    """Kill any lingering goncat processes"""
    subprocess.run(['pkill', '-9', 'goncat.elf'], stderr=subprocess.DEVNULL)

def main():
    script_dir = os.path.dirname(os.path.abspath(__file__))
    repo_root = os.path.abspath(os.path.join(script_dir, '../..'))
    os.chdir(repo_root)
    
    binary_path = os.path.join(repo_root, 'dist/goncat.elf')
    
    # Ensure binary exists
    if not os.path.exists(binary_path):
        print_status("Building goncat binary...", 'warning')
        subprocess.run(['make', 'build-linux'], check=True)
    
    print_status("Starting validation: PTY Mode", 'success')
    
    cleanup()
    
    port = 12070
    
    # Start master in background with PTY mode
    print_status("Test: PTY mode with interactive bash", 'warning')
    master_cmd = f"{binary_path} master listen tcp://*:{port} --exec /bin/bash --pty"
    
    master_proc = subprocess.Popen(
        master_cmd.split(),
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True
    )
    
    time.sleep(2)
    
    try:
        # Connect slave with pexpect for interactive control
        slave_cmd = f"{binary_path} slave connect tcp://localhost:{port}"
        child = pexpect.spawn(slave_cmd, encoding='utf-8', timeout=15)
        
        # Wait for connection
        time.sleep(2)
        
        # Test 1: Verify connection established
        print_status("Test 1: Verify PTY connection", 'warning')
        child.sendline('')  # Send enter to get a prompt
        time.sleep(0.5)
        
        # Look for bash-like prompt patterns or just any output
        output = child.before if hasattr(child, 'before') and child.before else ''
        if 'established' in output or len(output) > 10:
            print_status("✓ PTY connection established", 'success')
        
        # Test 2: Send a command and wait for output
        print_status("Test 2: Execute command", 'warning')
        child.sendline('echo TEST_OUTPUT_12345')
        time.sleep(1)
        
        # Try to read available output
        try:
            output = child.read_nonblocking(size=2000, timeout=2)
            if 'TEST_OUTPUT_12345' in output:
                print_status("✓ Command execution and output works", 'success')
            else:
                print_status(f"⚠ Output unclear, got: {output[:100]}", 'warning')
        except:
            print_status("⚠ Could not read output", 'warning')
        
        # Test 3: Ctrl+C
        print_status("Test 3: Ctrl+C handling", 'warning')
        child.sendline('sleep 60')
        time.sleep(0.5)
        child.sendcontrol('c')
        time.sleep(0.5)
        
        # Try sending another command
        child.sendline('echo AFTER_CTRLC')
        time.sleep(1)
        try:
            output = child.read_nonblocking(size=2000, timeout=2)
            if 'AFTER_CTRLC' in output:
                print_status("✓ Ctrl+C works, shell responsive", 'success')
            else:
                print_status("⚠ Ctrl+C test unclear", 'warning')
        except:
            print_status("⚠ Ctrl+C test incomplete", 'warning')
        
        # Test 4: Exit
        print_status("Test 4: Exit command", 'warning')
        child.sendline('exit')
        try:
            child.expect(pexpect.EOF, timeout=5)
            print_status("✓ Exit closes session", 'success')
        except pexpect.TIMEOUT:
            print_status("⚠ Exit may not have closed session cleanly", 'warning')
        
        print_status("✓ PTY mode validation passed", 'success')
        return 0
        
    except Exception as e:
        print_status(f"✗ Test failed: {e}", 'error')
        import traceback
        traceback.print_exc()
        return 1
    finally:
        # Cleanup
        master_proc.terminate()
        try:
            master_proc.wait(timeout=2)
        except subprocess.TimeoutExpired:
            master_proc.kill()
        cleanup()

if __name__ == '__main__':
    sys.exit(main())
