# Testing Guidelines for goncat

This document provides comprehensive testing guidelines for the goncat project. We follow idiomatic Go testing practices to ensure code quality, maintainability, and reliability.

## Overview

Our testing strategy includes:
- **Unit tests**: Test individual functions and components in isolation (in `*_test.go` files)
- **Integration tests**: Test complete flows with mocked dependencies (in `test/integration/`)
- **E2E tests**: End-to-end tests with real binaries in Docker (in `test/e2e/`)
- **Coverage goals**: Aim for meaningful coverage of edge cases and error paths, not just 100% line coverage
- **Test organization**: Unit tests live alongside code; integration/E2E tests in `test/` directory

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
- Run with race detector in CI: `go test -race ./...`

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

## Interfaces & Fakes

- Design code to accept interfaces for easy testing
- Use simple fakes/stubs over heavy mocking frameworks
- Keep fakes minimal and explicit:

```go
type fakeConn struct {
    readData  []byte
    writeData []byte
    closed    bool
}

func (f *fakeConn) Read(p []byte) (n int, err error) {
    n = copy(p, f.readData)
    f.readData = f.readData[n:]
    if len(f.readData) == 0 {
        err = io.EOF
    }
    return
}

func (f *fakeConn) Write(p []byte) (n int, err error) {
    f.writeData = append(f.writeData, p...)
    return len(p), nil
}

func (f *fakeConn) Close() error {
    f.closed = true
    return nil
}
```

## Dependency Injection & Mocks

The project uses dependency injection to support testing without real system resources. Most functions that interact with external systems (network, filesystem, command execution) accept an optional `*config.Dependencies` parameter.

### Mock Infrastructure

The `mocks/` package provides mock implementations:
- **MockTCPNetwork**: In-memory TCP network using `net.Pipe()`
- **MockStdio**: Simulated stdin/stdout with buffers
- **MockExec**: Simulated command execution without spawning processes

### Using Mocks in Unit Tests

When writing unit tests, prefer mocks over real system resources:

```go
func TestClient_Connect(t *testing.T) {
    // Create mock network
    mockNet := mocks.NewMockTCPNetwork()
    deps := &config.Dependencies{
        TCPDialer:   mockNet.DialTCP,
        TCPListener: mockNet.ListenTCP,
    }

    // Create a listener on mock network
    tcpAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:12345")
    listener, _ := mockNet.ListenTCP("tcp", tcpAddr)
    defer listener.Close()

    // Test dialing with mock network
    dialer, _ := tcp.NewDialer("127.0.0.1:12345", deps)
    conn, err := dialer.Dial()
    if err != nil {
        t.Fatalf("Dial() error = %v", err)
    }
    defer conn.Close()
}
```

### Using MockExec for Command Testing

```go
func TestRunCommand(t *testing.T) {
    mockExec := mocks.NewMockExec()
    deps := &config.Dependencies{
        ExecCommand: mockExec.Command,
    }

    conn := newFakeConn()
    conn.readBuf.WriteString("echo hello\n")
    conn.readBuf.WriteString("exit\n")

    err := exec.Run(ctx, conn, "/bin/sh", deps)
    if err != nil {
        t.Fatalf("Run() error = %v", err)
    }

    output := conn.writeBuf.String()
    if !strings.Contains(output, "hello") {
        t.Errorf("output = %q, want to contain 'hello'", output)
    }
}
```

### Benefits of Mocks

- **Fast**: No network latency or process spawning
- **Reliable**: No port conflicts or race conditions
- **Isolated**: Tests don't affect or depend on system state
- **CI-friendly**: No special privileges or resources needed
- **Deterministic**: Predictable behavior without timing issues

### When to Use Real Resources

Avoid mocks for:
- Integration tests validating complete workflows
- E2E tests in `test/e2e/` using Docker
- Tests specifically validating real system behavior

See `test/integration/README.md` for integration testing with mocks.

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

## Documentation

Tests serve as documentation:
- Use descriptive test names
- Include comments for complex test logic
- Document assumptions and invariants
- Test names should read like specifications

## Examples

See existing tests for examples:
- `cmd/shared/parsers_test.go` - table-driven tests with error cases
- More examples will be added as we expand test coverage

## Integration Tests

Integration tests in `test/integration/` validate complete workflows using mocked dependencies. These tests:
- Use mocked network connections (`mocks/mocktcp.go`) and stdio (`mocks/mockstdio.go`)
- Test the four operation modes via entrypoint functions in `pkg/entrypoint/`
- Validate bidirectional data flow without real network or terminal I/O
- Run fast (~2 seconds) and are fully deterministic

### Writing Integration Tests

1. **Setup mocks**: Use helpers from `test/helpers/` to create mock dependencies
2. **Configure**: Create `config.Shared` and mode-specific configs (e.g., `config.Master`)
3. **Use entrypoints**: Call `entrypoint.MasterListen()`, `entrypoint.SlaveConnect()`, etc.
4. **Validate**: Check that data flows correctly through mocked stdio

Example structure:
```go
func TestMyFlow(t *testing.T) {
    mockNet := mocks.NewMockTCPNetwork()
    masterStdio := mocks.NewMockStdio()
    slaveSt dio := mocks.NewMockStdio()
    
    // Setup configs with mocked dependencies
    masterCfg := &config.Shared{
        Protocol: config.ProtoTCP,
        Deps: &config.Dependencies{
            TCPDialer: mockNet.DialTCP,
            TCPListener: mockNet.ListenTCP,
            Stdin: func() io.Reader { return masterStdio.GetStdin() },
            Stdout: func() io.Writer { return masterStdio.GetStdout() },
        },
    }
    
    // Start master and slave using entrypoint functions
    go entrypoint.MasterListen(ctx, masterCfg, masterModeCfg)
    go entrypoint.SlaveConnect(ctx, slaveCfg)
    
    // Write to master stdin, verify appears on slave stdout
    masterStdio.WriteToStdin([]byte("test"))
    output := slaveStdio.ReadFromStdout()
}
```

**Important**: Integration tests must remain passing. They validate the complete tool behavior.

## Summary

Write tests that:
- Are fast, focused, and deterministic
- Cover important behaviors and edge cases
- Are easy to read and maintain
- Serve as documentation
- Catch regressions early
- Enable confident refactoring
