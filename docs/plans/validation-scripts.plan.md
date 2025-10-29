# Plan for Validation Scripts

Create a comprehensive collection of standalone bash validation scripts for GitHub Copilot Coding Agent to verify goncat functionality. These scripts will enable fast, automated verification of core features and UX behaviors without requiring elaborate infrastructure.

## Overview

The goal is to create validated bash scripts that GitHub Copilot Coding Agent can run to verify goncat behavior. Scripts must be:
- Standalone (no Docker, work on localhost)
- Fast (complete in seconds unless testing stability)
- Simple enough for agent modification
- Comprehensive (real depth, not shallow flag checking)
- Explicitly designed for agent environment compatibility

Scripts will be organized in `docs/scripts/` with an overview document `docs/VALIDATION.md` linking to them. Each script validates a specific feature or UX aspect across different transports.

## Implementation plan

- [X] Step 1: Create validation documentation structure
  - **Task**: Set up the validation documentation and scripts directory structure
  - **Files**: 
    - `docs/VALIDATION.md`: Create overview document with:
      - Header explaining these are agent-compatible scripts
      - Grep hint: `cat VALIDATION.md | grep '- \*\*Scenario:'`
      - Short introduction (keep under 2-3 pages)
      - List of all validation scenarios with brief descriptions
      - Link to each script
      - Note about potentially unsupported scenarios
    - `docs/scripts/`: Create directory for validation scripts
  - **Dependencies**: None
  - **Definition of done**: Directory structure exists, VALIDATION.md has header and structure ready for scenario additions
  - **Completed**: Created docs/VALIDATION.md with agent-compatible header, grep hint, usage instructions, troubleshooting section. Created docs/scripts/ and docs/scripts/helpers/ directories.

- [X] Step 2: Create transport verification scripts
  - **Task**: Create validation scripts for each transport protocol (tcp, ws, wss, udp)
  - **Files**:
    - `docs/scripts/01-transport-tcp.sh`: Validate basic TCP transport
      - Master listen on tcp, slave connect, echo test, verify data transfer
      - Test both master-listen-slave-connect and slave-listen-master-connect
    - `docs/scripts/02-transport-ws.sh`: Validate WebSocket transport
      - Master listen on ws, slave connect, echo test
    - `docs/scripts/03-transport-wss.sh`: Validate WebSocket Secure transport
      - Master listen on wss, slave connect, echo test with TLS
    - `docs/scripts/04-transport-udp.sh`: Validate UDP/QUIC transport
      - Master listen on udp, slave connect, echo test
  - **Script pattern**:
    ```bash
    #!/bin/bash
    # Transport: TCP validation
    # Purpose: Verify basic TCP transport works for master-listen and slave-listen modes
    # Expected: Data transfers successfully in both directions
    
    set -e
    cd "$(dirname "$0")/../.."
    
    # Build if needed
    if [ ! -f dist/goncat.elf ]; then
        make build-linux
    fi
    
    # Test 1: Master listen, slave connect
    # Start master in background
    # Start slave, exchange data
    # Verify output
    
    # Test 2: Slave listen, master connect
    # Similar pattern
    
    echo "✓ TCP transport validation passed"
    ```
  - **Dependencies**: Step 1 complete
  - **Definition of done**: 4 transport scripts exist, each validates basic connectivity and bidirectional data exchange for its protocol
  - **Completed**: Created all 4 transport validation scripts. Each script verifies connection establishment and data transfer. TCP script tests both master-listen and slave-listen modes. All scripts tested and passing.

- [X] Step 3: Create encryption and authentication scripts
  - **Task**: Validate --ssl and --key functionality across transports
  - **Files**:
    - `docs/scripts/05-encryption-ssl.sh`: Validate --ssl flag
      - Test with tcp, ws, wss protocols
      - Verify TLS handshake succeeds
      - Verify data transfer works encrypted
      - Test that connection fails without --ssl on one side
    - `docs/scripts/06-authentication-key.sh`: Validate --key mutual authentication
      - Test with correct password on both sides (success)
      - Test with mismatched passwords (failure expected)
      - Test --key requires --ssl (error expected without --ssl)
      - Verify certificate validation occurs
  - **Script depth**: Must actually verify TLS connection (not just that flag is accepted)
  - **Dependencies**: Step 2 complete
  - **Definition of done**: Scripts verify encryption works, authentication succeeds/fails appropriately
  - **Completed**: Created both scripts. 05-encryption-ssl.sh tests TLS with TCP and WebSocket, verifies mismatched SSL fails. 06-authentication-key.sh tests matching/mismatched passwords and validates --key requires --ssl. All tests passing.

- [X] Step 4: Create command execution scripts
  - **Task**: Validate --exec flag with and without --pty
  - **Files**:
    - `docs/scripts/07-exec-simple.sh`: Validate simple command execution (no PTY)
      - Execute `/bin/sh -c "echo test"` and verify output
      - Execute multiple commands in sequence
      - Test command exit codes propagate correctly
    - `docs/scripts/08-exec-pty.sh`: Validate PTY mode (if supported in agent environment)
      - Test --pty with /bin/bash
      - Verify interactive features work (or mark unsupported if PTY unavailable)
      - Test terminal size synchronization if possible
      - Send simple commands and verify output
  - **Environment consideration**: PTY may not work in agent environment - script should detect and mark as unsupported if needed
  - **Dependencies**: Step 2 complete
  - **Definition of done**: Scripts validate command execution, with real output verification
  - **Completed**: Created both scripts. 07-exec-simple.sh validates basic command execution. 08-exec-pty.sh validates PTY mode with TTY detection (marks unsupported if no TTY available). Both passing.

- [ ] Step 5: Create port forwarding scripts
  - **Task**: Validate local (-L) and remote (-R) port forwarding for TCP and UDP
  - **Files**:
    - `docs/scripts/09-portfwd-local-tcp.sh`: Local TCP port forwarding
      - Start simple HTTP server (python3 -m http.server)
      - Forward local port through goncat tunnel
      - Use curl to fetch through tunnel
      - Verify HTTP response matches expected
    - `docs/scripts/10-portfwd-local-udp.sh`: Local UDP port forwarding
      - Start simple UDP echo server
      - Forward local UDP port through tunnel
      - Send UDP packet, verify echo response
    - `docs/scripts/11-portfwd-remote-tcp.sh`: Remote TCP port forwarding
      - Start HTTP server on master side
      - Forward remote port on slave back to master
      - Connect from slave side, verify response
    - `docs/scripts/12-portfwd-remote-udp.sh`: Remote UDP port forwarding
      - Similar pattern with UDP echo server
    - `docs/scripts/13-portfwd-multiple.sh`: Multiple simultaneous forwards
      - Test multiple -L and -R flags together (mixed TCP/UDP)
  - **Real depth**: Must verify actual data transfer through forwards, not just port opening
  - **Dependencies**: Step 2 complete
  - **Definition of done**: Scripts verify data flows through port forwards correctly

- [ ] Step 6: Create SOCKS proxy scripts
  - **Task**: Validate SOCKS5 proxy functionality (-D flag) for TCP and UDP
  - **Files**:
    - `docs/scripts/14-socks-tcp-connect.sh`: SOCKS TCP CONNECT
      - Start goncat with -D flag
      - Start HTTP server accessible via tunnel
      - Use curl with --socks5 to access server
      - Verify HTTP response received
    - `docs/scripts/15-socks-udp-associate.sh`: SOCKS UDP ASSOCIATE
      - Start goncat with -D flag
      - Use SOCKS for UDP (if supported in test environment)
      - Test DNS query or UDP echo through SOCKS
      - Mark unsupported if UDP ASSOCIATE not testable in environment
  - **Dependencies**: Step 2 complete
  - **Definition of done**: Scripts verify SOCKS proxy routes traffic correctly

- [ ] Step 7: Create connection behavior scripts
  - **Task**: Validate connection lifecycle behaviors (close, reconnect, timeout)
  - **Files**:
    - `docs/scripts/16-behavior-connect-close.sh`: Connection close behavior
      - In connect mode: verify tool exits when connection closes
      - In listen mode: verify tool continues running after connection closes
      - Test graceful close vs abrupt close
    - `docs/scripts/17-behavior-timeout.sh`: Timeout flag verification
      - Test --timeout flag honored during connection attempts
      - Test connection drops after timeout when one side disappears
      - Test TLS handshake timeout
      - **Critical**: Test with very short timeout (100ms) that connections still work (no uncanceled timeouts)
    - `docs/scripts/18-behavior-stability.sh`: Connection stability
      - Establish connection with short --timeout (100ms)
      - Keep connection alive for 10+ seconds
      - Verify connection remains stable (no premature drops)
      - Exchange data periodically to confirm liveness
    - `docs/scripts/19-behavior-graceful-shutdown.sh`: CTRL+C handling
      - Start master and slave
      - Send SIGINT to one side
      - Verify graceful shutdown occurs
      - Verify other side detects shutdown and exits
  - **Dependencies**: Step 2 complete
  - **Definition of done**: Scripts verify connection lifecycle behaviors match expectations

- [ ] Step 8: Create session logging script
  - **Task**: Validate --log flag functionality
  - **Files**:
    - `docs/scripts/20-feature-logging.sh`: Session logging
      - Start master with --log flag
      - Execute commands through slave
      - Verify log file created
      - Verify log contains session data
      - Clean up log file
  - **Dependencies**: Step 2 complete
  - **Definition of done**: Script verifies log file is created with session content

- [ ] Step 9: Create cleanup script
  - **Task**: Validate --cleanup flag (slave self-deletion)
  - **Files**:
    - `docs/scripts/21-feature-cleanup.sh`: Self-deletion
      - Copy goncat binary to temp location
      - Run slave with --cleanup from temp location
      - After connection ends, verify binary is deleted
      - Note: May only work fully on Linux
  - **Dependencies**: Step 2 complete
  - **Definition of done**: Script verifies binary deletion on supported platforms

- [ ] Step 10: Create helper script for parameterized transport testing
  - **Task**: Create a helper script that can run validation scenarios across all transports
  - **Files**:
    - `docs/scripts/helpers/run-across-transports.sh`: Helper script
      - Accepts scenario script name as argument
      - Runs scenario for tcp, ws, wss, udp transports
      - Reports success/failure for each
    - `docs/scripts/helpers/echo-server.sh`: Simple echo server helper
      - TCP and UDP variants
      - Used by multiple test scripts
    - `docs/scripts/helpers/cleanup.sh`: Process cleanup helper
      - Kill all goncat processes
      - Clean up temp files/ports
      - Used by all test scripts
  - **Dependencies**: Steps 2-9 complete
  - **Definition of done**: Helper scripts exist and are used by test scripts

- [ ] Step 11: Update VALIDATION.md with all scenarios
  - **Task**: Complete the VALIDATION.md documentation with all scenarios
  - **Files**:
    - `docs/VALIDATION.md`: Update with complete scenario list
      - Add each scenario with: name, purpose, script path, expected outcome
      - Add usage instructions for agents
      - Add troubleshooting section
      - Keep document concise (2-3 pages max)
  - **Scenario list format**:
    ```markdown
    - **Scenario: TCP Transport**: Validates basic TCP connectivity and data transfer. See `scripts/01-transport-tcp.sh`
    - **Scenario: WebSocket Transport**: Validates WebSocket protocol works. See `scripts/02-transport-ws.sh`
    ...
    ```
  - **Dependencies**: Steps 2-10 complete
  - **Definition of done**: VALIDATION.md contains complete list of all scenarios with descriptions and links

- [ ] Step 12: Manual verification of validation scripts
  - **Task**: Manually run each validation script to ensure they work correctly
  - **Process**:
    1. Build goncat binary: `make build-linux`
    2. Run each script in `docs/scripts/` directory
    3. Verify each script exits with code 0 on success
    4. Verify output is clear and informative
    5. Fix any issues discovered
    6. For each script run:
       ```bash
       cd /home/runner/work/goncat/goncat
       bash docs/scripts/01-transport-tcp.sh
       echo "Exit code: $?"
       ```
  - **CRITICAL**: This step is **MANDATORY** and **CANNOT BE SKIPPED**
  - **If scripts fail**: Debug and fix issues, do not just mark them as unsupported unless truly blocked by environment
  - **If truly blocked**: Mark scenario as unsupported in VALIDATION.md with clear explanation
  - **Dependencies**: Steps 2-11 complete
  - **Definition of done**: All scripts run successfully OR are marked as unsupported with explanation. Clear verification output showing each script's result.

- [ ] Step 13: Create README in scripts directory
  - **Task**: Add README to scripts directory for documentation
  - **Files**:
    - `docs/scripts/README.md`: Brief overview
      - Purpose of validation scripts
      - How to run scripts
      - Expected environment
      - Link back to VALIDATION.md
  - **Dependencies**: Step 12 complete
  - **Definition of done**: README exists and explains scripts directory

- [ ] Step 14: Run linters and tests
  - **Task**: Ensure no code changes broke existing functionality
  - **Process**:
    ```bash
    make lint
    make test-unit
    make test-integration
    ```
  - **Note**: This task involves only documentation and scripts, no code changes expected
  - **Dependencies**: All previous steps complete
  - **Definition of done**: All linters and tests pass

## Notes for Implementation

### Script Template

Each validation script should follow this structure:

```bash
#!/bin/bash
# Validation Script: [Name]
# Purpose: [What this validates]
# Expected: [Expected behavior]
# Dependencies: [Any external tools needed: python3, curl, etc.]

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
    rm -f /tmp/goncat-test-*
}
trap cleanup EXIT

# Ensure binary exists
if [ ! -f "$REPO_ROOT/dist/goncat.elf" ]; then
    echo -e "${YELLOW}Building goncat binary...${NC}"
    make build-linux
fi

# Test implementation goes here
echo -e "${GREEN}Starting validation: [Name]${NC}"

# Test steps...

echo -e "${GREEN}✓ [Name] validation passed${NC}"
exit 0
```

### Environment Compatibility Notes

1. **PTY support**: May not work in headless environments - detect and mark unsupported if needed
2. **Port availability**: Use high port numbers (12000+) to avoid conflicts
3. **Process cleanup**: Always clean up background processes in trap
4. **Timeout values**: Keep reasonable for fast execution (2-5 seconds usually sufficient)
5. **Binary location**: Use relative path from script to repo root

### Data Depth Requirements

Scripts must verify actual functionality, not just that flags are accepted:
- Port forwarding: Actually transfer HTTP data through tunnel and verify response
- SOCKS: Actually route traffic through proxy and verify response
- Encryption: Verify connection fails with mismatched TLS settings
- Authentication: Verify connection fails with wrong password
- Execution: Verify command output is correct

### Marking Scenarios as Unsupported

If a scenario truly cannot be tested in the agent environment:

1. Create the script anyway with detection logic
2. Have script output clear message: `echo "UNSUPPORTED: [reason]"`
3. Exit with code 0 (not failure)
4. Document in VALIDATION.md: `- **Scenario: [Name]** (UNSUPPORTED: [reason])`

Example:
```bash
if ! command -v pty-test-tool &> /dev/null; then
    echo "UNSUPPORTED: PTY testing requires pty-test-tool which is not available"
    exit 0
fi
```

### Bug Discovery

If during implementation a bug is suspected in goncat:
1. Do NOT attempt to fix the bug
2. Add note in script: `# BUG SUSPECTED: [description]`
3. Document in VALIDATION.md: `- **Scenario: [Name]** (bug suspected: [description])`
4. Continue with other scenarios

### Transport Parameterization

Many scripts should accept transport as argument to test across all protocols:

```bash
TRANSPORT="${1:-tcp}"  # Default to tcp if not specified

# Use like: tcp://*:12345 or ws://*:12345
LISTEN_ADDR="${TRANSPORT}://*:12345"
CONNECT_ADDR="${TRANSPORT}://localhost:12345"
```

This enables running: `./script.sh tcp`, `./script.sh ws`, etc.

## Validation Checklist

After implementation, verify:
- [ ] All scripts are executable (`chmod +x docs/scripts/*.sh`)
- [ ] All scripts follow the template structure
- [ ] All scripts have proper cleanup (trap)
- [ ] All scripts produce clear pass/fail output
- [ ] All scripts are documented in VALIDATION.md
- [ ] VALIDATION.md has grep hint at top
- [ ] Scripts run in reasonable time (< 10 seconds each, except stability tests)
- [ ] Scripts verify real functionality (not shallow)
- [ ] Helper scripts exist and are reusable
- [ ] Manual verification completed for all scripts
- [ ] Linters and tests still pass
