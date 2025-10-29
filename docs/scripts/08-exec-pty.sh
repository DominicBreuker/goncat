#!/bin/bash
# Validation Script: PTY Mode Execution
# Purpose: Verify --pty flag enables pseudo-terminal mode
# Expected: PTY mode works with interactive features
# Dependencies: python3, pexpect, goncat binary

# Run the Python implementation
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
python3 "$SCRIPT_DIR/08-exec-pty.py"
