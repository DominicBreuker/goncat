# Copilot Instructions for goncat

## Project Overview

**goncat** is a netcat-like tool written in Go for creating bind or reverse shells with an SSH-like experience. Key features include:
- Encryption with mutual authentication (TLS)
- Cross-platform PTY support (Linux, Windows, macOS)
- Port forwarding (local/remote) and SOCKS proxy support
- Session logging and automatic cleanup capabilities

**Repository Stats:**
- ~125 files, ~5,600 lines of Go code
- Small footprint: ~850KB repository size
- Go version: 1.23+ (toolchain 1.24.0)
- License: GPL v3

## Build & Test Instructions

### Prerequisites
- Go 1.23+ installed
- Docker and Docker Compose for integration tests
- `openssl` command available (for KEY_SALT generation)

### Building

**IMPORTANT:** Always build binaries in the following order to avoid issues:

1. **Clean build (recommended before any build):**
   ```bash
   rm -rf dist
   ```

2. **Build all platforms (Linux, Windows, macOS):**
   ```bash
   make build
   ```
   - Creates binaries in `dist/` directory
   - Generates: `goncat.elf` (Linux), `goncat.exe` (Windows), `goncat.macho` (macOS)
   - Build time: ~15-30 seconds for all platforms

3. **Build individual platforms:**
   ```bash
   make build-linux    # Creates dist/goncat.elf
   make build-windows  # Creates dist/goncat.exe
   make build-darwin   # Creates dist/goncat.macho
   ```

**Build Notes:**
- The build uses CGO_ENABLED=0 for static compilation
- Each build generates a random KEY_SALT using `openssl rand -hex 64`
- The KEY_SALT and VERSION are embedded via ldflags
- Binary sizes are ~9-10MB each after stripping symbols

### Testing

**Unit Tests:**
```bash
go test ./...
```
- Fast execution: ~2-3 seconds
- Only one test file exists: `cmd/shared/parsers_test.go`
- Tests transport protocol parsing (tcp://, ws://, wss://)

**Integration Tests:**
```bash
make test
```
- **REQUIRES:** Docker and Docker Compose installed
- **REQUIRES:** Linux binary built first (runs `make build-linux` automatically)
- Tests both bind shell and reverse shell scenarios
- Tests all transport protocols: tcp, ws, wss
- Uses Alpine Linux containers with expect for testing
- Total test time: ~2-3 minutes
- Tests are in `tests/` directory with expect scripts

**Individual Test Targets:**
```bash
make test-unit         # Only unit tests (fast)
make test-integration  # Only integration tests (requires Docker)
```

**Integration Test Details:**
- Uses `docker compose` with test configurations in `tests/` directory
- Two main test scenarios:
  - `slave-listen` (bind shell): master connects to listening slave
  - `slave-connect` (reverse shell): slave connects to listening master
- Tests verify: basic connectivity, exec mode, PTY mode
- All tests run with `--exit-code-from` to propagate test failures

### Known Build/Test Issues

1. **Integration tests require Docker** - unit tests will pass without Docker but `make test` will fail
2. **KEY_SALT changes on each build** - this is intentional for security
3. **No linter configuration** - no golangci-lint, gofmt, or govet configs exist. Use standard Go tools:
   ```bash
   go fmt ./...
   go vet ./...  # Note: pre-existing unreachable code warning in pkg/handler/socks/slave/associate.go
   ```
4. **Pre-existing go vet warning** - `pkg/handler/socks/slave/associate.go:174:2: unreachable code` exists in main branch, ignore it

## Project Structure

### Root Files
```
LICENSE          - GPL v3 license
Makefile         - Build and test automation
README.md        - Main documentation
go.mod           - Go module definition
go.sum           - Go dependency checksums
.gitignore       - Ignores dist/ and notes.md
```

### Source Directories

**cmd/** - Command-line interface and entry points
- `main.go` - Application entry point using urfave/cli/v3
- `master/` - Master mode commands (controls the connection)
- `slave/` - Slave mode commands (executes instructions)
- `shared/` - Common flags and parsers for both modes
- `version/` - Version command implementation
- `masterconnect/`, `masterlisten/` - Master connection handlers
- `slaveconnect/`, `slavelisten/` - Slave connection handlers

**pkg/** - Core functionality packages
- `transport/` - Protocol implementations (tcp, ws/websocket)
- `crypto/` - TLS certificate generation and mutual auth
- `pty/` - Cross-platform PTY support (separate implementations for Unix and Windows)
- `handler/` - Connection handlers for master/slave, port forwarding, SOCKS
- `mux/` - Multiplexing using hashicorp/yamux
- `socks/` - SOCKS proxy implementation
- `exec/` - Command execution logic
- `server/` - Server-side connection handling
- `client/` - Client-side connection handling
- `clean/` - Self-deletion cleanup logic
- `log/` - Session logging
- `config/` - Configuration structures
- `terminal/` - Terminal handling utilities
- `pipeio/` - I/O piping utilities
- `format/` - Output formatting with color support

**tests/** - Integration test infrastructure
- `Dockerfile` - Alpine Linux with expect for testing
- `docker-compose.slave-*.yml` - Test orchestration configs
- `test-runner.sh` - Test execution wrapper
- `lib.tcl` - Expect library for tests
- `master-connect/`, `master-listen/` - Test scenarios

### Key Source Files

- `cmd/main.go` - Entry point, CLI app with master/slave subcommands
- `pkg/config/config.go` - Config structs, protocol constants (ProtoTCP/WS/WSS)
- `cmd/shared/shared.go` - CLI flag definitions
- `pkg/transport/` - Transport interface and implementations (tcp, ws)
- `pkg/pty/pty_unix.go`, `pkg/pty/pty_windows.go` - Platform-specific PTY
- `pkg/exec/exec_windows.go` - Windows-specific execution

### Dependencies (go.mod)

Main dependencies:
- `github.com/urfave/cli/v3` - CLI framework
- `github.com/coder/websocket` - WebSocket support
- `github.com/hashicorp/yamux` - Stream multiplexing
- `github.com/fatih/color` - Colored output
- `github.com/muesli/cancelreader` - Cancelable stdin reader
- `golang.org/x/sys`, `golang.org/x/term` - System and terminal support

## CI/CD Pipeline

**GitHub Actions Workflow:** `.github/workflows/test.yml`

Triggers:
- Push to `main` branch
- All pull requests

Build Steps:
1. Checkout repository
2. Setup Go 1.23
3. Run `make build` (builds all platforms)
4. List built binaries
5. Run `make test` (unit + integration tests)

**Environment:** ubuntu-latest with Docker support

**Expected Runtime:** ~5-7 minutes total

## Validation Steps

Before committing changes:

1. **Format code:**
   ```bash
   go fmt ./...
   ```

2. **Run unit tests:**
   ```bash
   go test ./...
   ```

3. **Build all platforms:**
   ```bash
   make build
   ```

4. **Verify binaries:**
   ```bash
   ls -lh dist/
   ./dist/goncat.elf version  # Should output: 0.0.1
   ```

5. **Run full test suite (if Docker available):**
   ```bash
   make test
   ```

## Common Tasks

**CLI flag:** Add to `cmd/shared/shared.go`, update `pkg/config/`, add tests to `cmd/shared/parsers_test.go`
**Transport protocol:** Create `pkg/transport/newproto/`, implement Transport interface, add to `pkg/config/config.go`
**PTY changes:** Edit `pkg/pty/pty_unix.go` (Unix) or `pkg/pty/pty_windows.go` (Windows) - platform-specific

**Manual testing:**
```bash
make build-linux
./dist/goncat.elf slave listen 'tcp://*:12345'  # Terminal 1
./dist/goncat.elf master connect tcp://localhost:12345 --exec /bin/sh  # Terminal 2
```

## Important Notes

1. **Always run `go test ./...` before committing** - Fast, catches parser issues
2. **Build generates random KEY_SALT each time** - Intentional for security
3. **Integration tests need Docker** - Skip if only non-functional changes
4. **Run `go fmt ./...` before committing** - No CI formatting check
5. **Binaries:** .elf (Linux), .exe (Windows), .macho (macOS) in gitignored `dist/`
6. **Static builds:** CGO disabled, no vendoring, dependencies via `go mod`
7. **Windows PTY quirks:** See `pkg/pty/pty_windows.go` comments

## Architecture

**Flow:** CLI (urfave/cli) → cmd/main.go → cmd/master or cmd/slave → pkg/transport (tcp/ws) → pkg/mux (yamux) → pkg/handler → pkg/exec or pkg/pty

**Master vs Slave:** Master controls params (exec, pty, logs, forwarding); Slave executes instructions. Both can listen or connect (4 combos).

**TLS:** Ephemeral certs per run, optional password-based mutual auth (`pkg/crypto/ca.go`)

## Trust These Instructions

These instructions have been validated by:
- Running all build commands successfully
- Executing unit tests (passing)
- Verifying binary creation for all platforms
- Reviewing all documentation and configuration files
- Testing the GitHub Actions workflow configuration

Only search beyond these instructions if you encounter specific issues not covered here or if the information appears outdated based on actual file contents.
