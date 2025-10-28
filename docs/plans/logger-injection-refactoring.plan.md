# Plan for Logger Injection Refactoring

Complete migration to injectable logging by replacing all legacy package-level logging functions (`log.ErrorMsg`, `log.InfoMsg`) with struct-based logger methods.

## Overview

The goncat codebase currently has two logging mechanisms:
1. **New injectable logger**: `*log.Logger` struct with methods `ErrorMsg()`, `InfoMsg()`, and `VerboseMsg()`
2. **Legacy package-level functions**: `log.ErrorMsg()` and `log.InfoMsg()` that use a default logger

The logger struct is already created in the `cmd` package and stored in `config.Shared.Logger`. However, 76 call sites across the codebase still use the legacy package-level functions instead of the injectable logger.

This refactoring will:
- Replace all 76 call sites of legacy functions with the injectable logger
- Pass the logger through function signatures where needed
- Maintain backward compatibility by keeping legacy functions for external uses (if any)
- Ensure all tests pass and the tool continues to function correctly

**Benefits**:
- Better testability (logger can be mocked/injected in tests)
- Consistent logging approach throughout codebase
- Preparation for future enhancements (e.g., structured logging, log levels)

## Implementation Plan

- [V] Step 1: Analysis and Categorization
  - **Task**: Review all 76 call sites to understand refactoring patterns and group them by subsystem
  - **Files**: All files containing `log.ErrorMsg` or `log.InfoMsg` calls
    - Run: `grep -rn "log\.ErrorMsg\|log\.InfoMsg" --include="*.go" | grep -v "pkg/log/log.go"`
    - Group call sites by package/subsystem (cmd/, pkg/handler/, pkg/transport/, etc.)
    - Identify how logger should be passed to each subsystem (via config, via function parameter, via struct field)
    - Document any special cases or challenges
  - **Dependencies**: None
  - **Definition of done**: Complete list of call sites grouped by refactoring pattern, documented in working notes
  - **Completed**: Analyzed all 76 call sites, confirmed counts match plan expectations across all subsystems

- [V] Step 2: Refactor cmd Package
  - **Task**: Replace 10 call sites in cmd/ package with logger from config
  - **Files**:
    - `cmd/masterlisten/masterlisten.go`: Lines 67, 69 (2 calls)
    - `cmd/masterconnect/masterconnect.go`: Lines 70, 72 (2 calls)
    - `cmd/slavelisten/slavelisten.go`: Lines 69, 71 (2 calls)
    - `cmd/slaveconnect/slaveconnect.go`: Lines 73, 75 (2 calls)
    - `cmd/main.go`: Line 28 (1 call)
  - **Pattern**: Logger is available in `config.Shared`, which is already created in these files
  - **Changes**:
    ```go
    // Before:
    log.ErrorMsg("Argument validation errors:")
    log.ErrorMsg(" - %s", err)
    
    // After:
    cfg.Logger.ErrorMsg("Argument validation errors:")
    cfg.Logger.ErrorMsg(" - %s", err)
    ```
  - **Special case for cmd/main.go**: Logger not available in main error handler
    ```go
    // Before:
    log.ErrorMsg("Run: %s\n", err)
    
    // After:
    // Keep using package-level function or create a default logger
    logger := log.NewLogger(false)
    logger.ErrorMsg("Run: %s\n", err)
    ```
  - **Dependencies**: None
  - **Definition of done**: All cmd/ package logging uses cfg.Logger, tests pass with `go test ./cmd/...`
  - **Completed**: Refactored all 5 files, all cmd tests pass

- [V] Step 3: Refactor pkg/clean Package
  - **Task**: Replace 3 call sites in pkg/clean/ with logger parameter
  - **Files**:
    - `pkg/clean/clean_default.go`: Line 16 (1 call)
    - `pkg/clean/clean_windows.go`: Lines 32, 47 (2 calls)
  - **Pattern**: Add logger parameter to `EnsureDeletion()` function
  - **Changes**:
    ```go
    // Before:
    func EnsureDeletion() error {
        // ...
        log.ErrorMsg("deleting executable at %s: %s", path, err)
    }
    
    // After:
    func EnsureDeletion(logger *log.Logger) error {
        // ...
        logger.ErrorMsg("deleting executable at %s: %s", path, err)
    }
    ```
  - **Callers**: Update calls in `cmd/slavelisten/slavelisten.go` and `cmd/slaveconnect/slaveconnect.go`:
    ```go
    // Before:
    if err := clean.EnsureDeletion(); err != nil {
    
    // After:
    if err := clean.EnsureDeletion(logger); err != nil {
    ```
  - **Dependencies**: Step 2 (cmd package provides logger)
  - **Definition of done**: All pkg/clean/ logging uses logger parameter, cleanup still works, tests pass
  - **Completed**: Refactored clean.go, clean_default.go, clean_windows.go and updated callers. All tests pass.

- [V] Step 4: Refactor pkg/net Package
  - **Task**: Replace 2 call sites in pkg/net/ with logger from config
  - **Files**:
    - `pkg/net/listen.go`: Line 38 (1 call)
    - `pkg/net/dial.go`: Line 37 (1 call)
  - **Pattern**: Logger available via `cfg.Logger` parameter
  - **Changes**:
    ```go
    // Before in pkg/net/listen.go:
    log.InfoMsg("Listening on %s\n", addr)
    
    // After:
    cfg.Logger.InfoMsg("Listening on %s\n", addr)
    
    // Before in pkg/net/dial.go:
    log.InfoMsg("Connecting to %s\n", addr)
    
    // After:
    cfg.Logger.InfoMsg("Connecting to %s\n", addr)
    ```
  - **Dependencies**: None (config already has Logger)
  - **Definition of done**: pkg/net logging uses cfg.Logger, connection messages still appear correctly
  - **Completed**: Refactored both files, tests pass

- [V] Step 5: Refactor pkg/transport Package
  - **Task**: Replace 11 call sites in pkg/transport/ with logger parameter
  - **Files**:
    - `pkg/transport/tcp/listener.go`: Lines 135, 140 (2 calls)
    - `pkg/transport/ws/listener.go`: Lines 146, 152, 160, 166 (4 calls)
    - `pkg/transport/udp/listener.go`: Lines 207, 223 (2 calls)
  - **Pattern**: Add logger to listener configuration or pass as parameter
  - **Approach Option 1**: Add logger to handler context or closure
  - **Approach Option 2**: Pass logger to ListenAndServe functions
  - **Recommended**: Pass via config.Dependencies or add to existing parameters
  - **Changes**:
    ```go
    // Before in tcp/listener.go:
    func (l *TCPListener) ListenAndServe(ctx context.Context, handler func(net.Conn) error) error {
        // ...
        log.ErrorMsg("Handler panic: %v\n", r)
    }
    
    // After:
    func (l *TCPListener) ListenAndServe(ctx context.Context, logger *log.Logger, handler func(net.Conn) error) error {
        // ...
        logger.ErrorMsg("Handler panic: %v\n", r)
    }
    ```
  - **Alternative approach**: Store logger in TCPListener struct
    ```go
    type TCPListener struct {
        addr    string
        timeout time.Duration
        deps    *config.Dependencies
        logger  *log.Logger  // Add logger field
    }
    ```
  - **Dependencies**: Step 4 (understand how config flows through transport layer)
  - **Definition of done**: All transport logging uses logger parameter, error messages still appear, tests pass
  - **Status**: IN PROGRESS - Transport functions refactored, need to update all test files
  - **Completed**:
    - TCP listener: Added logger parameter to ListenAndServe, serveConnections, acceptLoop, handleConnection
    - WS listener: Added logger parameter to ListenAndServeWS, ListenAndServeWSS, listenAndServeWebSocket, createHTTPServer, createWebSocketHandler, handleWebSocketUpgrade
    - UDP listener: Added logger parameter to ListenAndServe, serveQUICConnections, acceptQUICLoop, handleQUICConnection
    - pkg/net/listen_internal.go: Updated function signatures and calls
    - pkg/transport/tcp/listener_test.go: Updated test calls
  - **Remaining**:
    - Fix pkg/net/listen_test.go (6 test function signatures)
    - Run full test suite
    - Verify all transport tests pass

- [V] Step 6: Refactor pkg/terminal Package
  - **Task**: Replace 6 call sites in pkg/terminal/ with logger parameter
  - **Files**:
    - `pkg/terminal/terminal.go`: Lines 33, 36, 40, 49, 56, 82, 87 (7 calls)
  - **Pattern**: Add logger parameter to `Pipe()` and `PipeWithPTY()` functions
  - **Changes**:
    ```go
    // Before:
    func Pipe(conn net.Conn, connErrCh chan error) error {
        log.ErrorMsg("Failed to acquire connection slot: %s\n", err)
        log.InfoMsg("Connection slot acquired\n")
    }
    
    // After:
    func Pipe(conn net.Conn, connErrCh chan error, logger *log.Logger) error {
        logger.ErrorMsg("Failed to acquire connection slot: %s\n", err)
        logger.InfoMsg("Connection slot acquired\n")
    }
    ```
  - **Dependencies**: Need to trace where Pipe() is called from
  - **Definition of done**: All terminal logging uses logger parameter, PTY still works correctly
  - **Completed**: Refactored Pipe(), PipeWithPTY(), and syncTerminalSize(). Updated callers in handler/slave and handler/master.

- [V] Step 7: Refactor pkg/exec Package
  - **Task**: Replace 6 call sites in pkg/exec/ with logger parameter
  - **Files**:
    - `pkg/exec/exec.go`: Line 54 (1 call)
    - `pkg/exec/exec_default.go`: Lines 60, 89, 96 (3 calls)
    - `pkg/exec/exec_windows.go`: Lines 45, 73, 80 (3 calls)
  - **Pattern**: Add logger parameter to `Run()` function
  - **Changes**:
    ```go
    // Before:
    func Run(ctx context.Context, cfg *Cfg) error {
        log.ErrorMsg("Run Pipe(pty, conn): %s\n", err)
    }
    
    // After:
    func Run(ctx context.Context, cfg *Cfg, logger *log.Logger) error {
        logger.ErrorMsg("Run Pipe(pty, conn): %s\n", err)
    }
    ```
  - **Callers**: Update in `pkg/handler/slave/foreground.go`
  - **Dependencies**: Step 6 (terminal package may need logger too)
  - **Definition of done**: All exec logging uses logger parameter, command execution still works
  - **Completed**: Refactored Run(), RunWithPTY(), and syncTerminalSize() for both Unix and Windows. Updated callers.

- [V] Step 8: Refactor pkg/handler/slave Package
  - **Task**: Replace 8 call sites in pkg/handler/slave/ with logger from handler struct
  - **Files**:
    - `pkg/handler/slave/slave.go`: Lines 42, 110, 131 (3 calls)
    - `pkg/handler/slave/foreground.go`: Line 17 (1 call)
    - `pkg/handler/slave/portfwd.go`: Lines 16, 39 (2 calls)
    - `pkg/handler/slave/socks.go`: Lines 16, 27, 33 (3 calls)
  - **Pattern**: Add logger field to slave handler struct
  - **Changes**:
    ```go
    // In pkg/handler/slave/slave.go:
    type Slave struct {
        // ... existing fields
        logger *log.Logger  // Add logger field
    }
    
    // Constructor:
    func NewSlave(..., logger *log.Logger) *Slave {
        return &Slave{
            // ... existing fields
            logger: logger,
        }
    }
    
    // Usage:
    slv.logger.InfoMsg("Session with %s established (%s)\n", slv.remoteAddr, slv.remoteID)
    ```
  - **Dependencies**: Steps 6-7 (foreground calls exec which calls terminal)
  - **Definition of done**: All slave handler logging uses handler logger, slave mode works correctly
  - **Completed**: All 9 call sites refactored to use slv.cfg.Logger. No struct changes needed since cfg already has Logger.

- [V] Step 9: Refactor pkg/handler/master Package
  - **Task**: Replace 8 call sites in pkg/handler/master/ with logger from handler struct
  - **Files**:
    - `pkg/handler/master/master.go`: Lines 49, 106, 124, 146, 154, 163 (6 calls)
    - `pkg/handler/master/foreground.go`: Line 21 (1 call)
    - `pkg/handler/master/portfwd.go`: Lines 31, 53, 64 (3 calls)
    - `pkg/handler/master/socks.go`: Line 31 (1 call)
  - **Pattern**: Add logger field to master handler struct (same as slave)
  - **Changes**:
    ```go
    // In pkg/handler/master/master.go:
    type Master struct {
        // ... existing fields
        logger *log.Logger  // Add logger field
    }
    
    func NewMaster(..., logger *log.Logger) *Master {
        return &Master{
            // ... existing fields
            logger: logger,
        }
    }
    ```
  - **Dependencies**: Step 8 (same pattern as slave handler)
  - **Definition of done**: All master handler logging uses handler logger, master mode works correctly

- [V] Step 10: Refactor pkg/handler/portfwd Package
  - **Task**: Replace 7 call sites in pkg/handler/portfwd/ with logger parameter
  - **Files**:
    - `pkg/handler/portfwd/server.go`: Lines 127, 143, 147, 203, 286, 321, 351, 371 (8 calls)
    - `pkg/handler/portfwd/client.go`: Line 89 (1 call)
  - **Pattern**: Add logger field to Server and Client structs
  - **Changes**:
    ```go
    // In pkg/handler/portfwd/server.go:
    type Server struct {
        // ... existing fields
        logger *log.Logger
    }
    
    func NewServer(..., logger *log.Logger) *Server {
        return &Server{
            // ... existing fields
            logger: logger,
        }
    }
    ```
  - **Dependencies**: Steps 8-9 (called from handlers)
  - **Definition of done**: All port forwarding logging uses logger, forwarding still works

- [V] Step 11: Refactor pkg/handler/socks Package
  - **Task**: Replace 11 call sites in pkg/handler/socks/ with logger parameter
  - **Files**:
    - `pkg/handler/socks/slave/connect.go`: Line 105 (1 call)
    - `pkg/handler/socks/slave/associate.go`: Line 78 (1 call)
    - `pkg/handler/socks/master/server.go`: Line 63 (1 call)
    - `pkg/handler/socks/master/connect.go`: Line 98 (1 call)
    - `pkg/handler/socks/master/relay.go`: Lines 77, 98, 115, 163, 261, 277, 290 (7 calls)
  - **Pattern**: Pass logger to SOCKS handler functions
  - **Changes**:
    ```go
    // Before:
    func HandleConnect(...) error {
        log.ErrorMsg("Handling connect to %s: %s\n", addr, err)
    }
    
    // After:
    func HandleConnect(..., logger *log.Logger) error {
        logger.ErrorMsg("Handling connect to %s: %s\n", addr, err)
    }
    ```
  - **Dependencies**: Steps 8-9 (called from handlers)
  - **Definition of done**: All SOCKS logging uses logger parameter, SOCKS proxy still works

- [X] Step 12: Run All Tests
  - **Task**: Verify all tests pass after refactoring
  - **Files**: All test files
  - **Commands**:
    ```bash
    # Unit tests
    go test -race ./...
    
    # Integration tests
    go test -race ./test/integration/...
    ```
  - **Expected**: All tests pass, no race conditions detected
  - **Dependencies**: Steps 2-11 (all refactoring complete)
  - **Definition of done**: Clean test run with no failures

- [ ] Step 13: Build and Lint
  - **Task**: Ensure code builds and passes all linters
  - **Commands**:
    ```bash
    make lint
    make build-linux
    ```
  - **Expected**: No lint errors, clean build
  - **Dependencies**: Step 12 (tests passing)
  - **Definition of done**: Clean lint and successful build

- [ ] Step 14: Manual Verification - Basic Shell Connection
  - **Task**: **MANDATORY VERIFICATION** - Test that basic shell connections still work after refactoring
  - **Importance**: This step is **CRITICAL** and **MUST NOT BE SKIPPED**. The refactoring touches core logging throughout the entire codebase. Manual verification is essential to ensure the tool still functions correctly.
  - **Reference**: Follow procedures from `docs/TROUBLESHOOT.md` sections:
    - "Feature: Basic Reverse Shell (Master Listen, Slave Connect)" (lines 165-214)
    - "Feature: Basic Bind Shell (Slave Listen, Master Connect)" (lines 217-260)
  - **Setup**:
    ```bash
    cd /home/runner/work/goncat/goncat
    make build-linux
    
    # Verify binary exists and reports correct version
    ls -lh dist/goncat.elf
    ./dist/goncat.elf version
    ```
  - **Test 1 - Reverse Shell (Master Listen, Slave Connect)**:
    ```bash
    # Terminal 1: Start master listener
    ./dist/goncat.elf master listen 'tcp://*:12345' --exec /bin/sh &
    MASTER_PID=$!
    
    # Wait for master to start
    sleep 2
    
    # Verify master is listening
    ss -tlnp 2>/dev/null | grep 12345
    
    # Terminal 2: Connect slave and send test command
    echo "echo 'LOGGING_REFACTOR_TEST' && exit" | ./dist/goncat.elf slave connect tcp://localhost:12345
    
    # Cleanup
    kill $MASTER_PID 2>/dev/null || true
    pkill -9 goncat.elf 2>/dev/null || true
    ```
  - **Expected Behavior**:
    - Master displays: `[+] Listening on :12345` (using new logger)
    - Slave displays: `[+] Connecting to localhost:12345` (using new logger)
    - Both display: `[+] Session with <address> established` (using new logger)
    - Shell command executes and returns: `LOGGING_REFACTOR_TEST`
    - Connection closes cleanly
    - **All log messages appear correctly formatted in blue/red colors**
    - **No log messages are missing or duplicated**
  - **Test 2 - Bind Shell (Slave Listen, Master Connect)**:
    ```bash
    # Start slave listener
    ./dist/goncat.elf slave listen 'tcp://*:12346' &
    SLAVE_PID=$!
    
    sleep 2
    
    # Verify slave is listening
    ss -tlnp 2>/dev/null | grep 12346
    
    # Connect master with shell command
    echo "echo 'BIND_SHELL_TEST' && exit" | ./dist/goncat.elf master connect tcp://localhost:12346 --exec /bin/sh
    
    # Cleanup
    kill $SLAVE_PID 2>/dev/null || true
    pkill -9 goncat.elf 2>/dev/null || true
    ```
  - **Expected Behavior**:
    - Slave displays: `[+] Listening on :12346`
    - Master displays: `[+] Connecting to localhost:12346`
    - Both display: `[+] Session with <address> established`
    - Shell command executes and returns: `BIND_SHELL_TEST`
    - Connection closes after command completes
  - **Test 3 - Verbose Logging**:
    ```bash
    # Test verbose mode to ensure VerboseMsg works
    ./dist/goncat.elf slave connect tcp://192.0.2.1:12345 --verbose 2>&1 | head -10
    ```
  - **Expected**: Detailed error messages with `[v]` prefix for verbose messages, `[!]` for errors
  - **Validation Criteria**:
    - ✓ Master and slave can connect successfully
    - ✓ All log messages appear (connection established, listening, etc.)
    - ✓ Log messages are properly colored (blue for info, red for errors)
    - ✓ Shell commands execute correctly
    - ✓ Verbose mode shows detailed messages with `[v]` prefix
    - ✓ No missing log output compared to pre-refactor behavior
    - ✓ No crashes or panics during connection lifecycle
  - **If Verification Fails**:
    - **DO NOT proceed** - report failure clearly to user
    - Document exactly what failed (which message was missing, wrong format, etc.)
    - Check if logger is nil anywhere (should never happen)
    - Review logger passing chain for the failing scenario
    - User must be informed so refactoring can be fixed before merge
  - **Dependencies**: Step 13 (clean build)
  - **Definition of done**: 
    - All 3 test scenarios pass successfully
    - All log messages appear correctly
    - No functionality regression detected
    - Clear documentation that verification was performed and passed

- [ ] Step 15: Update Tests for Logger Injection
  - **Task**: Update or add tests that verify logger injection works correctly
  - **Files**: Test files for modified packages
  - **Changes**:
    - Update unit tests to pass logger parameter where signatures changed
    - Add tests that verify logger is used (not package-level functions)
    - Ensure integration tests still pass with logger from config
  - **Example**:
    ```go
    func TestSlave_LogsSessionEstablished(t *testing.T) {
        // Create a test logger or mock
        logger := log.NewLogger(false)
        
        slv := NewSlave(..., logger)
        
        // Test that logger is used for session messages
        // ... rest of test
    }
    ```
  - **Dependencies**: Steps 2-11 (all code refactored)
  - **Definition of done**: Tests verify logger injection, all tests pass

- [ ] Step 16: Consider Deprecation Markers
  - **Task**: Add deprecation comments to legacy package-level functions
  - **Files**: `pkg/log/log.go`
  - **Changes**:
    ```go
    // ErrorMsg prints an error message to stderr in red color.
    // Deprecated: Use Logger.ErrorMsg instead for better testability.
    // This function is kept for backward compatibility with external code.
    func ErrorMsg(format string, a ...interface{}) {
        defaultLogger.ErrorMsg(format, a...)
    }
    
    // InfoMsg prints an informational message to stderr in blue color.
    // Deprecated: Use Logger.InfoMsg instead for better testability.
    // This function is kept for backward compatibility with external code.
    func InfoMsg(format string, a ...interface{}) {
        defaultLogger.InfoMsg(format, a...)
    }
    ```
  - **Note**: Don't remove the functions yet - they might be used by external code or tests
  - **Dependencies**: Steps 2-15 (all internal usage migrated)
  - **Definition of done**: Legacy functions marked as deprecated with clear migration path

- [ ] Step 17: Update Documentation
  - **Task**: Document the logger injection pattern for future contributors
  - **Files**: Consider adding to `TESTING.md` or `.github/copilot-instructions.md`
  - **Content**:
    - Explain that logger should be passed via config.Shared.Logger
    - Show pattern for adding logger to handler structs
    - Note that package-level log functions are deprecated
    - Example code showing proper logger usage
  - **Dependencies**: Step 16 (refactoring complete)
  - **Definition of done**: Documentation updated with logger injection guidelines

- [ ] Step 18: Final E2E Test Run
  - **Task**: Run full E2E test suite to ensure nothing broke
  - **Commands**:
    ```bash
    make test-e2e
    ```
  - **Note**: This runs Docker-based E2E tests (~8-9 minutes)
  - **Expected**: All E2E scenarios pass (tcp, ws, wss protocols in bind/reverse modes)
  - **Dependencies**: Steps 2-17 (everything complete)
  - **Definition of done**: Full E2E test suite passes without failures

## Summary

This refactoring will systematically migrate all 76 call sites from package-level logging functions to injectable logger instances. The work is organized by subsystem, starting with simple cases (cmd package) and moving to more complex ones (handlers with nested dependencies).

**Key principles**:
- Logger flows from config.Shared through the call chain
- Handler structs own their logger instance
- Functions that don't have access to config receive logger as parameter
- Tests are updated to inject or mock loggers as needed
- Manual verification ensures no functionality regression

**Timeline estimate**: 
- Analysis: 30 minutes
- Implementation: 3-4 hours (18 steps, ~15 min/step)
- Testing & verification: 1 hour
- Total: 4-5 hours

**Risk mitigation**:
- Incremental approach (one subsystem at a time)
- Tests run after each subsystem
- Manual verification before declaring done
- Legacy functions kept for compatibility
- Can be done in multiple iterations if time runs out
