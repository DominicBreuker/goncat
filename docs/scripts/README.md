# Validation Scripts Directory

This directory contains standalone bash validation scripts for verifying goncat functionality.

## Purpose

These scripts are designed for GitHub Copilot Coding Agent to quickly validate that goncat features work correctly. Each script:

- Runs standalone without Docker or complex infrastructure
- Completes in seconds (except stability tests)
- Verifies real functionality with actual data transfer
- Provides clear pass/fail output
- Cleans up automatically

## Organization

### Transport Scripts (01-04)
- `01-transport-tcp.sh` - TCP transport validation
- `02-transport-ws.sh` - WebSocket transport
- `03-transport-wss.sh` - WebSocket Secure (TLS)
- `04-transport-udp.sh` - UDP/QUIC transport

### Security Scripts (05-06)
- `05-encryption-ssl.sh` - TLS encryption with --ssl flag
- `06-authentication-key.sh` - Mutual authentication with --key

### Execution Scripts (07-08)
- `07-exec-simple.sh` - Simple command execution
- `08-exec-pty.sh` - PTY mode for interactive shells

### Port Forwarding Scripts (09-13)
- `09-portfwd-local-tcp.sh` - Local TCP port forwarding
- Additional port forwarding scripts to be added

### Connection Behavior Scripts (16-19)
- `16-behavior-connect-close.sh` - Connection close behavior
- Additional behavior scripts to be added

### Feature Scripts (20-21)
- `20-feature-logging.sh` - Session logging
- Additional feature scripts to be added

### Helper Scripts
- `helpers/cleanup.sh` - Clean up processes and temp files
- Additional helpers to be added

## Usage

### Running Individual Scripts

```bash
# From repository root
bash docs/scripts/01-transport-tcp.sh
```

### Running All Scripts

```bash
for script in docs/scripts/[0-9]*.sh; do
    bash "$script" || echo "FAILED: $script"
done
```

### Transport-Parameterized Scripts

Some scripts accept transport as argument:

```bash
bash docs/scripts/01-transport-tcp.sh tcp
bash docs/scripts/01-transport-tcp.sh ws
```

## Requirements

- bash 4.0+
- python3 (for test servers)
- curl (for HTTP testing)
- Built goncat binary (scripts will build if missing)

## Script Template

All scripts follow this structure:

```bash
#!/bin/bash
set -euo pipefail

# Setup paths and colors
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

# Cleanup trap
cleanup() {
    pkill -9 goncat.elf 2>/dev/null || true
    rm -f /tmp/goncat-test-*
}
trap cleanup EXIT

# Build if needed
if [ ! -f "$REPO_ROOT/dist/goncat.elf" ]; then
    make build-linux
fi

# Test implementation
echo "Starting validation: [Name]"
# ... test steps ...
echo "✓ [Name] validation passed"
exit 0
```

## Adding New Scripts

1. Follow the template structure
2. Include proper cleanup via trap
3. Test manually before committing
4. Update VALIDATION.md with new scenario
5. Ensure script completes in < 10 seconds (except stability tests)
6. Verify real functionality, not just flag acceptance

## See Also

- `docs/VALIDATION.md` - Complete validation documentation
- `docs/plans/validation-scripts.plan.md` - Implementation plan
- `docs/TROUBLESHOOT.md` - Manual verification procedures

---

*These validation scripts are explicitly designed for GitHub Copilot Coding Agent.*
