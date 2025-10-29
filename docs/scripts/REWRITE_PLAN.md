# Validation Scripts Rewrite Plan

## Critical Data Flow Understanding

**CORRECT:** Master with `--exec /bin/sh`:
1. Master starts with `--exec /bin/sh` flag
2. When slave connects, **slave** starts the shell process
3. Commands entered on **master stdin** → sent to slave → executed by slave's shell → output sent back → appears on **master stdout**

**Testing Pattern:**
```bash
# Start master with command input piped
(echo "echo TOKEN"; echo "exit") | goncat master listen tcp://*:PORT --exec /bin/sh > master.log 2>&1 &
MASTER_PID=$!

# Connect slave
goncat slave connect tcp://localhost:PORT > slave.log 2>&1 &
SLAVE_PID=$!

# Verify TOKEN appears in master.log (not slave.log!)
```

## Common Patterns to Fix

### 1. Remove `|| true` that hides failures
**Bad:** `command || true`
**Good:** `command` (let failures propagate)

### 2. Use PID-based cleanup
**Bad:** `pkill -9 goncat.elf`
**Good:** 
```bash
MASTER_PID=""
SLAVE_PID=""
cleanup() {
    [ -n "$MASTER_PID" ] && kill "$MASTER_PID" 2>/dev/null && wait "$MASTER_PID" 2>/dev/null
    [ -n "$SLAVE_PID" ] && kill "$SLAVE_PID" 2>/dev/null && wait "$SLAVE_PID" 2>/dev/null
}
trap cleanup EXIT
```

### 3. Poll instead of fixed sleeps
**Bad:** `sleep 5`
**Good:**
```bash
poll_for_pattern() {
    local file="$1"
    local pattern="$2"
    local timeout="${3:-10}"
    local start=$(date +%s)
    while true; do
        if [ -f "$file" ] && grep -qE "$pattern" "$file" 2>/dev/null; then
            return 0
        fi
        local now=$(date +%s)
        if [ $((now - start)) -ge "$timeout" ]; then
            return 1
        fi
        sleep 0.1
    done
}
```

### 4. Use unique deterministic tokens
**Bad:** Checking for generic output
**Good:**
```bash
TOKEN="UNIQUE_$$_$RANDOM_$(date +%s)"
# Send via master stdin, verify in master stdout
```

## Script-Specific Requirements

### Transport Scripts (01-04)
- Send unique token via master stdin
- Verify token in master stdout
- Test both session established and closed messages
- Verify listen mode persistence

### Security Scripts (05-06)
- **05-encryption-ssl.sh** - Test matrix:
  1. --ssl both sides → success + token verification
  2. Mismatched --ssl → failure
  3. --ssl --key matching → success
  4. --ssl wrong key → failure
  5. --key without --ssl → CLI error

- **06-authentication-key.sh** - Similar matrix plus handshake marker grep

### Execution Scripts (07-08)
- **07-exec-simple.sh** - Token via master stdin, second connection test
- **08-exec-pty.py** - Drive MASTER with pexpect, test TTY features:
  - Deterministic PS1 prompt
  - test -t 0, stty -a, tput cols
  - Ctrl-C interrupt of sleep
  - Line editing (arrow keys)

### Port Forwarding (09)
- Serve unique token from HTTP server
- Decisive curl assertion (not "inconclusive")
- Second curl for persistence
- Curl after slave kill for teardown

### SOCKS (14)
- Unique token from HTTP server
- Decisive curl --socks5 + grep
- No timeout on slave
- Negative test: curl fails after slave killed

### Behavior Scripts (16-19)
- **16-connect-close** - Test connect mode exits, listen persists
- **17-timeout** - Force timeout scenarios (1-10ms), verify errors
- **18-stability** - Two spaced commands in one session
- **19-graceful-shutdown** - Split EOF and Ctrl+C tests

### Features (20)
- **20-logging** - Token in session log file, append/overwrite test, negative case

## Implementation Order

1. ✅ Fix data flow understanding (done in 93de4a8)
2. Fix transport scripts (01-04) - use as templates
3. Fix security scripts (05-06) - test matrices
4. Fix execution scripts (07-08) - PTY requires pexpect
5. Fix port forwarding (09) - tunnel validation
6. Fix SOCKS (14) - proxy validation
7. Fix behavior scripts (16-19) - UX validation
8. Fix features (20) - logging validation

## Testing Each Script

Always test with timeout:
```bash
timeout 30 bash docs/scripts/01-transport-tcp.sh
```

Check for:
- Exit code 0 on success
- No hanging processes
- Clear pass/fail output
- Proper cleanup on exit
