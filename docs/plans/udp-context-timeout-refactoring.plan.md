# Plan for UDP Context and Timeout Handling Refactoring

Refactor the `pkg/transport/udp` package to use caller-provided contexts and respect user-defined timeouts throughout, eliminating hardcoded context creation and timeout values.

## Overview

The UDP transport package currently creates its own contexts using `context.Background()` in some places and uses hardcoded timeouts (e.g., `5*time.Second`) for certain operations. This prevents proper cancellation propagation and ignores user-specified timeout values from the `--timeout` flag. This refactoring will ensure:

1. All functions in the UDP transport use the context passed in from callers
2. All timeout values respect the user-defined `--timeout` flag
3. Proper cancellation propagation throughout the UDP transport layer

## Implementation Plan

- [ ] Step 1: Audit UDP Transport for Context Issues
  - **Task**: Identify all places in `pkg/transport/udp` where `context.Background()` is used or where new contexts are created instead of using the caller's context
  - **Files**:
    - `pkg/transport/udp/dialer.go`: Review all function calls
    - `pkg/transport/udp/listener.go`: Review all function calls, especially `acceptQUICLoop` and `acceptStreamAndActivate`
    - `pkg/transport/udp/streamconn.go`: Review for context usage
  - **Expected findings**:
    - `acceptQUICLoop` (line 171): Uses `context.Background()` for `quicListener.Accept()`
    - `acceptStreamAndActivate` (line 222): Creates new context with hardcoded 5-second timeout
  - **Dependencies**: None
  - **Definition of done**: Complete list of all context and timeout issues documented, with line numbers and exact code locations

- [ ] Step 2: Audit UDP Transport for Hardcoded Timeouts
  - **Task**: Identify all places where hardcoded timeout values are used instead of the user-provided timeout parameter
  - **Files**:
    - `pkg/transport/udp/dialer.go`: Check all timeout usages
    - `pkg/transport/udp/listener.go`: Check all timeout usages, especially in `acceptStreamAndActivate`
  - **Expected findings**:
    - `acceptStreamAndActivate` (line 222): Hardcoded `5*time.Second` timeout for AcceptStream
  - **Dependencies**: Step 1
  - **Definition of done**: Complete list of all hardcoded timeout values with alternatives based on user-provided timeout

- [ ] Step 3: Design Context Propagation Strategy
  - **Task**: Design how to propagate the caller's context through the UDP transport functions
  - **Design considerations**:
    - The `ctx` parameter is already available in `Dial()` and `ListenAndServe()`
    - Need to pass context through to helper functions that currently use `context.Background()`
    - Consider using context for cancellation in long-running accept loops
    - Ensure backward compatibility with existing timeout behavior (user timeout should be used)
  - **Pseudocode**:
    ```go
    // acceptQUICLoop should accept context parameter
    func acceptQUICLoop(ctx context.Context, quicListener *quic.Listener, ...) error {
        for {
            // Use context instead of context.Background()
            conn, err := quicListener.Accept(ctx)
            if err != nil {
                // Handle context cancellation
                if ctx.Err() != nil {
                    return nil // graceful shutdown
                }
                return err
            }
            // ... rest of logic
        }
    }
    
    // acceptStreamAndActivate should use timeout from parameters
    func acceptStreamAndActivate(ctx context.Context, conn *quic.Conn, timeout time.Duration) (*quic.Stream, error) {
        // Use provided timeout instead of hardcoded 5 seconds
        acceptCtx, cancel := context.WithTimeout(ctx, timeout)
        defer cancel()
        
        stream, err := conn.AcceptStream(acceptCtx)
        // ... rest of logic
    }
    ```
  - **Dependencies**: Steps 1-2
  - **Definition of done**: Clear design documented for how context and timeout will flow through all UDP transport functions

- [ ] Step 4: Refactor `acceptQUICLoop` to Use Caller Context
  - **Task**: Modify `acceptQUICLoop` to accept and use the caller's context instead of `context.Background()`
  - **Files**:
    - `pkg/transport/udp/listener.go`: Modify `acceptQUICLoop` function signature and implementation
  - **Changes**:
    - Change signature: `func acceptQUICLoop(ctx context.Context, quicListener *quic.Listener, handler transport.Handler, sem chan struct{}) error`
    - Replace `context.Background()` with `ctx` in `quicListener.Accept(ctx)` call
    - Add context cancellation check in error handling
  - **Pseudocode**:
    ```go
    func acceptQUICLoop(ctx context.Context, quicListener *quic.Listener, handler transport.Handler, sem chan struct{}) error {
        for {
            conn, err := quicListener.Accept(ctx)  // Use ctx instead of context.Background()
            if err != nil {
                // Check if context was cancelled
                if ctx.Err() != nil {
                    return nil  // Graceful shutdown
                }
                // Check for clean shutdown
                if isClosedError(err) {
                    return nil
                }
                return err
            }
            // ... rest of logic unchanged
        }
    }
    ```
  - **Dependencies**: Step 3
  - **Definition of done**: 
    - `acceptQUICLoop` accepts context parameter
    - Uses caller's context for Accept operation
    - Properly handles context cancellation
    - All call sites updated to pass context

- [ ] Step 5: Refactor `acceptStreamAndActivate` to Use User Timeout
  - **Task**: Modify `acceptStreamAndActivate` to accept a timeout parameter and use it instead of hardcoded `5*time.Second`
  - **Files**:
    - `pkg/transport/udp/listener.go`: Modify `acceptStreamAndActivate` function signature and implementation
  - **Changes**:
    - Change signature: `func acceptStreamAndActivate(ctx context.Context, conn *quic.Conn, timeout time.Duration) (*quic.Stream, error)`
    - Replace hardcoded `5*time.Second` with `timeout` parameter
    - Use parent context from caller instead of creating new `context.Background()`
  - **Pseudocode**:
    ```go
    func acceptStreamAndActivate(ctx context.Context, conn *quic.Conn, timeout time.Duration) (*quic.Stream, error) {
        // Use provided timeout instead of hardcoded 5 seconds
        acceptCtx, cancel := context.WithTimeout(ctx, timeout)
        defer cancel()
        
        stream, err := conn.AcceptStream(acceptCtx)
        if err != nil {
            return nil, fmt.Errorf("no stream")
        }
        
        // Read and discard init byte (keep existing logic)
        initByte := make([]byte, 1)
        _, err = stream.Read(initByte)
        if err != nil {
            return nil, fmt.Errorf("failed to read init byte")
        }
        
        return stream, nil
    }
    ```
  - **Dependencies**: Step 4
  - **Definition of done**:
    - `acceptStreamAndActivate` accepts context and timeout parameters
    - Uses caller's context as parent for timeout context
    - Uses user-provided timeout instead of hardcoded value
    - All call sites updated to pass both context and timeout

- [ ] Step 6: Update `handleQUICConnection` to Pass Context and Timeout
  - **Task**: Ensure `handleQUICConnection` has access to context and timeout to pass to `acceptStreamAndActivate`
  - **Files**:
    - `pkg/transport/udp/listener.go`: Modify `handleQUICConnection` function signature
  - **Changes**:
    - Add context and timeout parameters to function signature
    - Pass these to `acceptStreamAndActivate` call
    - Update call site in `acceptQUICLoop` to pass context and timeout
  - **Pseudocode**:
    ```go
    func handleQUICConnection(ctx context.Context, conn *quic.Conn, handler transport.Handler, sem chan struct{}, timeout time.Duration) {
        defer func() {
            sem <- struct{}{} // Release slot
        }()
        defer func() {
            if r := recover(); r != nil {
                log.ErrorMsg("Handler panic: %v\n", r)
            }
        }()
        
        // Accept stream and read init byte - pass context and timeout
        stream, err := acceptStreamAndActivate(ctx, conn, timeout)
        if err != nil {
            _ = conn.CloseWithError(0, err.Error())
            return
        }
        
        // ... rest of logic unchanged
    }
    ```
  - **Dependencies**: Step 5
  - **Definition of done**:
    - `handleQUICConnection` accepts context and timeout parameters
    - Passes both to `acceptStreamAndActivate`
    - Call site in `acceptQUICLoop` updated to spawn goroutine with context and timeout

- [ ] Step 7: Update `serveQUICConnections` to Pass Context to Accept Loop
  - **Task**: Ensure the accept loop goroutine receives the context for proper cancellation
  - **Files**:
    - `pkg/transport/udp/listener.go`: Modify `serveQUICConnections` function
  - **Changes**:
    - Pass `ctx` to `acceptQUICLoop` goroutine
    - Ensure timeout is available to pass through the call chain
  - **Pseudocode**:
    ```go
    func serveQUICConnections(ctx context.Context, quicListener *quic.Listener, handler transport.Handler, sem chan struct{}, timeout time.Duration) error {
        errCh := make(chan error, 1)
        go func() {
            errCh <- acceptQUICLoop(ctx, quicListener, handler, sem, timeout)  // Pass ctx and timeout
        }()
        
        // ... rest of logic unchanged
    }
    ```
  - **Dependencies**: Step 6
  - **Definition of done**:
    - Context flows from `serveQUICConnections` to `acceptQUICLoop`
    - Timeout parameter added to function signature and passed through
    - All call sites updated

- [ ] Step 8: Update `ListenAndServe` to Pass Timeout Through Chain
  - **Task**: Ensure `ListenAndServe` passes the timeout parameter through the entire call chain
  - **Files**:
    - `pkg/transport/udp/listener.go`: Review and update `ListenAndServe` function
  - **Changes**:
    - Verify timeout is already available as parameter
    - Update call to `serveQUICConnections` to pass timeout parameter
  - **Pseudocode**:
    ```go
    func ListenAndServe(ctx context.Context, addr string, timeout time.Duration, handler transport.Handler) error {
        // ... existing setup code ...
        
        // Serve connections with semaphore - pass timeout through
        sem := createConnectionSemaphore(100)
        return serveQUICConnections(ctx, quicListener, handler, sem, timeout)
    }
    ```
  - **Dependencies**: Step 7
  - **Definition of done**:
    - `ListenAndServe` passes timeout to `serveQUICConnections`
    - Complete call chain from `ListenAndServe` to `acceptStreamAndActivate` uses user-provided timeout

- [ ] Step 9: Review Dialer for Context Issues
  - **Task**: Verify that `Dial()` and its helper functions properly use the caller's context
  - **Files**:
    - `pkg/transport/udp/dialer.go`: Review all functions
  - **Review points**:
    - Verify `dialQUIC` uses the `ctx` parameter (line 118: already correct)
    - Check if any other functions create new contexts
    - Ensure no hardcoded timeouts in dialer path
  - **Expected outcome**: Dialer should already be using context correctly, but verify
  - **Dependencies**: Step 8
  - **Definition of done**: 
    - All dialer functions reviewed
    - Confirm context is properly propagated
    - Document any issues found (expected: none)

- [ ] Step 10: Run Linters and Build
  - **Task**: Ensure code compiles and passes all linting checks
  - **Commands**:
    ```bash
    cd /home/runner/work/goncat/goncat
    make lint
    make build-linux
    ```
  - **Expected outcome**: No linting errors, successful build
  - **Dependencies**: Steps 4-9
  - **Definition of done**: 
    - `make lint` passes with no errors
    - `make build-linux` completes successfully
    - Binary created at `dist/goncat.elf`

- [ ] Step 11: Run Existing Unit and Integration Tests
  - **Task**: Ensure all existing tests still pass after refactoring
  - **Commands**:
    ```bash
    cd /home/runner/work/goncat/goncat
    make test-unit
    make test-integration
    ```
  - **Expected outcome**: All tests pass
  - **Dependencies**: Step 10
  - **Definition of done**:
    - All unit tests pass
    - All integration tests pass
    - No race conditions detected with `-race` flag

- [ ] Step 12: Manual Verification - Basic UDP Connection
  - **Task**: **REQUIRED - DO NOT SKIP!** Manually verify that UDP transport still works correctly with the refactored code
  - **Reference**: See `docs/TROUBLESHOOT.md` for detailed manual verification instructions
  - **Setup**:
    ```bash
    cd /home/runner/work/goncat/goncat
    make build-linux
    ```
  - **Test procedure**:
    ```bash
    # Terminal 1: Start master listener on UDP
    ./dist/goncat.elf master listen 'udp://*:12345' --exec /bin/sh &
    MASTER_PID=$!
    
    # Wait for master to start
    sleep 3
    
    # Verify master is listening (should see UDP socket)
    ss -unp 2>/dev/null | grep 12345
    
    # Terminal 2: Connect slave and test
    echo "echo 'UDP_CONTEXT_TEST' && exit" | timeout 10 ./dist/goncat.elf slave connect udp://localhost:12345
    
    # Cleanup
    kill $MASTER_PID 2>/dev/null || true
    pkill -9 goncat.elf 2>/dev/null || true
    ```
  - **Expected behavior**:
    - Master successfully binds to UDP port 12345
    - Slave connects successfully
    - Command executes and returns output "UDP_CONTEXT_TEST"
    - Connection closes cleanly
    - No timeout errors
  - **Definition of done**: 
    - Master listens on UDP port
    - Slave connects successfully
    - Shell command executes and returns expected output
    - No errors related to context cancellation or timeouts
    - **IF THIS DOES NOT WORK, REPORT TO USER IMMEDIATELY!**
  - **Dependencies**: Step 11

- [ ] Step 13: Manual Verification - Context Cancellation
  - **Task**: **REQUIRED - DO NOT SKIP!** Verify that context cancellation properly shuts down UDP listeners
  - **Test procedure**:
    ```bash
    # Start master listener on UDP
    ./dist/goncat.elf master listen 'udp://*:12346' --exec /bin/sh &
    MASTER_PID=$!
    
    # Wait for master to start
    sleep 2
    
    # Verify listening
    ss -unp 2>/dev/null | grep 12346 && echo "Listening: PASS" || echo "Listening: FAIL"
    
    # Kill the master process (simulates context cancellation)
    kill $MASTER_PID
    
    # Wait for cleanup
    sleep 2
    
    # Verify port is released (should not show up)
    ss -unp 2>/dev/null | grep 12346 && echo "Cleanup: FAIL (port still bound)" || echo "Cleanup: PASS (port released)"
    
    # Cleanup any stragglers
    pkill -9 goncat.elf 2>/dev/null || true
    ```
  - **Expected behavior**:
    - Master binds to UDP port
    - After killing master, port is released promptly (within 2 seconds)
    - No zombie processes or leaked connections
  - **Definition of done**:
    - Port is released after master terminates
    - No "address already in use" errors when restarting
    - Clean shutdown with no leaked resources
    - **IF THIS DOES NOT WORK, REPORT TO USER IMMEDIATELY!**
  - **Dependencies**: Step 12

- [ ] Step 14: Manual Verification - Custom Timeout
  - **Task**: **REQUIRED - DO NOT SKIP!** Verify that custom timeout values are respected
  - **Test procedure**:
    ```bash
    # Test with very short timeout (should work for localhost)
    ./dist/goncat.elf master listen 'udp://*:12347' --timeout 2000 --exec /bin/sh &
    MASTER_PID=$!
    
    sleep 2
    
    # This should succeed quickly
    echo "exit" | timeout 5 ./dist/goncat.elf slave connect udp://localhost:12347 --timeout 2000
    EXIT_CODE=$?
    
    kill $MASTER_PID 2>/dev/null || true
    
    [ $EXIT_CODE -eq 0 ] && echo "Custom timeout: PASS" || echo "Custom timeout: FAIL"
    
    # Cleanup
    pkill -9 goncat.elf 2>/dev/null || true
    ```
  - **Expected behavior**:
    - Connection succeeds with 2-second timeout (plenty for localhost)
    - No hardcoded 5-second delays observable
    - Fast connection establishment
  - **Definition of done**:
    - Connection with custom timeout succeeds
    - No timeout-related errors
    - Connection establishment respects user timeout
    - **IF THIS DOES NOT WORK, REPORT TO USER IMMEDIATELY!**
  - **Dependencies**: Step 13

- [ ] Step 15: Code Review and Documentation
  - **Task**: Review all changes and update any relevant documentation
  - **Files to review**:
    - All modified files in `pkg/transport/udp/`
    - Check if any comments need updates
    - Verify function signatures are documented
  - **Documentation updates**:
    - Ensure godoc comments mention context and timeout behavior
    - Update any inline comments that reference old behavior
  - **Dependencies**: Steps 12-14
  - **Definition of done**:
    - All modified functions have proper godoc comments
    - No references to old behavior (context.Background, hardcoded timeouts)
    - Code is clean and well-documented

- [ ] Step 16: Final Testing and Commit
  - **Task**: Run complete test suite and commit changes
  - **Commands**:
    ```bash
    # Clean build
    rm -rf dist/
    make build-linux
    
    # Run all tests with race detection
    go test -race ./pkg/transport/udp/...
    make test-unit-with-race
    make test-integration-with-race
    
    # Verify no issues
    make lint
    ```
  - **Expected outcome**: All tests pass, no race conditions, clean build
  - **Dependencies**: Step 15
  - **Definition of done**:
    - All tests pass with race detection enabled
    - Clean build with no warnings
    - Changes ready for commit

## Summary

This refactoring addresses two key issues in the UDP transport:

1. **Context propagation**: Eliminates use of `context.Background()` and ensures the caller's context is used throughout, enabling proper cancellation
2. **Timeout handling**: Removes hardcoded timeouts (like `5*time.Second`) and ensures all operations respect the user-provided `--timeout` flag

The changes are surgical and focused on the UDP transport package only. The refactoring maintains backward compatibility while improving shutdown behavior and timeout control.

**Key risks mitigated:**
- Comprehensive manual verification ensures functionality remains intact
- Existing test suite validates no regressions
- Incremental changes make it easy to identify any issues

**Critical success criteria:**
- All manual verification steps must pass
- If any manual test fails, immediately report to user before proceeding
- Zero tolerance for broken functionality
