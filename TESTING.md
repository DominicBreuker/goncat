# Testing Guidelines for goncat

This document provides comprehensive testing guidelines for the goncat project. We follow idiomatic Go testing practices to ensure code quality, maintainability, and reliability.

## Overview

Our testing strategy includes three distinct types of tests:

### Unit Tests
- **Location**: `*_test.go` files next to source code
- **Purpose**: Test individual functions and components in isolation
- **Mocking strategy**: Use internal dependency injection with fake implementations
- **No external mocks**: Unit tests do NOT use mocks from `mocks/` package
- **Pattern**: Create internal unexported functions that accept injected dependencies (e.g., `masterListen()` accepting `serverFactory`), while exported functions (e.g., `MasterListen()`) delegate to internal functions with real dependencies
- **Example**: `pkg/entrypoint/*_test.go` tests use fake servers/clients injected via function parameters

### Integration Tests
- **Location**: `test/integration/` directory
- **Purpose**: Test complete tool workflows using entrypoint functions
- **Mocking strategy**: Use mocks from `mocks/` package (MockTCPNetwork, MockStdio, MockExec)
- **Dependencies**: Passed via `config.Dependencies` struct to avoid real system resources
- **Scope**: Test master-slave communication flows without real network/terminal I/O
- **Speed**: Fast (~2 seconds) and deterministic
- **Example**: `test/integration/plain/` tests complete bidirectional data flow

### End-to-End (E2E) Tests
- **Location**: `test/e2e/` directory
- **Purpose**: Validate real compiled binaries in realistic environments
- **Mocking strategy**: No mocks - uses actual compiled binaries
- **Environment**: Docker containers with Alpine Linux and expect scripts
- **Scope**: Tests all transport protocols (tcp, ws, wss) in bind/reverse shell scenarios
- **Speed**: Slower (~2-3 minutes) but validates real-world behavior
- **Requirements**: Docker and Docker Compose

### Coverage & Organization
- **Coverage goals**: Aim for meaningful coverage of edge cases and error paths, not just 100% line coverage
- **Unit tests**: Fast, isolated, test behavior and error handling
- **Integration tests**: Validate complete flows remain working
- **E2E tests**: Ensure real binaries work in production-like scenarios

## Structure & Naming

### Test Files
- Place tests in `*_test.go` files next to the code they test
- Use the same package name for white-box testing (access to private functions)
- Use `packagename_test` for black-box testing (only public API)

### Test Functions
- **Unit tests**: `func TestXxx(t *testing.T)`
- **Benchmarks**: `func BenchmarkXxx(b *testing.B)`
- **Fuzz tests**: `func FuzzXxx(f *testing.F)`
- Name tests after behavior: `TestParseTransport_EmptyInput`, not `Test1`

## Table-Driven Tests

Prefer table-driven tests with subtests for better organization and debugging:

```go
func TestSum(t *testing.T) {
    t.Parallel()
    cases := []struct{
        name string
        in   []int
        want int
    }{
        {"empty", nil, 0},
        {"single", []int{2}, 2},
        {"many", []int{1,2,3}, 6},
    }
    for _, tc := range cases {
        tc := tc // capture range variable
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            if got := Sum(tc.in...); got != tc.want {
                t.Fatalf("Sum(%v) = %d; want %d", tc.in, got, tc.want)
            }
        })
    }
}
```

## Assertions & Error Handling

- Use plain `if got != want { t.Fatalf(...) }` - avoid external assertion libraries
- Use `t.Fatalf()` when correctness is compromised and further tests would be meaningless
- Use `t.Errorf()` to log errors but continue testing other cases
- For functions returning `(T, error)`, always check the error first:

```go
got, err := ParseTransport(input)
if err != nil {
    if !tc.wantErr {
        t.Fatalf("unexpected error: %v", err)
    }
    return
}
if tc.wantErr {
    t.Fatalf("expected error but got none")
}
```

## Test Helpers & Cleanup

### Helpers
Extract repetitive checks into helper functions and mark them with `t.Helper()`:

```go
func readFile(t *testing.T, path string) []byte {
    t.Helper()
    b, err := os.ReadFile(path)
    if err != nil {
        t.Fatalf("read: %v", err)
    }
    return b
}
```

### Cleanup
- Use `t.Cleanup()` for deterministic teardown
- Use `testing.TempDir()` for temporary filesystem operations
- Cleanup runs in LIFO order (last registered runs first)

```go
func TestWithTempFile(t *testing.T) {
    dir := t.TempDir()
    file := filepath.Join(dir, "test.txt")
    
    f, err := os.Create(file)
    if err != nil {
        t.Fatal(err)
    }
    t.Cleanup(func() {
        f.Close()
    })
    
    // test using file
}
```

## Concurrency & Timing

- Make tests fast and deterministic
- Avoid `time.Sleep()` - use channels or `context.WithTimeout()` instead
- Use `t.Parallel()` where safe to speed up test suite
- Guard shared state with mutexes or avoid it entirely
- **CRITICAL**: All tests must be race-free
  - Always run tests with race detector: `go test -race ./...`
  - CI runs with race detection enabled - tests will fail if races are detected
  - Protect concurrent access to shared state with `sync.Mutex` or other synchronization primitives
  - Example: If a fake struct has a `closed` field accessed by multiple goroutines, protect it:
    ```go
    type fakeServer struct {
        closed bool
        mu     sync.Mutex
    }
    
    func (f *fakeServer) Close() error {
        f.mu.Lock()
        defer f.mu.Unlock()
        if !f.closed {
            f.closed = true
        }
        return nil
    }
    ```

## Golden Files

For tests with large expected outputs:
- Store expected outputs in `testdata/` directory
- Load with `os.ReadFile(filepath.Join("testdata", ...))`
- Optionally allow updates via environment flag (never default):

```go
var update = flag.Bool("update", false, "update golden files")

func TestFormat(t *testing.T) {
    got := Format(input)
    golden := filepath.Join("testdata", "expected.txt")
    
    if *update {
        os.WriteFile(golden, []byte(got), 0644)
        return
    }
    
    want, err := os.ReadFile(golden)
    if err != nil {
        t.Fatal(err)
    }
    
    if got != string(want) {
        t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
    }
}
```

## Unit Test Dependency Injection

Unit tests use **internal dependency injection** to avoid real system dependencies:

### Pattern: Internal Function with Injected Dependencies

1. Create an internal (unexported) function that accepts fake implementations via function parameters
2. The exported function delegates to the internal function, passing real implementations
3. Tests call the internal function with fakes
4. When there are many dependencies, group them into a struct for better maintainability

Example from `pkg/entrypoint/masterlisten.go`:

```go
// Exported function - uses real dependencies
func MasterListen(ctx context.Context, cfg *config.Shared, mCfg *config.Master) error {
    return masterListen(ctx, cfg, mCfg, func(...) (serverInterface, error) {
        return server.New(ctx, cfg, handle)  // Real server
    }, makeMasterHandler)
}

// Internal function - accepts injected dependencies for testing
func masterListen(
    ctx context.Context,
    cfg *config.Shared,
    mCfg *config.Master,
    newServer serverFactory,  // Injected dependency
    makeHandler func(...) func(net.Conn) error,
) error {
    s, err := newServer(ctx, cfg, makeHandler(ctx, cfg, mCfg))
    // ... rest of implementation
}
```

Example with grouped dependencies from `pkg/client/client.go`:

```go
// dependencies holds the injectable dependencies for testing.
type dependencies struct {
    newTCPDialer func(string, *config.Dependencies) (transport.Dialer, error)
    newWSDialer  func(context.Context, string, config.Protocol) transport.Dialer
    tlsUpgrader  func(net.Conn, string, time.Duration) (net.Conn, error)
}

// Connect establishes a connection (exported function - uses real dependencies)
func (c *Client) Connect() error {
    deps := &dependencies{
        newTCPDialer: func(addr string, deps *config.Dependencies) (transport.Dialer, error) {
            return tcp.NewDialer(addr, deps)
        },
        newWSDialer: func(ctx context.Context, addr string, proto config.Protocol) transport.Dialer {
            return ws.NewDialer(ctx, addr, proto)
        },
        tlsUpgrader: upgradeToTLS,
    }
    return c.connect(deps)
}

// connect is the internal implementation that accepts injected dependencies for testing
func (c *Client) connect(deps *dependencies) error {
    // ... implementation using deps.newTCPDialer, deps.newWSDialer, deps.tlsUpgrader
}
```

### Creating Fakes for Unit Tests

Keep fakes minimal and in test files:

```go
// In *_test.go file
type fakeServer struct {
    serveErr error
    closed   bool
}

func (f *fakeServer) Serve() error {
    return f.serveErr
}

func (f *fakeServer) Close() error {
    f.closed = true
    return nil
}

func TestMasterListen_Success(t *testing.T) {
    fs := &fakeServer{}
    newServer := func(ctx, cfg, handle) (serverInterface, error) {
        return fs, nil
    }
    
    err := masterListen(ctx, cfg, mCfg, newServer, makeHandler)
    // ... assertions
}
```

**Important**: Unit tests do NOT use mocks from the `mocks/` package.

## Integration Test Mocks

Integration tests use the `mocks/` package to avoid real system resources while testing complete workflows:

### Mock Infrastructure

The `mocks/` package provides:
- **MockTCPNetwork**: In-memory TCP network using `net.Pipe()`
- **MockStdio**: Simulated stdin/stdout with buffers
- **MockExec**: Simulated command execution

### Using Mocks in Integration Tests

Integration tests pass mocks via `config.Dependencies`:

```go
func TestMasterSlaveFlow(t *testing.T) {
    mockNet := mocks.NewMockTCPNetwork()
    masterStdio := mocks.NewMockStdio()
    
    cfg := &config.Shared{
        Protocol: config.ProtoTCP,
        Deps: &config.Dependencies{
            TCPDialer:   mockNet.DialTCP,
            TCPListener: mockNet.ListenTCP,
            Stdin:  func() io.Reader { return masterStdio.GetStdin() },
            Stdout: func() io.Writer { return masterStdio.GetStdout() },
        },
    }
    
    // Use entrypoint functions with mocked dependencies
    go entrypoint.MasterListen(ctx, cfg, masterCfg)
    // ... test complete workflow
}
```

### Benefits of Integration Test Mocks

- **Fast**: No network latency or process spawning (~2 seconds for full suite)
- **Reliable**: No port conflicts or race conditions
- **Isolated**: Tests don't affect or depend on system state
- **CI-friendly**: No special privileges or Docker required
- **Deterministic**: Predictable behavior without timing issues

See `test/integration/README.md` for more details on integration testing.

## Benchmarks

- Reset timer after setup: `b.ResetTimer()`
- Avoid allocations inside loops when measuring performance
- Store results in package-level variable to prevent optimization elimination

```go
var result int

func BenchmarkSum(b *testing.B) {
    data := make([]int, 1000)
    for i := range data {
        data[i] = i
    }
    b.ResetTimer()
    
    var r int
    for i := 0; i < b.N; i++ {
        r = Sum(data...)
    }
    result = r
}
```

## Fuzz Testing

Use fuzzing for important parsers and decoders:

```go
func FuzzParseTransport(f *testing.F) {
    // Add seed inputs
    f.Add("tcp://localhost:8080")
    f.Add("ws://example.com:443")
    
    f.Fuzz(func(t *testing.T, input string) {
        _, _, _, err := ParseTransport(input)
        // Validate invariants - should never panic
        // May return error for invalid input
        _ = err
    })
}
```

## Coverage & Organization

### Running Tests
```bash
go test ./...                    # All tests
go test -v ./...                 # Verbose output
go test -cover ./...             # With coverage
go test -race ./...              # With race detector
go test -run TestSpecific ./...  # Run specific test
```

### Coverage Goals
- Aim for meaningful coverage, not 100%
- Cover edge cases and error paths
- Don't test trivial getters/setters
- Focus on business logic and complex code paths

### Test Organization
- Keep tests independent - no global state leaks
- Each test should set up its own fixtures
- Tests should be runnable in any order
- Prefer mocks over real resources to keep tests fast and reliable
- Use `-short` flag only for truly slow tests (e.g., those that must use real resources)

## External Dependencies

- No network/filesystem/external dependencies by default
- If unavoidable, gate with `-short` flag:

```go
func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    // test with external dependencies
}
```

## Common Pitfalls

### Loop Variable Capture
```go
// WRONG
for _, tc := range cases {
    t.Run(tc.name, func(t *testing.T) {
        t.Parallel()
        // tc may have wrong value
    })
}

// CORRECT
for _, tc := range cases {
    tc := tc // capture range variable
    t.Run(tc.name, func(t *testing.T) {
        t.Parallel()
        // tc has correct value
    })
}
```

### Time-Sensitive Tests
```go
// WRONG - flaky
time.Sleep(100 * time.Millisecond)

// BETTER - use timeout
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()
```

### Brittle Assertions
```go
// WRONG - brittle to message changes
if err.Error() != "connection failed" {
    t.Fatal("wrong error")
}

// BETTER - check error presence
if err == nil {
    t.Fatal("expected error")
}
```

## CI/CD Integration

Our CI pipeline runs:
```bash
make lint        # Format, vet, staticcheck
make test-unit   # Unit tests with coverage
make test        # All tests including integration
```

**Race Detection in CI:**
- All tests run with `-race` flag enabled in CI
- Any race condition will cause the build to fail
- Always test locally with `go test -race ./...` before pushing
- See "Concurrency & Timing" section for guidelines on writing race-free tests

## Documentation

Tests serve as documentation:
- Use descriptive test names
- Include comments for complex test logic
- Document assumptions and invariants
- Test names should read like specifications

## Examples

See existing tests for examples:
- `cmd/shared/parsers_test.go` - table-driven tests with error cases
- `pkg/entrypoint/*_test.go` - dependency injection with function parameters
- `pkg/client/client_test.go` - dependency injection with grouped dependencies struct

## Detailed Testing Strategies

### Unit Tests (in package directories)

**Purpose**: Test individual functions in isolation  
**Location**: `*_test.go` files next to source code  
**Mocking**: Internal dependency injection with fakes defined in test files  
**Pattern**: Create internal functions accepting injected dependencies

Example from `pkg/entrypoint/`:
- Exported `MasterListen()` calls internal `masterListen()` with real dependencies
- Tests call `masterListen()` with fake server factories
- Fakes are simple structs in test files implementing required interfaces

### Integration Tests (in test/integration/)

**Purpose**: Validate complete tool workflows  
**Location**: `test/integration/` directory  
**Mocking**: Use `mocks/` package (MockTCPNetwork, MockStdio, MockExec)  
**Pattern**: Pass mocks via `config.Dependencies` to entrypoint functions

Example structure:
```go
func TestMasterSlaveFlow(t *testing.T) {
    mockNet := mocks.NewMockTCPNetwork()
    masterStdio := mocks.NewMockStdio()
    slaveStdio := mocks.NewMockStdio()
    
    masterCfg := &config.Shared{
        Protocol: config.ProtoTCP,
        Deps: &config.Dependencies{
            TCPDialer:   mockNet.DialTCP,
            TCPListener: mockNet.ListenTCP,
            Stdin:  func() io.Reader { return masterStdio.GetStdin() },
            Stdout: func() io.Writer { return masterStdio.GetStdout() },
        },
    }
    
    // Start master and slave using entrypoint functions
    go entrypoint.MasterListen(ctx, masterCfg, masterModeCfg)
    go entrypoint.SlaveConnect(ctx, slaveCfg)
    
    // Test bidirectional data flow through mocked I/O
    masterStdio.WriteToStdin([]byte("test\n"))
    output := slaveStdio.ReadFromStdout()
    // Assert output matches expected
}
```

**Key Points**:
- Integration tests validate tool workflows without touching real system resources
- They test at the entrypoint level (highest level before CLI)
- Must remain passing - they verify complete tool behavior
- See `test/integration/README.md` for more details

### E2E Tests (in test/e2e/)

**Purpose**: Validate real compiled binaries  
**Location**: `test/e2e/` directory  
**Environment**: Docker containers with Alpine Linux and expect scripts  
**No mocking**: Uses actual compiled binaries with real network

**Run E2E tests**:
```bash
make test              # All tests (unit + E2E)
make test-integration  # Only E2E tests
```

**Requirements**: Docker and Docker Compose  
**Duration**: ~2-3 minutes  
**Scenarios**: Bind shell and reverse shell with all transport protocols (tcp, ws, wss)

## Summary

Write tests that:
- Are fast, focused, and deterministic
- Cover important behaviors and edge cases
- Are easy to read and maintain
- Serve as documentation
- Catch regressions early
- Enable confident refactoring
