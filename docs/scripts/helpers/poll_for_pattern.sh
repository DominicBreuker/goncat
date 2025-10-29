#!/bin/bash
# Helper: Poll for a pattern in a file
# Usage: poll_for_pattern <file> <pattern> <timeout_seconds> [<interval_seconds>]
# Returns: 0 if pattern found, 1 if timeout

FILE="$1"
PATTERN="$2"
TIMEOUT="${3:-10}"
INTERVAL="${4:-0.1}"

START=$(date +%s)
while true; do
    if [ -f "$FILE" ] && grep -q "$PATTERN" "$FILE" 2>/dev/null; then
        exit 0
    fi
    
    NOW=$(date +%s)
    if [ $((NOW - START)) -ge "$TIMEOUT" ]; then
        exit 1
    fi
    
    sleep "$INTERVAL"
done
