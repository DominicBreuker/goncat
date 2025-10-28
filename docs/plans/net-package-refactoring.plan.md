# Plan for Net Package Refactoring

Refactor the `pkg/client` and `pkg/server` packages into a unified `pkg/net` package with simplified APIs (`Dial` and `ListenAndServe`), improve timeout handling to ensure timeouts are cleared after operations complete, and enhance code readability by splitting long functions into well-named private functions.

## Overview

Currently, the `pkg/client` and `pkg/server` packages expose struct-based APIs that require callers to:
1. Create a client/server instance with `client.New()` or `server.New()`
2. Call `Connect()` or `Serve()` to establish/accept connections
3. Call `GetConnection()` to retrieve the connection (client only)
4. Remember to call `Close()` to clean up resources

This creates an unnecessarily complicated dance for callers. This refactoring simplifies the API by:

1. **Creating a new `pkg/net` package** with two public functions:
   - `Dial(ctx, cfg) (net.Conn, error)` - Client implementation that returns the connection directly
   - `ListenAndServe(ctx, cfg, handler) error` - Server implementation that blocks until done

2. **Improving timeout handling** to ensure:
   - Timeouts are set before potentially blocking network operations
   - Timeouts are cleared immediately after operations complete
   - Healthy connections are never killed due to lingering timeouts

3. **Enhancing code readability** by:
   - Splitting long functions into smaller, well-named private functions
   - Using descriptive function names that explain intent
   - Reducing function complexity through logical decomposition
   - Making the code read like a book for human reviewers

4. **Migrating all call sites** in `pkg/entrypoint` to use the new API

5. **Removing old packages** and updating documentation

The new API keeps all implementation details (structs, timeout management, cleanup) private, so callers don't have to worry about them.

## Implementation Plan

- [V] **Step 1: Create the new `pkg/net` package structure**
  - **Task**: Create the foundation of the new `pkg/net` package with the basic file structure and API signatures. This establishes the public interface that will replace the old client/server packages.
  - **Files**:
    - `pkg/net/dial.go`: New file with `Dial` function signature
      ```go
      package net
      
      import (
          "context"
          "net"
          
          "dominicbreuker/goncat/pkg/config"
      )
      
      // Dial establishes a connection to the configured remote address.
      // It supports TCP, WebSocket, and UDP protocols with optional TLS encryption.
      // The function handles all connection setup, TLS upgrades, timeout management,
      // and cleanup internally. Returns the established connection or an error.
      //
      // The context can be used to cancel the dial operation at any time.
      // Timeouts for individual operations are controlled by cfg.Timeout.
      func Dial(ctx context.Context, cfg *config.Shared) (net.Conn, error) {
          // Implementation in Step 2
          return nil, nil
      }
      ```
    - `pkg/net/listen.go`: New file with `ListenAndServe` function signature
      ```go
      package net
      
      import (
          "context"
          
          "dominicbreuker/goncat/pkg/config"
          "dominicbreuker/goncat/pkg/transport"
      )
      
      // ListenAndServe creates a listener on the configured address and serves
      // incoming connections using the provided handler function.
      // It supports TCP, WebSocket, and UDP protocols with optional TLS encryption.
      // The function blocks until the context is cancelled or an error occurs.
      //
      // The handler function is called for each accepted connection. It should
      // handle the connection and return when done. The connection will be
      // closed after the handler returns.
      //
      // All cleanup, timeout management, and resource lifecycle is handled
      // internally. Callers only need to provide the handler logic.
      func ListenAndServe(ctx context.Context, cfg *config.Shared, handler transport.Handler) error {
          // Implementation in Step 3
          return nil
      }
      ```
    - `pkg/net/dial_test.go`: New file with initial test structure
      ```go
      package net
      
      import (
          "context"
          "testing"
          
          "dominicbreuker/goncat/pkg/config"
      )
      
      func TestDial_PlaceholderForStep2(t *testing.T) {
          t.Skip("Placeholder - implementation in Step 2")
      }
      ```
    - `pkg/net/listen_test.go`: New file with initial test structure
      ```go
      package net
      
      import (
          "context"
          "testing"
          
          "dominicbreuker/goncat/pkg/config"
      )
      
      func TestListenAndServe_PlaceholderForStep3(t *testing.T) {
          t.Skip("Placeholder - implementation in Step 3")
      }
      ```
  - **Dependencies**: None
  - **Definition of done**: 
    - Package `pkg/net` exists with `dial.go` and `listen.go` files
    - Both files have stub implementations with proper godoc comments
    - Test files exist with placeholder tests
    - Code compiles without errors: `go build ./pkg/net`
    - No lint warnings: `make lint` passes

- [V] **Step 2: Implement Dial function with proper timeout handling**
  - **Task**: Implement the `Dial` function by extracting and refactoring code from `pkg/client/client.go`. Focus on clean, readable code with private helper functions for each logical step. Pay special attention to timeout handling - set timeouts before blocking operations and clear them immediately after success.
  - **Files**:
    - `pkg/net/dial.go`: Implement `Dial` function
      - Extract dialer creation logic into private function `createDialer(ctx, cfg, deps) (transport.Dialer, error)`
      - Extract connection establishment into `establishConnection(ctx, dialer, cfg) (net.Conn, error)` 
        - **Critical**: Set deadline before `Dial()`, clear immediately after success
        - **Pattern**: 
          ```go
          if cfg.Timeout > 0 {
              // Set deadline only for the Dial operation
              deadline := time.Now().Add(cfg.Timeout)
              // Deadline will be cleared after successful dial
          }
          conn, err := dialer.Dial(ctx)
          if err != nil {
              return nil, err
          }
          // Clear deadline immediately - connection is healthy
          if cfg.Timeout > 0 {
              _ = conn.SetDeadline(time.Time{})
          }
          ```
      - Extract TLS upgrade into `upgradeTLS(conn, cfg) (net.Conn, error)`
        - **Critical**: Set deadline before handshake, clear immediately after
        - Follow same pattern: set deadline, perform operation, clear deadline on success
      - Main `Dial` function orchestrates these helpers with clear flow
      - Keep certificate verification as private helper `verifyPeerCertificate`
    - `pkg/net/dial_internal.go`: New file for private helper functions
      ```go
      package net
      
      // Private helper functions for Dial implementation
      // Each function has a single, well-defined responsibility
      
      import (
          "context"
          "crypto/tls"
          "crypto/x509"
          "fmt"
          "net"
          "time"
          
          "dominicbreuker/goncat/pkg/config"
          "dominicbreuker/goncat/pkg/crypto"
          "dominicbreuker/goncat/pkg/format"
          "dominicbreuker/goncat/pkg/log"
          "dominicbreuker/goncat/pkg/transport"
          "dominicbreuker/goncat/pkg/transport/tcp"
          "dominicbreuker/goncat/pkg/transport/udp"
          "dominicbreuker/goncat/pkg/transport/ws"
      )
      
      // dialDependencies holds injectable dependencies for testing
      type dialDependencies struct {
          newTCPDialer func(string, *config.Dependencies) (transport.Dialer, error)
          newWSDialer  func(context.Context, string, config.Protocol) transport.Dialer
          newUDPDialer func(string, time.Duration) (transport.Dialer, error)
      }
      
      // createDialer creates the appropriate transport dialer based on protocol
      func createDialer(ctx context.Context, cfg *config.Shared, deps *dialDependencies) (transport.Dialer, error) {
          // Implementation: switch on protocol, create appropriate dialer
      }
      
      // establishConnection dials using the provided dialer with timeout handling
      func establishConnection(ctx context.Context, dialer transport.Dialer, cfg *config.Shared) (net.Conn, error) {
          // Implementation: dial with timeout, clear deadline on success
      }
      
      // upgradeTLS wraps the connection with TLS, handling timeouts properly
      func upgradeTLS(conn net.Conn, cfg *config.Shared) (net.Conn, error) {
          // Implementation: TLS handshake with timeout, clear deadline on success
      }
      
      // buildTLSConfig creates the TLS configuration for the client
      func buildTLSConfig(key string, logger *log.Logger) (*tls.Config, error) {
          // Implementation: configure TLS with optional mutual auth
      }
      
      // performTLSHandshake performs the TLS handshake with proper timeout handling
      func performTLSHandshake(tlsConn *tls.Conn, timeout time.Duration, logger *log.Logger) error {
          // Implementation: handshake with deadline, clear on success
      }
      
      // verifyPeerCertificate validates server certificate against CA pool
      func verifyPeerCertificate(caCert *x509.CertPool, rawCerts [][]byte) error {
          // Implementation: custom certificate verification
      }
      ```
    - `pkg/net/dial_test.go`: Comprehensive unit tests
      - Test successful dial for each protocol (TCP, WS, WSS, UDP)
      - Test TLS upgrade with and without mutual auth
      - Test timeout handling (verify deadlines are set and cleared)
      - Test error cases (connection refused, TLS handshake failure, etc.)
      - Test context cancellation
      - Use table-driven tests with dependency injection pattern from existing client tests
  - **Dependencies**: Step 1
  - **Definition of done**: 
    - `Dial` function fully implemented with clean, readable code
    - Private helper functions are small (<50 lines each) and well-named
    - Timeouts are set before blocking operations and cleared immediately after
    - All unit tests pass: `go test ./pkg/net -v`
    - All tests pass with race detector: `go test -race ./pkg/net`
    - No lint warnings: `make lint` passes
    - Code is readable - a human can understand the flow by reading function names
    - Manual code review confirms timeout handling is correct

- [V] **Step 3: Implement ListenAndServe function with proper timeout handling**
  - **Task**: Implement the `ListenAndServe` function by extracting and refactoring code from `pkg/server/server.go`. Use private helper functions to keep the main function clean and readable. Ensure TLS handshake timeouts are properly managed.
  - **Files**:
    - `pkg/net/listen.go`: Implement `ListenAndServe` function
      - Extract listener creation into `createListener(ctx, cfg) (transport.Listener, error)`
      - Extract TLS handler wrapping into `wrapWithTLS(handler, cfg) (transport.Handler, error)`
        - **Critical**: Set deadline before TLS handshake, clear immediately after
        - Follow same pattern as Dial: set deadline, perform operation, clear deadline
      - Main function orchestrates: create listener, wrap handler (if TLS), serve
      - Handle context cancellation and graceful shutdown
    - `pkg/net/listen_internal.go`: New file for private helper functions
      ```go
      package net
      
      // Private helper functions for ListenAndServe implementation
      
      import (
          "context"
          "crypto/tls"
          "fmt"
          "net"
          "time"
          
          "dominicbreuker/goncat/pkg/config"
          "dominicbreuker/goncat/pkg/crypto"
          "dominicbreuker/goncat/pkg/format"
          "dominicbreuker/goncat/pkg/log"
          "dominicbreuker/goncat/pkg/transport"
          "dominicbreuker/goncat/pkg/transport/tcp"
          "dominicbreuker/goncat/pkg/transport/udp"
          "dominicbreuker/goncat/pkg/transport/ws"
      )
      
      // createListener creates the appropriate transport listener based on protocol
      func createListener(ctx context.Context, cfg *config.Shared) (transport.Listener, error) {
          // Implementation: switch on protocol, create appropriate listener
      }
      
      // wrapWithTLS wraps the handler with TLS, handling timeouts properly
      func wrapWithTLS(handler transport.Handler, cfg *config.Shared) (transport.Handler, error) {
          // Implementation: return wrapped handler that does TLS handshake
      }
      
      // buildServerTLSConfig creates the TLS configuration for the server
      func buildServerTLSConfig(key string, logger *log.Logger) (*tls.Config, error) {
          // Implementation: configure TLS with optional client auth
      }
      
      // performServerTLSHandshake performs the server-side TLS handshake with timeout handling
      func performServerTLSHandshake(tlsConn *tls.Conn, timeout time.Duration, logger *log.Logger) error {
          // Implementation: handshake with deadline, clear on success
      }
      ```
    - `pkg/net/listen_test.go`: Comprehensive unit tests
      - Test successful listen for each protocol
      - Test TLS wrapping with and without client auth
      - Test timeout handling for TLS handshakes
      - Test context cancellation and graceful shutdown
      - Test handler execution
      - Use table-driven tests with dependency injection
  - **Dependencies**: Step 1
  - **Definition of done**: 
    - `ListenAndServe` function fully implemented with clean, readable code
    - Private helper functions are small and well-named
    - TLS handshake timeouts are properly set and cleared
    - All unit tests pass: `go test ./pkg/net -v`
    - All tests pass with race detector: `go test -race ./pkg/net`
    - No lint warnings: `make lint` passes
    - Code is readable and well-structured
    - Manual code review confirms timeout handling is correct

- [V] **Step 4: Perform code readability review and refactoring**
  - **Task**: Review all code in `pkg/net` specifically for readability and clarity. Split any remaining long or complex functions into smaller, well-named private functions. Ensure the code reads like a book.
  - **Files**:
    - All files in `pkg/net/`:
      - Review function length (target: <50 lines per function)
      - Review function complexity (target: cyclomatic complexity <10)
      - Review function names (must clearly describe purpose)
      - Split complex functions into logical steps with descriptive names
      - Add inline comments only where truly needed to explain "why", not "what"
      - Ensure error messages are clear and actionable
  - **Review checklist**:
    - [ ] Can a human understand what each function does from its name alone?
    - [ ] Are functions focused on a single responsibility?
    - [ ] Is the control flow clear and easy to follow?
    - [ ] Are edge cases handled explicitly with clear error messages?
    - [ ] Are timeout operations easy to identify and verify?
    - [ ] Could a new developer understand this code without asking questions?
  - **Dependencies**: Steps 2, 3
  - **Definition of done**: 
    - All functions in `pkg/net` are <50 lines (excluding tests)
    - Function names clearly describe their purpose
    - Code passes readability review by a human reviewer
    - No "clever" code - everything is explicit and clear
    - Tests remain passing after any refactoring

- [X] **Step 5: Update pkg/entrypoint to use new Dial function**
  - **Task**: Migrate `masterconnect.go` and `slaveconnect.go` to use the new `pkg/net.Dial` function instead of creating client instances. This simplifies the code significantly.
  - **Files**:
    - `pkg/entrypoint/internal.go`: 
      - Remove `clientInterface` interface
      - Remove `clientFactory` type
      - Remove `realClientFactory()` function
      - Keep server-related types (they'll be updated in Step 6)
    - `pkg/entrypoint/masterconnect.go`: 
      - Remove import of `"dominicbreuker/goncat/pkg/client"`
      - Add import of `netpkg "dominicbreuker/goncat/pkg/net"` (aliased to avoid conflict with stdlib "net")
      - Change `MasterConnect` to call `netpkg.Dial` directly:
        ```go
        func MasterConnect(ctx context.Context, cfg *config.Shared, mCfg *config.Master) error {
            return masterConnect(ctx, cfg, mCfg, netpkg.Dial, master.Handle)
        }
        ```
      - Update `masterConnect` signature to accept a dial function instead of client factory:
        ```go
        func masterConnect(
            parent context.Context,
            cfg *config.Shared,
            mCfg *config.Master,
            dial func(context.Context, *config.Shared) (net.Conn, error),
            handle masterHandler,
        ) error {
            ctx, cancel := context.WithCancel(parent)
            defer cancel()
            
            conn, err := dial(ctx, cfg)
            if err != nil {
                return fmt.Errorf("dialing: %w", err)
            }
            var closeOnce sync.Once
            closeConn := func() { closeOnce.Do(func() { _ = conn.Close() }) }
            defer closeConn()
            
            // Run handler...
        }
        ```
      - Remove all client creation, Connect(), GetConnection() calls
    - `pkg/entrypoint/slaveconnect.go`:
      - Apply same changes as masterconnect.go
      - Update `SlaveConnect` and `slaveConnect` functions similarly
    - `pkg/entrypoint/masterconnect_test.go`:
      - Update tests to use fake dial function instead of fake client
      - Simplify test setup (no more client mocks needed)
    - `pkg/entrypoint/slaveconnect_test.go`:
      - Apply same test updates as masterconnect_test.go
  - **Dependencies**: Step 2
  - **Definition of done**: 
    - `masterconnect.go` and `slaveconnect.go` use `netpkg.Dial` directly
    - No more client struct creation or method calls
    - Code is simpler and more readable
    - All unit tests pass: `go test ./pkg/entrypoint/...`
    - All tests pass with race detector: `go test -race ./pkg/entrypoint`
    - Integration tests still pass: `go test ./test/integration/...`

- [X] **Step 6: Update pkg/entrypoint to use new ListenAndServe function**
  - **Task**: Migrate `masterlisten.go` and `slavelisten.go` to use the new `pkg/net.ListenAndServe` function instead of creating server instances. This eliminates the need for Serve(), Close() calls, and error channel handling.
  - **Files**:
    - `pkg/entrypoint/internal.go`:
      - Remove `serverInterface` interface
      - Remove `serverFactory` type  
      - Remove `realServerFactory()` function
      - Keep only handler type definitions
    - `pkg/entrypoint/masterlisten.go`:
      - Remove import of `"dominicbreuker/goncat/pkg/server"`
      - Add import of `netpkg "dominicbreuker/goncat/pkg/net"`
      - Simplify `MasterListen`:
        ```go
        func MasterListen(ctx context.Context, cfg *config.Shared, mCfg *config.Master) error {
            if cfg.Deps == nil {
                cfg.Deps = &config.Dependencies{}
            }
            cfg.Deps.ConnSem = semaphore.New(1, cfg.Timeout)
            
            return netpkg.ListenAndServe(ctx, cfg, makeMasterHandler(ctx, cfg, mCfg, master.Handle))
        }
        ```
      - Remove `masterListen` internal function (no longer needed - complexity moved to net package)
      - Keep `makeMasterHandler` helper as-is
      - Remove error channel handling, server closing logic (now in net package)
    - `pkg/entrypoint/slavelisten.go`:
      - Apply same changes as masterlisten.go
      - Simplify `SlaveListen` to directly call `netpkg.ListenAndServe`
      - Keep `makeSlaveHandler` helper
    - `pkg/entrypoint/masterlisten_test.go`:
      - Update tests to use fake ListenAndServe function
      - Simplify test setup (no more server mocks)
    - `pkg/entrypoint/slavelisten_test.go`:
      - Apply same test updates
  - **Dependencies**: Step 3
  - **Definition of done**: 
    - `masterlisten.go` and `slavelisten.go` use `netpkg.ListenAndServe` directly
    - Code is dramatically simpler (no more goroutines, channels, server management)
    - All unit tests pass: `go test ./pkg/entrypoint/...`
    - All tests pass with race detector: `go test -race ./pkg/entrypoint`
    - Integration tests still pass: `go test ./test/integration/...`

- [X] **Step 7: Run full test suite to verify all functionality**
  - **Task**: Run the complete test suite including unit, integration, and E2E tests to verify that the refactoring hasn't broken any functionality. This is a checkpoint before removing old code.
  - **Commands**:
    ```bash
    # Run all unit tests with race detection
    go test -race ./pkg/...
    
    # Run all integration tests with race detection  
    go test -race ./test/integration/...
    
    # Run all tests including E2E (requires Docker)
    make test
    ```
  - **Files**: None (validation step)
  - **Dependencies**: Steps 5, 6
  - **Definition of done**: 
    - All unit tests pass with race detection
    - All integration tests pass with race detection
    - E2E tests pass (if Docker is available)
    - No regressions detected
    - Build succeeds: `make build-linux`

- [X] **Step 8: Manual verification - basic functionality**
  - **Task**: Manually verify that the tool still works correctly with the new implementation. Test the most common use cases to ensure the refactoring hasn't introduced subtle bugs that automated tests might miss.
  - **Test scenarios** (from `docs/TROUBLESHOOT.md`):
    1. **Basic reverse shell (TCP)**:
       ```bash
       # Terminal 1: Master listen
       ./dist/goncat.elf master listen 'tcp://*:12345' --exec /bin/sh &
       MASTER_PID=$!
       sleep 2
       
       # Terminal 2: Slave connect and test
       echo "echo 'REVERSE_SHELL_TEST' && exit" | ./dist/goncat.elf slave connect tcp://localhost:12345
       
       # Cleanup
       kill $MASTER_PID
       ```
    2. **Basic bind shell (TCP)**:
       ```bash
       # Terminal 1: Slave listen
       ./dist/goncat.elf slave listen 'tcp://*:12346' &
       SLAVE_PID=$!
       sleep 2
       
       # Terminal 2: Master connect
       echo "echo 'BIND_SHELL_TEST' && exit" | ./dist/goncat.elf master connect tcp://localhost:12346 --exec /bin/sh
       
       # Cleanup
       kill $SLAVE_PID
       ```
    3. **TLS encryption**:
       ```bash
       # Terminal 1: Master with TLS
       ./dist/goncat.elf master listen 'tcp://*:12347' --ssl --exec /bin/sh &
       MASTER_PID=$!
       sleep 2
       
       # Terminal 2: Slave with TLS
       echo "echo 'TLS_TEST' && exit" | ./dist/goncat.elf slave connect tcp://localhost:12347 --ssl
       
       # Cleanup
       kill $MASTER_PID
       ```
    4. **Mutual authentication**:
       ```bash
       # Terminal 1: Master with key
       ./dist/goncat.elf master listen 'tcp://*:12348' --ssl --key testpass123 --exec /bin/sh &
       MASTER_PID=$!
       sleep 2
       
       # Terminal 2: Correct password (should work)
       echo "echo 'AUTH_SUCCESS' && exit" | ./dist/goncat.elf slave connect tcp://localhost:12348 --ssl --key testpass123
       
       # Terminal 3: Wrong password (should fail)
       echo "echo 'SHOULD_FAIL'" | ./dist/goncat.elf slave connect tcp://localhost:12348 --ssl --key wrongpass
       
       # Cleanup
       kill $MASTER_PID
       ```
  - **Files**: None (validation step)
  - **Dependencies**: Step 7
  - **Definition of done**: 
    - All four test scenarios execute successfully
    - Basic reverse shell works and executes commands
    - Basic bind shell works and executes commands
    - TLS encryption works without errors
    - Mutual authentication accepts correct password and rejects wrong password
    - No timeout issues observed (connections don't get killed prematurely)
    - **CRITICAL**: If ANY test fails, the implementation must be fixed before proceeding
    - **EMPHASIZE**: Agent is **NOT ALLOWED** to skip this step - if tests don't work, report to user clearly!

- [X] **Step 9: Remove old pkg/client and pkg/server packages**
  - **Task**: Remove the old `pkg/client` and `pkg/server` packages since they've been fully replaced by `pkg/net`. This cleanup step ensures we don't have duplicate code in the repository.
  - **Files**:
    - Delete: `pkg/client/client.go`
    - Delete: `pkg/client/client_test.go`
    - Delete: `pkg/server/server.go`
    - Delete: `pkg/server/server_test.go`
    - Remove directories: `pkg/client/` and `pkg/server/`
  - **Commands**:
    ```bash
    rm -rf pkg/client pkg/server
    go mod tidy  # Clean up any unused imports
    ```
  - **Dependencies**: Steps 7, 8 (must verify everything works before deleting)
  - **Definition of done**: 
    - `pkg/client` directory no longer exists
    - `pkg/server` directory no longer exists
    - Code compiles: `go build ./...`
    - All tests still pass: `go test ./...`
    - No references to old packages remain: `grep -r "pkg/client" . && grep -r "pkg/server" .` (should only find documentation or this plan)

- [X] **Step 10: Update documentation**
  - **Task**: Update all documentation to reflect the new `pkg/net` package and the simplified API. This ensures future developers understand the new architecture.
  - **Files**:
    - `docs/ARCHITECTURE.md`:
      - Update "Server and Client" section (lines 64-74) to describe `pkg/net` instead
      - Replace references to client/server structs with `Dial` and `ListenAndServe` functions
      - Update code flow descriptions to reflect new API
      - Update dependency diagram if it mentions client/server packages
      - Update architectural invariants (lines 266-286) to mention net package
      - Update "Boundaries & Integrations" section to reference net package
    - `.github/copilot-instructions.md`:
      - Update "Project Structure" section to replace `pkg/server` and `pkg/client` with `pkg/net`
      - Update "Key Source Files" section
      - Add note about simplified API (Dial and ListenAndServe functions)
      - Update any code examples that reference old API
    - `README.md` (if it has code examples):
      - Update any code examples that import or use client/server packages
  - **Dependencies**: Step 9
  - **Definition of done**: 
    - `docs/ARCHITECTURE.md` updated with accurate descriptions of `pkg/net`
    - `.github/copilot-instructions.md` reflects new package structure
    - All documentation is consistent with new implementation
    - No mentions of deleted packages remain (except in historical context)
    - Documentation is clear and helpful for future developers

- [X] **Step 11: Final validation and cleanup**
  - **Task**: Perform a final comprehensive validation to ensure everything works correctly, the code is clean, and no issues remain.
  - **Validation steps**:
    1. Clean build from scratch:
       ```bash
       rm -rf dist/
       make build-linux
       ```
    2. Run all linters:
       ```bash
       make lint
       ```
    3. Run full test suite with race detection:
       ```bash
       make test-unit-with-race
       make test-integration-with-race
       ```
    4. Check for any TODO comments or debug code:
       ```bash
       grep -r "TODO\|FIXME\|DEBUG\|XXX" pkg/net
       ```
    5. Verify timeout handling in code review:
       - Manually review `pkg/net/dial_internal.go` and `pkg/net/listen_internal.go`
       - Confirm all `SetDeadline` calls have corresponding `SetDeadline(time.Time{})` calls
       - Confirm deadlines are cleared immediately after successful operations
    6. Run E2E tests (if Docker available):
       ```bash
       make test-e2e
       ```
  - **Files**: None (validation step)
  - **Dependencies**: Step 10
  - **Definition of done**: 
    - Clean build succeeds
    - All linters pass
    - All unit tests pass with race detection
    - All integration tests pass with race detection
    - E2E tests pass (if available)
    - No TODO/FIXME/DEBUG comments in new code
    - Timeout handling verified manually
    - Code is production-ready

## Success Criteria

The refactoring is complete when:

1. **New API is fully functional**:
   - `pkg/net.Dial()` successfully establishes connections for all protocols (TCP, WS, WSS, UDP)
   - `pkg/net.ListenAndServe()` successfully accepts connections for all protocols
   - Both functions handle TLS and mutual authentication correctly

2. **Timeout handling is correct**:
   - Timeouts are set before blocking network operations
   - Timeouts are cleared immediately after operations complete successfully
   - No healthy connections are killed due to lingering timeouts
   - Manual code review confirms proper timeout management

3. **Code is readable and maintainable**:
   - All functions are <50 lines (excluding tests)
   - Function names clearly describe their purpose
   - Code can be understood by reading function names alone
   - No complex or "clever" code remains

4. **All call sites migrated**:
   - `pkg/entrypoint` functions use new API
   - No references to old client/server structs remain in production code
   - Code is significantly simpler at call sites

5. **Tests comprehensive and passing**:
   - All unit tests pass with race detection
   - All integration tests pass with race detection
   - E2E tests pass (if available)
   - Manual verification scenarios pass

6. **Documentation is updated**:
   - Architecture docs reflect new package
   - Copilot instructions are current
   - No outdated references remain

7. **Old code is removed**:
   - `pkg/client` package deleted
   - `pkg/server` package deleted
   - Repository is clean and tidy

## Notes for Implementation

### Timeout Handling Pattern

The most critical aspect of this refactoring is proper timeout management. Follow this pattern consistently:

```go
// Before a potentially blocking network operation:
if timeout > 0 {
    deadline := time.Now().Add(timeout)
    _ = conn.SetDeadline(deadline)
}

// Perform the operation
result, err := blockingOperation()

// After successful operation, clear the deadline immediately
if err == nil && timeout > 0 {
    _ = conn.SetDeadline(time.Time{})
}

return result, err
```

**Why this matters**: If deadlines aren't cleared, they remain active and can kill healthy connections later when unrelated operations run. This is a common source of mysterious timeout errors.

### Code Readability Guidelines

When splitting functions, follow these principles:

1. **Single Responsibility**: Each function should do one thing
2. **Descriptive Names**: Function name should explain what it does (e.g., `establishConnection`, not `doConnect`)
3. **Short Functions**: Target <50 lines per function
4. **Clear Flow**: Main functions should read like a recipe - step 1, step 2, step 3
5. **Minimal Abstraction**: Don't create abstractions that hide complexity - make it explicit

**Example of good refactoring**:

```go
// BEFORE: Long, complex function
func Connect() error {
    // 100 lines of mixed concerns
    // - protocol selection
    // - dialer creation  
    // - connection establishment
    // - TLS upgrade
    // - error handling
}

// AFTER: Clean, readable flow
func Dial(ctx context.Context, cfg *config.Shared) (net.Conn, error) {
    dialer, err := createDialer(ctx, cfg)
    if err != nil {
        return nil, fmt.Errorf("creating dialer: %w", err)
    }
    
    conn, err := establishConnection(ctx, dialer, cfg)
    if err != nil {
        return nil, fmt.Errorf("establishing connection: %w", err)
    }
    
    if cfg.SSL {
        conn, err = upgradeTLS(conn, cfg)
        if err != nil {
            _ = conn.Close()
            return nil, fmt.Errorf("upgrading to TLS: %w", err)
        }
    }
    
    return conn, nil
}
```

### Testing Strategy

- Use the same testing patterns as existing code (dependency injection for unit tests, mocks for integration tests)
- Table-driven tests for different protocols and configurations
- Focus on timeout behavior - verify deadlines are set and cleared
- Test error cases thoroughly (connection refused, TLS failures, timeouts)
- Use race detector on all tests

### Migration Approach

- Implement new package completely before changing call sites
- Migrate entrypoint functions one at a time
- Run tests after each migration step
- Keep manual verification step - it's critical for catching subtle issues
- Don't delete old code until everything is verified working

### Common Pitfalls to Avoid

1. **Forgetting to clear deadlines** - This will cause mysterious timeout errors later
2. **Over-abstracting** - Keep code explicit and clear, don't hide complexity
3. **Skipping manual verification** - Automated tests don't catch everything
4. **Deleting old code too early** - Verify everything works first
5. **Not running race detector** - Concurrency bugs are hard to debug later

## Timeline Estimate

- Steps 1-3: ~2-3 hours (core implementation)
- Step 4: ~1 hour (readability review)
- Steps 5-6: ~1-2 hours (migration)
- Steps 7-8: ~1 hour (validation)
- Steps 9-11: ~1 hour (cleanup and docs)

**Total**: ~6-8 hours for a careful, thorough implementation

This is a significant refactoring but it's worth doing right to improve code quality and maintainability.

## Bug Fix (Post-Implementation)

- [X] **Bug Fix: Nil pointer dereference in TLS upgrade failure**
  - **Issue**: E2E tests revealed a panic when TLS client connects to plain server
    - Error: `panic: runtime error: invalid memory address or nil pointer dereference`
    - Location: `pkg/net/dial.go:56` when trying to close connection after TLS upgrade failure
  - **Root Cause**: In `dial()` function, when `upgradeTLS()` failed, it returned `nil, error`
    - The code did: `conn, err = upgradeTLS(conn, cfg)` 
    - This overwrote the original `conn` with `nil` when TLS failed
    - Then tried to close the nil connection: `_ = conn.Close()`
  - **Fix Applied**: Use separate variable for TLS result
    - Changed to: `tlsConn, err := upgradeTLS(conn, cfg)`
    - Now closes original `conn` on error, not the nil `tlsConn`
    - Only assigns `conn = tlsConn` after successful TLS upgrade
  - **Testing**:
    - Reproduced panic with TLS client → plain server scenario
    - Verified fix: now fails gracefully with `Error: TLS handshake: EOF`
    - All unit tests pass
    - All integration tests pass  
    - Manual verification tests pass (4/4)
    - Full test suite with race detection passes
  - **Definition of done**: 
    - No panic when TLS upgrade fails
    - Graceful error message instead
    - Original connection properly closed
    - All tests passing
