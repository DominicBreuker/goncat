# Copilot Instructions for goncat

## User Documentation

For a comprehensive user-oriented guide with practical examples and common use cases, see [`docs/USAGE.md`](../docs/USAGE.md). This guide is intended for end users of the goncat CLI tool.

## Project Overview

**goncat** is a netcat-like tool written in Go for creating bind or reverse shells with an SSH-like experience. Key features include:
- Encryption with mutual authentication (TLS)
- Cross-platform PTY support (Linux, Windows, macOS)
- Port forwarding (local/remote) with TCP and UDP protocol support
- SOCKS5 proxy support (TCP CONNECT and UDP ASSOCIATE)
- Session logging and automatic cleanup capabilities

**Repository Stats:**
- 158 Go files, ~20,000 lines of Go code
- Repository size: ~12MB (binaries in `dist/` excluded via .gitignore)
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
   - Build time: ~30-40 seconds for all platforms (~11s for Linux only)

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

### Linting

**Run all linters:**
```bash
make lint              # Runs fmt, vet, and staticcheck
```

**Individual linters:**
```bash
make fmt               # Auto-format all Go files
make vet               # Run go vet static analysis
make staticcheck       # Run staticcheck (installs if needed)
```

**Linting Notes:**
- `staticcheck` is automatically installed on first run via `go install honnef.co/go/tools/cmd/staticcheck@latest`
- Pre-existing `go vet` warning in `pkg/handler/socks/slave/associate.go:174:2: unreachable code` is known and can be ignored
- Always run `make fmt` before committing to ensure consistent formatting
- Address all staticcheck issues for new code; existing issues are being addressed incrementally

### Testing

We have three distinct types of tests with different purposes and mocking strategies:

**Unit Tests (in `*_test.go` files):**
```bash
make test-unit         # Unit tests with coverage report
go test -cover ./...   # Same as above
```
- **Location**: `*_test.go` files next to source code
- **Mocking**: Internal dependency injection with simple fakes in test files
- **Pattern**: Internal unexported functions accept injected dependencies (e.g., `masterListen()` with `serverFactory`)
- **Do NOT use**: `mocks/` package (that's for integration tests only)
- **Fast execution**: ~5 seconds
- **With race detection**: ~24 seconds (as run in CI)
- **Example**: `pkg/entrypoint/*_test.go` uses fake servers/clients

**Integration Tests (in `test/integration/`):**
```bash
go test ./test/integration/...  # Integration tests with mocked system resources
```
- **Location**: `test/integration/` directory
- **Mocking**: Use `mocks/` package (MockTCPNetwork, MockStdio, MockExec)
- **Pattern**: Pass mocks via `config.Dependencies` to entrypoint functions
- **Purpose**: Test complete tool workflows without real network/terminal I/O
- **Fast execution**: ~1-2 seconds
- **With race detection**: ~4 seconds (as run in CI)
- **IMPORTANT**: All integration tests must remain passing
- **See**: `test/integration/README.md` for details

**E2E Tests (in `test/e2e/`):**
```bash
make test              # All tests (unit + E2E)
make test-e2e          # Only E2E tests
```
- **Location**: `test/e2e/` directory with expect scripts
- **Mocking**: None - uses real compiled binaries
- **Environment**: Docker containers with Alpine Linux
- **Requirements**: Docker and Docker Compose
- **Duration**: ~8-9 minutes for full suite (as run in CI)
- **Scenarios**: Bind/reverse shell with tcp, ws, wss protocols
- **Partial E2E run**: To save time during development, run a single scenario:
  ```bash
  TRANSPORT='tcp' TEST_SET='master-connect' docker compose -f test/e2e/docker-compose.slave-listen.yml up --exit-code-from master
  ```

**Testing Guidelines:**
- See `TESTING.md` for comprehensive testing guidelines
- Unit tests: Use internal dependency injection, create simple fakes in test files
- Integration tests: Use `mocks/` package and `config.Dependencies`
- E2E tests: No mocking, validates real binaries
- All test types: Use table-driven tests, test behavior, cover edge cases
- **CRITICAL**: All tests must be race-free
  - Always run `go test -race ./...` before committing
  - CI runs with race detection - builds fail on race conditions
  - Protect shared state in fakes with `sync.Mutex`

### Known Build/Test Issues

1. **Race detection is mandatory** - All tests must pass `go test -race ./...` - CI will fail on race conditions
2. **Integration tests require Docker** - unit tests will pass without Docker but `make test` will fail
3. **KEY_SALT changes on each build** - this is intentional for security
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
- `entrypoint/` - Entry functions for the four operation modes (masterlisten, masterconnect, slavelisten, slaveconnect)
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

**test/** - Test infrastructure
- `integration/` - High-level integration tests with mocked dependencies
- `e2e/` - End-to-end tests using Docker (Alpine Linux with expect)
- `helpers/` - Common test utilities and helper functions

**mocks/** - Mock implementations for testing
- `mocktcp.go` - Mock TCP network using net.Pipe()
- `mockstdio.go` - Mock stdin/stdout for testing I/O

### Key Source Files

- `cmd/main.go` - Entry point, CLI app with master/slave subcommands
- `pkg/config/config.go` - Config structs, protocol constants (ProtoTCP/WS/WSS)
- `cmd/shared/shared.go` - CLI flag definitions
- `pkg/transport/` - Transport implementations (tcp, ws, udp) with function-based API
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
2. Setup Go 1.24
3. Run `make lint` (format, vet, staticcheck)
4. Run `make build` (builds all platforms)
5. Run `make test-unit-with-race` (~25 seconds)
6. Run `make test-integration-with-race` (~4 seconds)
7. Run `make test-e2e` (~8-9 minutes)

**Environment:** ubuntu-latest with Docker support

**Expected Runtime:** ~10-11 minutes total

## Code Quality Standards

### Documentation (godoc)
- Write godoc comments for all exported packages, functions, types, and constants
- Start comments with the name of the item: "ParseTransport parses a transport string..."
- Use full sentences ending with periods
- Each package needs a clear package comment describing its purpose
- Document special cases, but not internal implementation details
- See `TESTING.md` for testing standards

### Testing
- Follow guidelines in `TESTING.md` for all new code
- Use table-driven tests with subtests
- Test edge cases and error paths
- Maintain meaningful code coverage
- Keep tests fast and deterministic

### Linting
- Run `make lint` before committing
- All code should pass `go fmt`, `go vet`, and `staticcheck`
- Address any new issues reported by linters

## Validation Steps

Before committing changes, always follow these steps:

1. **Run linters:**
   ```bash
   make lint
   ```
   - This runs `go fmt`, `go vet`, and `staticcheck`
   - **REQUIRED** before every commit to ensure proper formatting and no issues

2. **Run unit tests:**
   ```bash
   make test-unit
   ```
   - Quick validation of code changes (~5 seconds)
   - Run with race detection for thorough validation: `make test-unit-with-race` (~24 seconds)

3. **Run integration tests:**
   ```bash
   make test-integration
   ```
   - Validates complete workflows (~1-2 seconds)
   - Run with race detection: `make test-integration-with-race` (~4 seconds)

4. **Build binaries (when finishing work on PR):**
   ```bash
   make build-linux    # Fast: ~11 seconds
   # OR for all platforms:
   make build          # Slower: ~30-40 seconds
   ```
   - Ensures the tool compiles successfully
   - Required before E2E tests

5. **Verify binaries (optional):**
   ```bash
   ls -lh dist/
   ./dist/goncat.elf version  # Should output: 0.0.1
   ```

6. **Run E2E tests (if time permits and significant changes made):**
   ```bash
   make test-e2e       # Full suite: ~8-9 minutes
   # OR run partial E2E for faster validation:
   TRANSPORT='tcp' TEST_SET='master-connect' docker compose -f test/e2e/docker-compose.slave-listen.yml up --exit-code-from master
   ```

## Common Tasks

**CLI flag:** Add to `cmd/shared/shared.go`, update `pkg/config/`, add tests to `cmd/shared/parsers_test.go`
**Transport protocol:** Create `pkg/transport/newproto/`, add `Dial()` and `ListenAndServe()` functions matching the pattern, update `pkg/net/`
**PTY changes:** Edit `pkg/pty/pty_unix.go` (Unix) or `pkg/pty/pty_windows.go` (Windows) - platform-specific

**Manual testing:**
```bash
make build-linux
./dist/goncat.elf slave listen 'tcp://*:12345'  # Terminal 1
./dist/goncat.elf master connect tcp://localhost:12345 --exec /bin/sh  # Terminal 2
```

## Important Notes

1. **Always run `make lint` before committing** - This is REQUIRED for every commit
2. **When finishing work on a pull request:**
   - Run `make build-linux` to ensure the tool compiles
   - Run `make test-unit` and `make test-integration` to ensure tests pass
   - If time permits and bigger changes were made, run partial E2E tests (see Validation Steps)
3. **Partial work is acceptable** - If you run out of time/tokens while working on a pull request:
   - Commit your partial work using the reporting mechanism
   - Clearly state in your response that the task is only partially completed
   - Describe the current status, progress made, and open TODOs
   - The remaining work can be completed in the next iteration
4. **Build generates random KEY_SALT each time** - Intentional for security
5. **Integration and E2E tests need Docker** - Unit tests will pass without Docker
6. **Binaries:** .elf (Linux), .exe (Windows), .macho (macOS) in gitignored `dist/`
7. **Static builds:** CGO disabled, no vendoring, dependencies via `go mod`
8. **Windows PTY quirks:** See `pkg/pty/pty_windows.go` comments
9. **Pre-existing go vet warning** - `pkg/handler/socks/slave/associate.go:174:2: unreachable code` exists in main branch, can be ignored

## Architecture

**Flow:** CLI (urfave/cli) → cmd/main.go → cmd/master or cmd/slave → pkg/net → pkg/transport (tcp/ws/udp with function-based API) → pkg/mux (yamux) → pkg/handler → pkg/exec or pkg/pty

**Master vs Slave:** Master controls params (exec, pty, logs, forwarding); Slave executes instructions. Both can listen or connect (4 combos).

**TLS:** Ephemeral certs per run, optional password-based mutual auth (`pkg/crypto/ca.go`)

**Transport API:** Simple function calls - `Dial(ctx, addr, timeout)` and `ListenAndServe(ctx, addr, timeout, handler)` replace old interface-based API

**For a comprehensive overview** of the system design, package relationships, architectural invariants, and data flow diagrams, see [`docs/ARCHITECTURE.md`](../docs/ARCHITECTURE.md).

## Trust These Instructions

These instructions have been validated by:
- Running all build commands successfully
- Executing unit tests (passing)
- Verifying binary creation for all platforms
- Reviewing all documentation and configuration files
- Testing the GitHub Actions workflow configuration

Only search beyond these instructions if you encounter specific issues not covered here or if the information appears outdated based on actual file contents.
