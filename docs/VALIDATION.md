# Validation Scripts for GitHub Copilot Coding Agent

> **Agent-Compatible Validation Scripts**: These validation scripts have been explicitly designed and verified for use with GitHub Copilot Coding Agent. The scripts run in a compatible environment and provide fast, reliable validation of goncat functionality without requiring Docker or complex infrastructure.

**Quick Scenario Listing**: To list all validation scenarios, run:
```bash
cat docs/VALIDATION.md | grep '- \*\*Scenario:'
```

## Overview

This directory contains standalone bash validation scripts that verify goncat's core functionality and user experience behaviors. Each script:

- **Runs standalone** on localhost without Docker or external infrastructure
- **Completes quickly** (typically under 10 seconds, except stability tests)
- **Verifies real functionality** with actual data transfer, not just flag acceptance
- **Provides clear output** with success/failure indication
- **Handles cleanup** automatically via trap handlers
- **Can be modified** easily by agents for debugging or adaptation

## Purpose

These scripts enable GitHub Copilot Coding Agent to:
1. Verify that goncat features work correctly after code changes
2. Validate user experience behaviors (connection lifecycle, timeouts, etc.)
3. Test functionality across different transport protocols
4. Identify bugs or regressions quickly
5. Confirm fixes work as expected

## Usage

### Running Individual Scripts

```bash
# From repository root
cd /home/runner/work/goncat/goncat

# Run a specific validation script
bash docs/scripts/01-transport-tcp.sh

# Check exit code
echo $?  # 0 = success, non-zero = failure
```

### Running All Scripts

```bash
# Run all validation scripts
for script in docs/scripts/[0-9]*.sh; do
    echo "Running: $script"
    bash "$script" || echo "FAILED: $script"
done
```

### Transport-Parameterized Scripts

Some scripts accept a transport protocol as an argument:

```bash
# Test with specific transport
bash docs/scripts/01-transport-tcp.sh tcp
bash docs/scripts/01-transport-tcp.sh ws
bash docs/scripts/01-transport-tcp.sh wss
bash docs/scripts/01-transport-tcp.sh udp
```

## Validation Scenarios

The following scenarios are validated by these scripts:

- **Scenario: TCP Transport**: Validates basic TCP connectivity and bidirectional data transfer. Tests both master-listen-slave-connect and slave-listen-master-connect modes. See `scripts/01-transport-tcp.sh`
- **Scenario: WebSocket Transport**: Validates WebSocket protocol connection establishment and data transfer. See `scripts/02-transport-ws.sh`
- **Scenario: WebSocket Secure Transport**: Validates WSS (WebSocket Secure) with TLS encryption. See `scripts/03-transport-wss.sh`
- **Scenario: UDP/QUIC Transport**: Validates UDP transport with QUIC protocol for reliable streaming. See `scripts/04-transport-udp.sh`
- **Scenario: TLS Encryption**: Validates --ssl flag enables TLS across transports. Tests both successful encryption and failure when SSL flags don't match. See `scripts/05-encryption-ssl.sh`
- **Scenario: Mutual Authentication**: Validates --key flag provides password-based mutual authentication. Tests matching passwords (success), mismatched passwords (failure), and requirement for --ssl. See `scripts/06-authentication-key.sh`
- **Scenario: Simple Command Execution**: Validates --exec flag executes commands correctly without PTY. See `scripts/07-exec-simple.sh`
- **Scenario: PTY Mode**: Validates --pty flag for interactive pseudo-terminal support. Detects TTY availability and marks unsupported if not available. See `scripts/08-exec-pty.sh`
- **Scenario: Local TCP Port Forwarding**: Validates -L flag forwards local TCP ports through tunnel. Tests with HTTP server and curl. See `scripts/09-portfwd-local-tcp.sh`
- **Scenario: Connection Close Behavior**: Validates that listen mode continues after connection closes and can accept new connections. See `scripts/16-behavior-connect-close.sh`
- **Scenario: Session Logging**: Validates --log flag creates session log files. See `scripts/20-feature-logging.sh`

<!-- Additional scenarios will be added as more scripts are implemented -->

## Environment Requirements

### Supported Platforms
- Linux (primary target, all scripts tested)
- macOS (should work, not all scripts tested)
- WSL on Windows (should work, not all scripts tested)

### Required Tools
- `bash` (4.0+)
- `python3` (for test HTTP/UDP servers)
- `curl` (for HTTP testing)
- `make` (for building goncat)
- `go` (1.23+ for building)

### Optional Tools
- `nc` (netcat) - for some UDP tests
- `dig` - for DNS testing via SOCKS proxy

## Unsupported Scenarios

Some scenarios may be marked as unsupported if they cannot be validated in the agent's environment:

<!-- Unsupported scenarios will be documented as discovered -->

## Troubleshooting

### Script Fails with "Port already in use"

Kill any lingering goncat processes:
```bash
pkill -9 goncat.elf
```

### Script Fails with "Binary not found"

Build the binary first:
```bash
make build-linux
```

### Script Hangs

Kill all background processes and retry:
```bash
pkill -9 goncat.elf
pkill -9 python3
```

### General Debugging

Enable bash tracing to see what's happening:
```bash
bash -x docs/scripts/01-transport-tcp.sh
```

## Bug Discovery

If a script suspects a bug in goncat, it will be documented here:

<!-- Bug discoveries will be noted as found -->

## Contributing

When adding new validation scripts:

1. Follow the template in `docs/plans/validation-scripts.plan.md`
2. Include proper cleanup via trap handlers
3. Test the script manually before committing
4. Update this document with the new scenario
5. Ensure script completes in < 10 seconds (except stability tests)
6. Verify real functionality, not just flag acceptance

## Script Organization

```
docs/scripts/
├── 01-transport-tcp.sh          # TCP transport validation
├── 02-transport-ws.sh           # WebSocket transport validation
├── 03-transport-wss.sh          # WebSocket Secure validation
├── 04-transport-udp.sh          # UDP/QUIC transport validation
├── 05-encryption-ssl.sh         # TLS encryption validation
├── 06-authentication-key.sh     # Mutual authentication validation
├── 07-exec-simple.sh            # Command execution validation
├── 08-exec-pty.sh               # PTY mode validation
├── 09-portfwd-local-tcp.sh      # Local TCP port forwarding
├── 10-portfwd-local-udp.sh      # Local UDP port forwarding
├── 11-portfwd-remote-tcp.sh     # Remote TCP port forwarding
├── 12-portfwd-remote-udp.sh     # Remote UDP port forwarding
├── 13-portfwd-multiple.sh       # Multiple simultaneous forwards
├── 14-socks-tcp-connect.sh      # SOCKS TCP CONNECT
├── 15-socks-udp-associate.sh    # SOCKS UDP ASSOCIATE
├── 16-behavior-connect-close.sh # Connection close behavior
├── 17-behavior-timeout.sh       # Timeout handling
├── 18-behavior-stability.sh     # Connection stability
├── 19-behavior-graceful-shutdown.sh # Graceful shutdown
├── 20-feature-logging.sh        # Session logging
├── 21-feature-cleanup.sh        # Self-deletion cleanup
├── helpers/
│   ├── cleanup.sh               # Process cleanup utilities
│   ├── echo-server.sh           # Test echo servers
│   └── run-across-transports.sh # Transport test runner
└── README.md                    # This directory overview
```

## Quick Reference

| Script | Purpose | Duration | Dependencies |
|--------|---------|----------|--------------|
| Coming soon... | | | |

---

*This validation suite was designed specifically for GitHub Copilot Coding Agent to enable fast, reliable functionality verification.*
