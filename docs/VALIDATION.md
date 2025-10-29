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

### Fully Validated Scripts (✅ TESTED & PASSING)

- **Scenario: TCP Transport**: Validates basic TCP connectivity and bidirectional data transfer with unique token verification in master output. **Status: ✅ PASSING**. See `scripts/01-transport-tcp.sh`
- **Scenario: WebSocket Transport**: Validates WebSocket protocol connection establishment and data transfer with token validation. **Status: ✅ PASSING**. See `scripts/02-transport-ws.sh`
- **Scenario: WebSocket Secure Transport**: Validates WSS (WebSocket Secure) with TLS encryption and data transfer. **Status: ✅ PASSING**. See `scripts/03-transport-wss.sh`
- **Scenario: UDP/QUIC Transport**: Validates UDP transport with multi-line payload for segmentation testing. **Status: ✅ PASSING**. See `scripts/04-transport-udp.sh`
- **Scenario: TLS Encryption**: Validates --ssl flag with 6 test cases: SSL success, SSL mismatches (master/slave only), mTLS success, mTLS wrong key, key-without-SSL error. **Status: ✅ PASSING**. See `scripts/05-encryption-ssl.sh`
- **Scenario: Mutual Authentication**: Validates --key flag with 3 test cases: mTLS success, wrong key failure, key-without-SSL error. **Status: ✅ PASSING**. See `scripts/06-authentication-key.sh`
- **Scenario: Simple Command Execution**: Validates --exec flag executes commands (whoami, id) with token verification in master output. Tests listener persistence. **Status: ✅ PASSING**. See `scripts/07-exec-simple.sh`
- **Scenario: PTY Mode**: Validates --pty flag with pexpect testing: TTY allocation, command execution, line editing (arrow keys), Ctrl+C interrupt, terminal dimensions, graceful exit. **Status: ✅ PASSING**. See `scripts/08-exec-pty.py`
- **Scenario: Local TCP Port Forwarding**: Validates -L flag forwards local TCP ports. Tests unique token through HTTP tunnel, persistence, and teardown. **Status: ✅ PASSING**. See `scripts/09-portfwd-local-tcp.sh`

### Refactored Scripts (⚠️ NEEDS FINAL TESTING)

- **Scenario: SOCKS TCP CONNECT**: Validates -D flag creates SOCKS5 proxy for TCP connections. Tests unique token through SOCKS, persistence, and teardown. **Status: ⚠️ REFACTORED, NEEDS TESTING**. See `scripts/14-socks-tcp-connect.sh`
- **Scenario: Connection Close Behavior**: Validates listen mode persists after close, accepts new connections. Tests connect mode exits. **Status: ⚠️ REFACTORED, NEEDS TESTING**. See `scripts/16-behavior-connect-close.sh`
- **Scenario: Timeout Handling**: Validates --timeout flag with normal connections and timeout detection when connection dies (SIGKILL). **Status: ⚠️ REFACTORED, NEEDS TESTING**. See `scripts/17-behavior-timeout.sh`
- **Scenario: Connection Stability**: Validates connections work correctly with 100ms timeout for 3+ seconds without false timeouts. **Status: ⚠️ REFACTORED, NEEDS TESTING**. See `scripts/18-behavior-stability.sh`
- **Scenario: Graceful Shutdown**: Validates graceful shutdown via SIGINT propagates to both sides, master continues in listen mode. **Status: ⚠️ REFACTORED, NEEDS TESTING**. See `scripts/19-behavior-graceful-shutdown.sh`
- **Scenario: Session Logging**: Validates --log flag creates session log files with actual data, tests multiple sessions. **Status: ⚠️ REFACTORED, NEEDS TESTING**. See `scripts/20-feature-logging.sh`

### Future Work (Not Implemented)

- **Scenario: Local UDP Port Forwarding**: Not yet implemented
- **Scenario: Remote TCP Port Forwarding**: Not yet implemented
- **Scenario: Remote UDP Port Forwarding**: Not yet implemented
- **Scenario: Multiple Simultaneous Forwards**: Not yet implemented
- **Scenario: SOCKS UDP ASSOCIATE**: Not yet implemented
- **Scenario: Self-Deletion Cleanup**: Not yet implemented

<!-- Additional scenarios can be added following the established pattern -->

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
├── 01-transport-tcp.sh          # ✅ TCP transport validation
├── 02-transport-ws.sh           # ✅ WebSocket transport validation
├── 03-transport-wss.sh          # ✅ WebSocket Secure validation
├── 04-transport-udp.sh          # ✅ UDP/QUIC transport validation
├── 05-encryption-ssl.sh         # ✅ TLS encryption validation (6 test cases)
├── 06-authentication-key.sh     # ✅ Mutual authentication validation (3 test cases)
├── 07-exec-simple.sh            # ✅ Command execution validation
├── 08-exec-pty.py               # ✅ PTY mode validation (pexpect)
├── 09-portfwd-local-tcp.sh      # ✅ Local TCP port forwarding
├── 14-socks-tcp-connect.sh      # ⚠️ SOCKS TCP CONNECT (needs testing)
├── 16-behavior-connect-close.sh # ⚠️ Connection close behavior (needs testing)
├── 17-behavior-timeout.sh       # ⚠️ Timeout handling (needs testing)
├── 18-behavior-stability.sh     # ⚠️ Connection stability (needs testing)
├── 19-behavior-graceful-shutdown.sh # ⚠️ Graceful shutdown (needs testing)
├── 20-feature-logging.sh        # ⚠️ Session logging (needs testing)
├── helpers/
│   ├── cleanup.sh               # Process cleanup utilities
│   └── poll_for_pattern.sh      # Polling helper for log checking
├── REWRITE_PLAN.md              # Refactoring documentation
└── README.md                    # Scripts directory overview

Future work (not implemented):
├── 10-portfwd-local-udp.sh      # Local UDP port forwarding
├── 11-portfwd-remote-tcp.sh     # Remote TCP port forwarding
├── 12-portfwd-remote-udp.sh     # Remote UDP port forwarding
├── 13-portfwd-multiple.sh       # Multiple simultaneous forwards
├── 15-socks-udp-associate.sh    # SOCKS UDP ASSOCIATE
└── 21-feature-cleanup.sh        # Self-deletion cleanup
```

## Quick Reference

| Script | Purpose | Status | Duration | Key Features |
|--------|---------|--------|----------|--------------|
| 01-transport-tcp.sh | TCP transport | ✅ PASSING | ~5s | Token validation, listener persistence |
| 02-transport-ws.sh | WebSocket | ✅ PASSING | ~5s | WS protocol, token validation |
| 03-transport-wss.sh | WebSocket Secure | ✅ PASSING | ~5s | WSS with TLS, token validation |
| 04-transport-udp.sh | UDP/QUIC | ✅ PASSING | ~5s | Multi-line payload testing |
| 05-encryption-ssl.sh | TLS encryption | ✅ PASSING | ~15s | 6 test cases (match/mismatch) |
| 06-authentication-key.sh | Mutual TLS auth | ✅ PASSING | ~10s | 3 test cases (success/failure) |
| 07-exec-simple.sh | Command execution | ✅ PASSING | ~5s | No PTY, token validation |
| 08-exec-pty.py | PTY mode | ✅ PASSING | ~10s | pexpect, Ctrl+C, line editing |
| 09-portfwd-local-tcp.sh | Local TCP forwarding | ✅ PASSING | ~10s | HTTP through tunnel, teardown |
| 14-socks-tcp-connect.sh | SOCKS5 proxy | ⚠️ NEEDS TEST | ~10s | HTTP through SOCKS, teardown |
| 16-behavior-connect-close.sh | Connection lifecycle | ⚠️ NEEDS TEST | ~10s | Listen persist, connect exit |
| 17-behavior-timeout.sh | Timeout handling | ⚠️ NEEDS TEST | ~10s | Normal + timeout detection |
| 18-behavior-stability.sh | Connection stability | ⚠️ NEEDS TEST | ~10s | 100ms timeout stability |
| 19-behavior-graceful-shutdown.sh | Graceful shutdown | ⚠️ NEEDS TEST | ~5s | SIGINT propagation |
| 20-feature-logging.sh | Session logging | ⚠️ NEEDS TEST | ~10s | Log file creation, data capture |

---

*This validation suite was designed specifically for GitHub Copilot Coding Agent to enable fast, reliable functionality verification.*
