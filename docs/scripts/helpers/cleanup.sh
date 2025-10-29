#!/bin/bash
# Helper: Cleanup goncat processes and temp files
# Purpose: Clean up all goncat processes and test files
# Usage: source this script or run directly

pkill -9 goncat.elf 2>/dev/null || true
pkill -9 python3 2>/dev/null || true  # Kill test servers
rm -f /tmp/goncat-test-* 2>/dev/null || true

echo "Cleanup complete"
