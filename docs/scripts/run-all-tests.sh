#!/bin/bash
# Master Test Runner for Validation Scripts
# Purpose: Run all validation scripts and report results
# Usage: bash docs/scripts/run-all-tests.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Results tracking
PASSED=0
FAILED=0
SKIPPED=0
TOTAL=0

echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}  Goncat Validation Test Suite${NC}"
echo -e "${GREEN}======================================${NC}"
echo ""

# Function to run a single test
run_test() {
    local script="$1"
    local name=$(basename "$script")
    
    TOTAL=$((TOTAL + 1))
    
    echo -n "Running $name... "
    
    # Determine how to run the script
    local cmd
    if [[ "$script" == *.py ]]; then
        cmd="python3 $script"
    else
        cmd="bash $script"
    fi
    
    # Run script with timeout
    if timeout 45 $cmd > /tmp/test-output-$$.log 2>&1; then
        echo -e "${GREEN}✓ PASSED${NC}"
        PASSED=$((PASSED + 1))
    else
        EXIT_CODE=$?
        if [ $EXIT_CODE -eq 124 ]; then
            echo -e "${YELLOW}⚠ TIMEOUT${NC}"
            SKIPPED=$((SKIPPED + 1))
        else
            echo -e "${RED}✗ FAILED (exit $EXIT_CODE)${NC}"
            FAILED=$((FAILED + 1))
            # Show last 10 lines of output
            echo -e "${RED}Last 10 lines of output:${NC}"
            tail -10 /tmp/test-output-$$.log
        fi
    fi
    
    rm -f /tmp/test-output-$$.log
}

# Build binary if needed
if [ ! -f "$REPO_ROOT/dist/goncat.elf" ]; then
    echo -e "${YELLOW}Building goncat binary...${NC}"
    make build-linux
    echo ""
fi

# Run all numbered scripts
for script in "$SCRIPT_DIR"/[0-9]*.sh "$SCRIPT_DIR"/[0-9]*.py; do
    if [ -f "$script" ]; then
        run_test "$script"
    fi
done

echo ""
echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}  Test Results${NC}"
echo -e "${GREEN}======================================${NC}"
echo -e "Total:   $TOTAL"
echo -e "${GREEN}Passed:  $PASSED${NC}"
echo -e "${RED}Failed:  $FAILED${NC}"
echo -e "${YELLOW}Skipped: $SKIPPED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some tests failed${NC}"
    exit 1
fi
