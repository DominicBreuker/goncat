# Integration Tests for cmd/masterconnect

This document describes the integration tests for the `cmd/masterconnect` package.

## Overview

The integration tests in `integration_test.go` validate the full interaction between master and slave handlers, simulating the behavior of the goncat application when a master connects to a listening slave.

## Test Suite

### Test Functions

1. **TestMasterConnectBasic**
   - Tests basic connectivity between master and slave
   - Validates that handlers can be created and communicate
   - Ensures proper cleanup on context cancellation

2. **TestMasterConnectExec** (2 subtests)
   - Tests command execution over master-slave connection
   - Subtests:
     - `echo_command`: Executes `sh` with echo command
     - `simple_command`: Executes `sh` with printf command
   - Validates the `--exec` flag functionality

3. **TestMasterConnectMultiplexing**
   - Tests that multiple port forwards can be set up concurrently
   - Validates the core multiplexing functionality
   - Creates two separate local port forwards over one connection

4. **TestMasterConnectConfiguration** (2 subtests)
   - Tests various configuration options
   - Subtests:
     - `tcp_without_ssl`: Basic TCP connection
     - `tcp_with_verbose`: TCP with verbose logging
   - Ensures different configurations work correctly

5. **TestMasterConnectErrorHandling** (2 subtests)
   - Tests error conditions and cleanup behavior
   - Subtests:
     - `context_cancellation`: Validates cleanup after context cancel
     - `connection_close`: Validates cleanup after connection close
   - Ensures resources are properly released

6. **TestMasterConnectSessionLifecycle** (2 subtests)
   - Tests complete session lifecycle with different timeouts
   - Subtests:
     - `short_session`: 2-second session
     - `longer_session`: 5-second session
   - Validates session management over time

## Running the Tests

### Run all tests
```bash
go test ./cmd/masterconnect/...
```

### Run with coverage
```bash
go test -cover ./cmd/masterconnect/...
```

### Run with verbose output
```bash
go test -v ./cmd/masterconnect/...
```

### Skip integration tests (run only unit tests)
```bash
go test -short ./cmd/masterconnect/...
```

## Test Design Principles

All tests follow the guidelines from `TESTING.md`:

- **Table-driven**: Using subtests for better organization and debugging
- **Parallel execution**: Tests use `t.Parallel()` where safe
- **Proper cleanup**: Using `defer`, wait groups, and contexts
- **Short mode skip**: Integration tests skip when `-short` flag is set
- **Reasonable timeouts**: Prevent hanging tests
- **Error handling**: Test both happy paths and error conditions

## Architecture

The tests use `net.Pipe()` to simulate network connections between master and slave:

```
Master Handler <---> net.Pipe() <---> Slave Handler
```

This allows testing the full protocol without requiring actual network setup.

## Test Coverage

Current coverage: **16.2%** of statements in `cmd/masterconnect`

This covers:
- Command creation and flag setup
- Handler initialization and basic communication
- Error handling paths
- Configuration validation

## What's Not Covered

The following areas are not covered due to complexity:

1. **PTY Mode Testing**: Requires terminal/TTY simulation which is complex in automated tests
2. **Actual Data Transfer**: Complete data flow through port forwards is complex due to the foreground job management
3. **SOCKS Proxy**: Requires SOCKS client simulation

These features are covered by the expect-based integration tests in `tests/master-connect/`.

## Future Improvements

Potential areas for expansion:

1. Add helper functions to reduce test boilerplate
2. Create test fixtures for common configurations
3. Add benchmarks for performance testing
4. Expand port forwarding tests with actual data transfer (requires refactoring to handle stdin/stdout properly)
5. Add tests for different transport protocols (WebSocket, WSS)
6. Add tests with SSL/TLS and mutual authentication

## Comparison with Expect Tests

The Go integration tests complement the expect-based tests in `tests/master-connect/`:

**Go Integration Tests** (this file):
- Test internal handler behavior
- Fast execution (~0.4s)
- Easy to debug
- Good for unit-level integration testing

**Expect Tests** (`tests/master-connect/`):
- Test complete binary behavior
- Test user-facing functionality
- Test actual terminal interaction
- Better for end-to-end validation

Both test suites are valuable and serve different purposes.
