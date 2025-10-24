# Troubleshooting & Verification Guide

> **Developer-facing document** for building, verifying, and debugging **goncat**. This is your internal playbook for confirming features work correctly and diagnosing issues quickly.

## Purpose & Scope

This guide is for **developers and agents maintaining the goncat CLI tool**, not end users. Use it to:

- **Build and verify** that features behave correctly from a clean environment
- **Diagnose issues** quickly with proven debugging strategies
- **Keep a record** of past problems and their resolutions
- **Perform manual verification** tests to confirm functionality

This is a living document that should be updated as new issues are discovered and resolved.

## Prerequisites

### Expected Environment

- **Operating System**: Clean Ubuntu VM (or similar Linux distribution)
- **Go Version**: 1.23 or higher (check with `go version`)
- **Build Tools**: `make`, `git`, `openssl` (for KEY_SALT generation)
- **Network Tools**: `netstat` or `ss`, `curl`, `python3` (for testing)
- **Dependencies**: Automatically fetched by `go mod tidy` (no manual setup needed)

### Environment Preparation

```bash
# Verify Go is installed
go version

# Ensure you're in the repository root
cd /home/runner/work/goncat/goncat  # Or your repo path
pwd

# Verify all dependencies are available
go mod tidy
```

## Building the Tool

### Clean Build Process

Always perform a clean build when starting fresh or after significant changes:

```bash
# Remove previous builds
rm -rf dist/

# Build Linux binary (fastest, ~11 seconds)
make build-linux

# Verify the binary was created
ls -lh dist/goncat.elf

# Expected: ~9-10MB binary
```

### Build All Platforms

To build for all platforms (Linux, Windows, macOS):

```bash
# Clean previous builds
rm -rf dist/

# Build all platforms (~30-40 seconds)
make build

# Verify all binaries
ls -lh dist/
# Expected output:
# goncat.elf   (~9-10MB) - Linux
# goncat.exe   (~9-10MB) - Windows  
# goncat.macho (~9-10MB) - macOS
```

### Build Notes

- Each build generates a **random KEY_SALT** using `openssl rand -hex 64`
- Builds use `CGO_ENABLED=0` for static compilation (no external dependencies)
- The KEY_SALT and VERSION are embedded via ldflags
- Binary sizes are approximately 9-10MB after symbol stripping

### Quick Build Verification

```bash
# Check version (should output: 0.0.1)
./dist/goncat.elf version

# Check help works
./dist/goncat.elf --help

# Verify exit code
./dist/goncat.elf version && echo "Build OK" || echo "Build FAILED"
```

## Manual Verification Scenarios

### Feature: Version Check

**Goal:** Verify the binary reports the correct version.

**Setup:** Build the binary (see Building the Tool).

**Steps:**
```bash
cd /home/runner/work/goncat/goncat
./dist/goncat.elf version
```

**Expected Behavior:**
- Output: `0.0.1`
- Exit code: 0

**Validation:**
```bash
./dist/goncat.elf version | grep -q "0.0.1" && echo "PASS" || echo "FAIL"
```

**Cleanup:** None needed.

---

### Feature: Help Commands

**Goal:** Verify all help commands display correct information.

**Setup:** Build the binary.

**Steps:**
```bash
cd /home/runner/work/goncat/goncat

# Main help
./dist/goncat.elf --help

# Master help
./dist/goncat.elf master --help

# Slave help
./dist/goncat.elf slave --help

# Master listen help
./dist/goncat.elf master listen --help

# Master connect help
./dist/goncat.elf master connect --help
```

**Expected Behavior:**
- Each command displays usage information
- No error messages
- Exit code: 0

**Validation:**
```bash
./dist/goncat.elf --help | grep -q "goncat" && echo "PASS" || echo "FAIL"
```

**Cleanup:** None needed.

---

### Feature: Basic Reverse Shell (Master Listen, Slave Connect)

**Goal:** Verify the most common use case - reverse shell with master listening and slave connecting.

**Setup:**
```bash
cd /home/runner/work/goncat/goncat
make build-linux
```

**Steps:**
```bash
# Terminal 1 (or background process): Start master listener
./dist/goncat.elf master listen 'tcp://*:12345' --exec /bin/sh &
MASTER_PID=$!

# Wait for master to start
sleep 2

# Verify master is listening
ss -tlnp 2>/dev/null | grep 12345

# Terminal 2: Connect slave and send test command
echo "echo 'REVERSE_SHELL_TEST' && exit" | ./dist/goncat.elf slave connect tcp://localhost:12345

# Check master output (should show connection established)
```

**Expected Behavior:**
- Master displays: `[+] Listening on :12345`
- Slave displays: `[+] Connecting to localhost:12345`
- Both display: `[+] Session with <address> established`
- Shell command executes and returns output
- Connection closes cleanly

**Validation:**
```bash
# Verify port is listening
ss -tlnp 2>/dev/null | grep 12345 | grep -q goncat && echo "Master listening: PASS" || echo "FAIL"

# Test connection succeeds
echo "exit" | timeout 5 ./dist/goncat.elf slave connect tcp://localhost:12345 && echo "Connection: PASS" || echo "FAIL"
```

**Cleanup:**
```bash
kill $MASTER_PID 2>/dev/null || true
pkill -9 goncat.elf 2>/dev/null || true
```

---

### Feature: Basic Bind Shell (Slave Listen, Master Connect)

**Goal:** Verify bind shell mode where slave listens and master connects.

**Setup:**
```bash
cd /home/runner/work/goncat/goncat
make build-linux
```

**Steps:**
```bash
# Terminal 1: Start slave listener
./dist/goncat.elf slave listen 'tcp://*:12346' &
SLAVE_PID=$!

# Wait for slave to start
sleep 2

# Verify slave is listening
ss -tlnp 2>/dev/null | grep 12346

# Terminal 2: Connect master with shell command
echo "echo 'BIND_SHELL_TEST' && exit" | ./dist/goncat.elf master connect tcp://localhost:12346 --exec /bin/sh
```

**Expected Behavior:**
- Slave displays: `[+] Listening on :12346`
- Master displays: `[+] Connecting to localhost:12346`
- Both display: `[+] Session with <address> established`
- Shell command executes
- Connection closes after command completes

**Validation:**
```bash
# Verify slave is listening
ss -tlnp 2>/dev/null | grep 12346 | grep -q goncat && echo "Slave listening: PASS" || echo "FAIL"
```

**Cleanup:**
```bash
kill $SLAVE_PID 2>/dev/null || true
pkill -9 goncat.elf 2>/dev/null || true
```

---

### Feature: TLS Encryption

**Goal:** Verify encrypted connections using TLS.

**Setup:**
```bash
cd /home/runner/work/goncat/goncat
make build-linux
```

**Steps:**
```bash
# Start master with TLS
./dist/goncat.elf master listen 'tcp://*:12347' --ssl --exec /bin/sh &
MASTER_PID=$!

sleep 2

# Connect slave with TLS
echo "echo 'TLS_TEST' && exit" | ./dist/goncat.elf slave connect tcp://localhost:12347 --ssl
```

**Expected Behavior:**
- Connection establishes successfully
- No TLS handshake errors
- Command executes and returns output
- Both sides show "Session established" message

**Validation:**
```bash
# Test TLS connection succeeds
echo "exit" | timeout 10 ./dist/goncat.elf slave connect tcp://localhost:12347 --ssl
EXIT_CODE=$?
[ $EXIT_CODE -eq 0 ] && echo "TLS connection: PASS" || echo "FAIL (exit code: $EXIT_CODE)"
```

**Cleanup:**
```bash
kill $MASTER_PID 2>/dev/null || true
pkill -9 goncat.elf 2>/dev/null || true
```

---

### Feature: Mutual Authentication

**Goal:** Verify password-based mutual authentication works correctly.

**Setup:**
```bash
cd /home/runner/work/goncat/goncat
make build-linux
```

**Steps:**
```bash
# Start master with TLS and password
./dist/goncat.elf master listen 'tcp://*:12348' --ssl --key testpassword123 --exec /bin/sh &
MASTER_PID=$!

sleep 2

# Test 1: Connect with CORRECT password (should succeed)
echo "echo 'AUTH_SUCCESS' && exit" | ./dist/goncat.elf slave connect tcp://localhost:12348 --ssl --key testpassword123
AUTH_SUCCESS=$?

# Test 2: Connect with WRONG password (should fail)
echo "echo 'SHOULD_NOT_SEE_THIS'" | ./dist/goncat.elf slave connect tcp://localhost:12348 --ssl --key wrongpassword 2>&1 | grep -q "certificate signed by unknown authority"
AUTH_FAIL=$?
```

**Expected Behavior:**
- **Correct password**: Connection succeeds, command executes
- **Wrong password**: Connection fails with TLS certificate verification error
- Error message: "x509: certificate signed by unknown authority"

**Validation:**
```bash
# Verify correct password works
[ $AUTH_SUCCESS -eq 0 ] && echo "Correct password: PASS" || echo "FAIL"

# Verify wrong password fails
[ $AUTH_FAIL -eq 0 ] && echo "Wrong password rejected: PASS" || echo "FAIL"
```

**Cleanup:**
```bash
kill $MASTER_PID 2>/dev/null || true
pkill -9 goncat.elf 2>/dev/null || true
```

---

### Feature: Session Logging

**Goal:** Verify session data can be logged to a file.

**Setup:**
```bash
cd /home/runner/work/goncat/goncat
make build-linux
rm -f /tmp/test-session.log
```

**Steps:**
```bash
# Start master with logging
./dist/goncat.elf master listen 'tcp://*:12349' --exec /bin/sh --log /tmp/test-session.log &
MASTER_PID=$!

sleep 2

# Connect slave and execute commands
echo "echo 'LOGGED_OUTPUT' && exit" | ./dist/goncat.elf slave connect tcp://localhost:12349

# Wait for session to complete
sleep 2

# Check log file was created
ls -lh /tmp/test-session.log
```

**Expected Behavior:**
- Log file is created at specified path
- File contains session data (may be empty or contain output depending on timing)
- File has appropriate permissions (readable by owner)

**Validation:**
```bash
# Verify log file exists
[ -f /tmp/test-session.log ] && echo "Log file created: PASS" || echo "FAIL"
```

**Cleanup:**
```bash
kill $MASTER_PID 2>/dev/null || true
pkill -9 goncat.elf 2>/dev/null || true
rm -f /tmp/test-session.log
```

---

### Feature: WebSocket Transport (ws)

**Goal:** Verify WebSocket protocol works for connections.

**Setup:**
```bash
cd /home/runner/work/goncat/goncat
make build-linux
```

**Steps:**
```bash
# Start master with WebSocket protocol
./dist/goncat.elf master listen 'ws://*:8080' --exec /bin/sh &
MASTER_PID=$!

sleep 2

# Verify WebSocket server is listening
ss -tlnp 2>/dev/null | grep 8080

# Connect slave via WebSocket
echo "echo 'WS_TEST' && exit" | timeout 5 ./dist/goncat.elf slave connect ws://localhost:8080
```

**Expected Behavior:**
- Master listens on port 8080
- Connection establishes with "websocket" protocol indicator
- Command executes successfully
- Connection closes cleanly

**Validation:**
```bash
# Verify WebSocket port is listening
ss -tlnp 2>/dev/null | grep 8080 | grep -q goncat && echo "WebSocket listening: PASS" || echo "FAIL"
```

**Cleanup:**
```bash
kill $MASTER_PID 2>/dev/null || true
pkill -9 goncat.elf 2>/dev/null || true
```

---

### Feature: WebSocket Secure Transport (wss)

**Goal:** Verify WebSocket Secure (TLS) protocol works.

**Setup:**
```bash
cd /home/runner/work/goncat/goncat
make build-linux
```

**Steps:**
```bash
# Start master with WebSocket Secure protocol
./dist/goncat.elf master listen 'wss://*:8443' --exec /bin/sh &
MASTER_PID=$!

sleep 2

# Verify WSS server is listening
ss -tlnp 2>/dev/null | grep 8443

# Connect slave via WebSocket Secure
echo "echo 'WSS_TEST' && exit" | timeout 5 ./dist/goncat.elf slave connect wss://localhost:8443
```

**Expected Behavior:**
- Master listens on port 8443
- TLS handshake completes successfully
- Connection establishes with "websocket" protocol indicator
- Command executes successfully

**Validation:**
```bash
# Verify WSS port is listening
ss -tlnp 2>/dev/null | grep 8443 | grep -q goncat && echo "WSS listening: PASS" || echo "FAIL"
```

**Cleanup:**
```bash
kill $MASTER_PID 2>/dev/null || true
pkill -9 goncat.elf 2>/dev/null || true
```

---

### Feature: Local Port Forwarding

**Goal:** Verify local port forwarding (-L) works correctly.

**Setup:**
```bash
cd /home/runner/work/goncat/goncat
make build-linux

# Create a test HTTP server on port 9999
python3 -m http.server 9999 &
HTTP_PID=$!
sleep 2
```

**Steps:**
```bash
# Start master with local port forwarding
# Forward local port 8888 to localhost:9999 via slave
./dist/goncat.elf master listen 'tcp://*:12350' --exec /bin/sh -L 8888:localhost:9999 &
MASTER_PID=$!

sleep 2

# Connect slave
./dist/goncat.elf slave connect tcp://localhost:12350 &
SLAVE_PID=$!

sleep 3

# Verify forwarded port is accessible
curl -s http://localhost:8888/ | head -5
```

**Expected Behavior:**
- Port 8888 opens on master side
- Connections to localhost:8888 are forwarded through slave to localhost:9999
- HTTP server responds through the tunnel
- Output shows HTML directory listing from Python HTTP server

**Validation:**
```bash
# Test forwarded port
curl -s http://localhost:8888/ | grep -q "Directory listing" && echo "Port forwarding: PASS" || echo "FAIL"
```

**Cleanup:**
```bash
kill $HTTP_PID 2>/dev/null || true
kill $MASTER_PID 2>/dev/null || true
kill $SLAVE_PID 2>/dev/null || true
pkill -9 goncat.elf 2>/dev/null || true
pkill -9 python3 2>/dev/null || true
```

---

### Feature: Remote Port Forwarding

**Goal:** Verify remote port forwarding (-R) works correctly.

**Setup:**
```bash
cd /home/runner/work/goncat/goncat
make build-linux

# Create a test HTTP server on port 9997
python3 -m http.server 9997 &
HTTP_PID=$!
sleep 2
```

**Steps:**
```bash
# Start master with remote port forwarding
# Forward slave's port 8889 to master's localhost:9997
./dist/goncat.elf master listen 'tcp://*:12360' --exec /bin/sh -R 8889:localhost:9997 &
MASTER_PID=$!

sleep 2

# Connect slave
./dist/goncat.elf slave connect tcp://localhost:12360 &
SLAVE_PID=$!

sleep 3

# Verify forwarded port is accessible on slave side (localhost)
# Since slave is on same machine, we can test it
curl -s http://localhost:8889/ | head -5
```

**Expected Behavior:**
- Port 8889 opens on slave side
- Connections to slave's localhost:8889 are forwarded back through master to localhost:9997
- HTTP server responds through the tunnel

**Validation:**
```bash
# Test remote forwarded port
curl -s http://localhost:8889/ | grep -q "Directory listing" && echo "Remote port forwarding: PASS" || echo "FAIL"
```

**Cleanup:**
```bash
kill $HTTP_PID 2>/dev/null || true
kill $MASTER_PID 2>/dev/null || true
kill $SLAVE_PID 2>/dev/null || true
pkill -9 goncat.elf 2>/dev/null || true
pkill -9 python3 2>/dev/null || true
```

---

### Feature: SOCKS Proxy

**Goal:** Verify SOCKS5 proxy (-D) functionality.

**Setup:**
```bash
cd /home/runner/work/goncat/goncat
make build-linux

# Create a test HTTP server
python3 -m http.server 9996 &
HTTP_PID=$!
sleep 2
```

**Steps:**
```bash
# Start master with SOCKS proxy on port 1080
./dist/goncat.elf master listen 'tcp://*:12351' --exec /bin/sh -D 1080 &
MASTER_PID=$!

sleep 2

# Connect slave
./dist/goncat.elf slave connect tcp://localhost:12351 &
SLAVE_PID=$!

sleep 3

# Verify SOCKS proxy port is listening
ss -tlnp 2>/dev/null | grep 1080

# Test SOCKS proxy with curl
curl -s --socks5 127.0.0.1:1080 http://localhost:9996/ | head -5
```

**Expected Behavior:**
- SOCKS proxy listens on 127.0.0.1:1080
- Requests through proxy are tunneled via slave
- HTTP requests succeed through the SOCKS tunnel

**Validation:**
```bash
# Verify SOCKS port is listening
ss -tlnp 2>/dev/null | grep 1080 | grep -q goncat && echo "SOCKS proxy listening: PASS" || echo "FAIL"
```

**Cleanup:**
```bash
kill $HTTP_PID 2>/dev/null || true
kill $MASTER_PID 2>/dev/null || true
kill $SLAVE_PID 2>/dev/null || true
pkill -9 goncat.elf 2>/dev/null || true
pkill -9 python3 2>/dev/null || true
```

---

### Feature: PTY Interactive Shell

**Goal:** Verify pseudo-terminal (PTY) support for interactive shells.

**Setup:**
```bash
cd /home/runner/work/goncat/goncat
make build-linux
```

**Steps:**
```bash
# Start master with PTY enabled
./dist/goncat.elf master listen 'tcp://*:12355' --exec /bin/bash --pty &
MASTER_PID=$!

sleep 2

# Connect slave (manual testing recommended for full PTY features)
# For automated testing, just verify connection works
echo "exit" | ./dist/goncat.elf slave connect tcp://localhost:12355
```

**Expected Behavior:**
- Master enables PTY mode with bash
- Interactive features should work (arrow keys, tab completion, etc.)
- Terminal size synchronization occurs automatically

**Validation:**
```bash
# Basic connection test
echo "exit" | timeout 5 ./dist/goncat.elf slave connect tcp://localhost:12355
[ $? -eq 0 ] && echo "PTY mode connection: PASS" || echo "FAIL"
```

**Manual Testing:**
For full PTY verification, manually test in two terminals:
1. Terminal 1: Run `./dist/goncat.elf master listen 'tcp://*:12355' --exec /bin/bash --pty`
2. Terminal 2: Run `./dist/goncat.elf slave connect tcp://localhost:12355`
3. Test: Arrow keys, tab completion, running `vim` or `top`

**Cleanup:**
```bash
kill $MASTER_PID 2>/dev/null || true
pkill -9 goncat.elf 2>/dev/null || true
```

---

### Feature: Self-Deleting Slave (Cleanup)

**Goal:** Verify --cleanup flag causes slave binary to delete itself.

**Setup:**
```bash
cd /home/runner/work/goncat/goncat
make build-linux

# Create a test copy of the binary
cp dist/goncat.elf /tmp/goncat-cleanup-test.elf
ls -lh /tmp/goncat-cleanup-test.elf
```

**Steps:**
```bash
# Start master
./dist/goncat.elf master listen 'tcp://*:12356' --exec /bin/sh &
MASTER_PID=$!

sleep 2

# Run slave with cleanup flag from /tmp
echo "exit" | /tmp/goncat-cleanup-test.elf slave connect tcp://localhost:12356 --cleanup

# Wait a moment for cleanup to complete
sleep 3

# Check if binary still exists
ls /tmp/goncat-cleanup-test.elf 2>&1
```

**Expected Behavior:**
- **Linux**: Binary deletes itself immediately after execution
- **Windows**: CMD script waits 5 seconds then deletes binary
- File should not exist after cleanup completes

**Validation:**
```bash
# Verify binary is deleted
if [ ! -f /tmp/goncat-cleanup-test.elf ]; then
    echo "Cleanup: PASS (file deleted)"
else
    echo "Cleanup: FAIL (file still exists)"
    rm -f /tmp/goncat-cleanup-test.elf
fi
```

**Cleanup:**
```bash
kill $MASTER_PID 2>/dev/null || true
pkill -9 goncat.elf 2>/dev/null || true
rm -f /tmp/goncat-cleanup-test.elf
```

---

### Feature: Timeout Configuration

**Goal:** Verify custom timeout values work correctly.

**Setup:**
```bash
cd /home/runner/work/goncat/goncat
make build-linux
```

**Steps:**
```bash
# Start master with 30-second timeout
./dist/goncat.elf master listen 'tcp://*:12357' --timeout 30000 --ssl --exec /bin/sh &
MASTER_PID=$!

sleep 2

# Connect slave with matching timeout
echo "echo 'TIMEOUT_TEST' && exit" | ./dist/goncat.elf slave connect tcp://localhost:12357 --timeout 30000 --ssl
```

**Expected Behavior:**
- Connection establishes without timeout errors
- TLS handshake completes within timeout period
- Command executes successfully

**Validation:**
```bash
# Test with custom timeout
echo "exit" | timeout 35 ./dist/goncat.elf slave connect tcp://localhost:12357 --timeout 30000 --ssl
[ $? -eq 0 ] && echo "Custom timeout: PASS" || echo "FAIL"
```

**Cleanup:**
```bash
kill $MASTER_PID 2>/dev/null || true
pkill -9 goncat.elf 2>/dev/null || true
```

---

### Feature: Verbose Logging

**Goal:** Verify --verbose flag enables detailed error logging.

**Setup:**
```bash
cd /home/runner/work/goncat/goncat
make build-linux
```

**Steps:**
```bash
# Test verbose mode with intentional error (connecting to non-existent host)
./dist/goncat.elf slave connect tcp://192.0.2.1:12345 --verbose 2>&1 | head -10
```

**Expected Behavior:**
- Detailed error messages printed to stderr
- More information than default mode
- Stack traces or detailed diagnostics visible

**Validation:**
```bash
# Verify verbose mode produces output
OUTPUT=$(./dist/goncat.elf slave connect tcp://192.0.2.1:12345 --verbose 2>&1)
echo "$OUTPUT" | grep -q "Error" && echo "Verbose logging: PASS" || echo "FAIL"
```

**Cleanup:** None needed.

---

## Common Problems and Fixes

This section documents recurring issues and their resolutions. Add new problems as they are discovered.

### Adding Temporary Debug Print Statements

**When to use:** Behavior is unclear or something fails unexpectedly.

**Process:**

1. **Form a theory** about the possible root cause of the issue.

2. **Add temporary debug print statements** in relevant functions instead of editing logic immediately:
   ```go
   fmt.Println("DEBUG_PRINT: reached point X, variable Y=", Y)
   fmt.Printf("DEBUG_PRINT: function foo() called with args: %+v\n", args)
   ```

3. **Rebuild the tool** with your debug statements:
   ```bash
   make build-linux
   ```

4. **Perform the relevant manual verification test** that reproduces the issue.

5. **Observe console output** and adjust your theory based on what you see.

6. **Make the minimal fix** once you've confirmed the root cause.

7. **Run the same manual test again** to verify the behavior is corrected.

8. **Remove all debug prints** using this cleanup command:
   ```bash
   # Linux:
   find . -name '*.go' -type f -exec sed -i '/DEBUG_PRINT:/d' {} +
   
   # macOS (requires .bak extension):
   find . -name '*.go' -type f -exec sed -i.bak '/DEBUG_PRINT:/d' {} + && find . -name '*.go.bak' -delete
   ```

9. **Verify cleanup** worked:
   ```bash
   git diff  # Should show only your actual fixes, no debug prints
   ```

10. **Commit only clean code** without any debug print statements.

---

### Port Already in Use

**Symptom:** Error binding to port: "address already in use"

**Cause:** Previous goncat process still running or another service using the port

**Resolution:**
```bash
# Find process using the port (example: port 12345)
lsof -i :12345  # Linux/macOS
# OR
ss -tlnp | grep 12345  # Linux

# Kill the process
kill -9 <PID>

# Verify port is free
ss -tlnp | grep 12345  # Should return nothing

# Alternative: Choose a different port
./dist/goncat.elf master listen 'tcp://*:12346' --exec /bin/sh
```

---

### Connection Timeout

**Symptom:** "connection timeout" or "i/o timeout" errors

**Cause:** 
- Network connectivity issues
- Firewall blocking the connection
- Master not listening when slave tries to connect
- Default timeout too short for slow networks

**Resolution:**
```bash
# 1. Increase timeout value (default is 10 seconds)
./dist/goncat.elf master listen 'tcp://*:12345' --timeout 30000 --exec /bin/sh
./dist/goncat.elf slave connect tcp://host:12345 --timeout 30000

# 2. Verify master is listening before connecting slave
ss -tlnp | grep 12345

# 3. Check firewall rules
sudo ufw status  # Ubuntu
sudo iptables -L  # Other Linux

# 4. Test basic connectivity
ping <host>
telnet <host> <port>
```

---

### TLS Handshake Failure

**Symptom:** "tls handshake" error or "x509: certificate signed by unknown authority"

**Cause:**
- Password mismatch when using `--key` flag
- Only one side using `--ssl` flag
- Timeout during TLS negotiation

**Resolution:**
```bash
# 1. Ensure BOTH sides use --ssl flag
./dist/goncat.elf master listen 'tcp://*:12345' --ssl --exec /bin/sh
./dist/goncat.elf slave connect tcp://host:12345 --ssl

# 2. If using --key, ensure EXACT same password on both sides
./dist/goncat.elf master listen 'tcp://*:12345' --ssl --key "mypassword" --exec /bin/sh
./dist/goncat.elf slave connect tcp://host:12345 --ssl --key "mypassword"

# 3. Increase timeout for slow connections
./dist/goncat.elf master listen 'tcp://*:12345' --ssl --timeout 30000 --exec /bin/sh
./dist/goncat.elf slave connect tcp://host:12345 --ssl --timeout 30000
```

---

### Binary Not Found or Permission Denied

**Symptom:** "command not found" or "permission denied" when running goncat

**Cause:**
- Binary not in PATH
- Binary not executable
- Binary in wrong location

**Resolution:**
```bash
# 1. Use full path to binary
/home/runner/work/goncat/goncat/dist/goncat.elf version

# 2. Make binary executable
chmod +x /home/runner/work/goncat/goncat/dist/goncat.elf

# 3. Copy to PATH location
sudo cp dist/goncat.elf /usr/local/bin/goncat

# 4. OR add dist directory to PATH
export PATH=$PATH:/home/runner/work/goncat/goncat/dist
```

---

### PTY Not Working

**Symptom:** Shell doesn't respond to arrow keys or tab completion

**Cause:**
- `--pty` flag not specified on master side
- Using shell that doesn't support PTY
- PTY not supported on platform (Windows < 10 1809)

**Resolution:**
```bash
# 1. Ensure --pty is on MASTER side only
./dist/goncat.elf master listen 'tcp://*:12345' --exec /bin/bash --pty
./dist/goncat.elf slave connect tcp://localhost:12345

# 2. Use appropriate shell for platform
# Linux/macOS: /bin/bash, /bin/zsh, /bin/sh
# Windows: cmd.exe, powershell.exe

# 3. Test without PTY first to isolate issue
./dist/goncat.elf master listen 'tcp://*:12345' --exec /bin/sh
# Then add --pty if basic shell works
```

---

### Build Failures

**Symptom:** `make build-linux` fails with compilation errors

**Cause:**
- Missing dependencies
- Wrong Go version
- Corrupted go.mod or go.sum

**Resolution:**
```bash
# 1. Verify Go version (need 1.23+)
go version

# 2. Clean and refresh dependencies
go clean -cache -modcache
go mod tidy
go mod download

# 3. Try building again
make build-linux

# 4. Check for syntax errors if still failing
go vet ./...
go fmt ./...
```

---

### Multiple Processes After Testing

**Symptom:** Many goncat processes still running after tests

**Cause:** Background processes not properly cleaned up

**Resolution:**
```bash
# Kill all goncat processes
pkill -9 goncat.elf
# OR
killall -9 goncat.elf

# Verify all processes are gone
ps aux | grep goncat

# Check for orphaned port bindings
ss -tlnp | grep goncat
```

---

## Appendix: Quick Verification Checklist

Use this checklist to quickly verify core functionality works after changes:

```bash
# ============================================
# Quick Verification Checklist
# ============================================

# 1. Clean build
rm -rf dist/
make build-linux
[ $? -eq 0 ] && echo "✓ Build successful" || echo "✗ Build failed"

# 2. Version check
./dist/goncat.elf version | grep -q "0.0.1"
[ $? -eq 0 ] && echo "✓ Version check passed" || echo "✗ Version check failed"

# 3. Help works
./dist/goncat.elf --help | grep -q "goncat"
[ $? -eq 0 ] && echo "✓ Help works" || echo "✗ Help failed"

# 4. Basic reverse shell (5 second test)
timeout 10 bash -c '
  ./dist/goncat.elf master listen "tcp://*:12399" --exec /bin/sh &
  MASTER_PID=$!
  sleep 2
  echo "exit" | ./dist/goncat.elf slave connect tcp://localhost:12399 &
  sleep 3
  kill $MASTER_PID 2>/dev/null
'
[ $? -eq 0 ] && echo "✓ Basic connection works" || echo "✗ Basic connection failed"

# 5. TLS encryption (5 second test)
timeout 10 bash -c '
  ./dist/goncat.elf master listen "tcp://*:12398" --ssl --exec /bin/sh &
  MASTER_PID=$!
  sleep 2
  echo "exit" | ./dist/goncat.elf slave connect tcp://localhost:12398 --ssl &
  sleep 3
  kill $MASTER_PID 2>/dev/null
'
[ $? -eq 0 ] && echo "✓ TLS encryption works" || echo "✗ TLS failed"

# 6. Cleanup
pkill -9 goncat.elf 2>/dev/null
echo ""
echo "✓ Quick verification complete"
```

**Time required:** ~30 seconds

**Expected result:** All checks pass with ✓ marks

---

## Testing Integration

### Running Existing Test Suites

Before making changes or when verifying fixes:

```bash
# Unit tests (fast, ~5 seconds)
make test-unit

# Integration tests (~1-2 seconds)
make test-integration

# E2E tests (slow, ~8-9 minutes, requires Docker)
make test-e2e

# All tests
make test
```

### When to Run What Tests

- **Unit tests**: After any code change, before committing
- **Integration tests**: After completing a feature or fix
- **E2E tests**: Before finalizing a PR or release
- **Manual verification**: When automated tests don't cover the scenario

---

## Best Practices

### Development Workflow

1. **Always start with a clean build**
   ```bash
   rm -rf dist/ && make build-linux
   ```

2. **Run unit tests before manual testing**
   ```bash
   make test-unit
   ```

3. **Use manual verification scenarios** from this guide to confirm features work

4. **Add debug prints temporarily** when debugging (remove before committing)

5. **Test one feature at a time** to isolate issues

6. **Document new issues** in the "Common Problems" section as you discover them

7. **Update this guide** when manual verification processes change

### Debugging Tips

- Use `--verbose` flag to see detailed error messages
- Check ports are listening with `ss -tlnp | grep <port>`
- Verify no orphaned processes with `ps aux | grep goncat`
- Test basic connectivity before complex features
- Isolate issues by testing without encryption first, then add `--ssl`
- Check logs when using `--log` flag to understand session behavior

---

## Appendix: Environment Information

### Checking Your Environment

```bash
# Operating system
uname -a

# Go version (need 1.23+)
go version

# Available tools
which make git openssl curl python3 netstat ss

# Network interfaces
ip addr show  # Linux
ifconfig      # macOS

# Free ports
ss -tlnp  # See what ports are in use
```

### Repository Information

```bash
# Current branch
git branch --show-current

# Current commit
git rev-parse HEAD
git rev-parse --short HEAD

# Uncommitted changes
git status
git diff

# Repository size
du -sh .
```

---

## Freshness Footer

**Generated from commit `fe27e5b` on `2025-10-24`.**

Last verified: 2025-10-24

This document should be updated whenever:
- New features are added
- New issues are discovered and resolved
- Build process changes
- Manual verification scenarios change
