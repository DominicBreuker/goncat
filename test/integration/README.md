# Integration Testing

This directory contains integration test utilities and examples for testing goncat functionality with mocked dependencies.

## Key Features

**Fast and Deterministic Tests**: The mock implementations provide `WaitForOutput` and `WaitForListener` methods that block until expected conditions are met, eliminating the need for arbitrary `time.Sleep` calls. This makes tests both faster (9x improvement) and more reliable.

## Mock TCP Network

The `mocktcp.go` file provides a `MockTCPNetwork` that simulates TCP connections without using real network sockets.

**Features**: 
- In-memory connections via `net.Pipe()`
- Multiple listeners
- Automatic lifecycle management
- `WaitForListener(addr, timeoutMs)` - Blocks until a listener is bound to the specified address or timeout expires. Useful for synchronizing test flow without arbitrary sleeps.

**Usage**:
```go
mockNet := NewMockTCPNetwork()
deps := &config.Dependencies{
    TCPDialer:   mockNet.DialTCP,
    TCPListener: mockNet.ListenTCP,
}

// Wait for a service to start listening
if err := mockNet.WaitForListener("127.0.0.1:12345", 2000); err != nil {
    t.Fatalf("Service failed to start: %v", err)
}
```

## Mock Standard I/O

The `mockstdio.go` file provides `MockStdio` for mocking stdin and stdout streams.

**Features**: 
- Buffer-based stdin/stdout
- Thread-safe read/write operations
- `WaitForOutput(expected, timeoutMs)` - Blocks until the expected string appears in stdout or timeout expires. Useful for synchronizing test flow without arbitrary sleeps.

**Usage**:
```go
mockStdio := NewMockStdio()
mockStdio.WriteToStdin([]byte("input data"))
deps := &config.Dependencies{
    Stdin:  func() io.Reader { return mockStdio.GetStdin() },
    Stdout: func() io.Writer { return mockStdio.GetStdout() },
}

// Wait for expected output instead of sleeping
if err := mockStdio.WaitForOutput("expected text", 2000); err != nil {
    t.Errorf("Expected output not found: %v", err)
}

// Or check all output collected so far
output := mockStdio.ReadFromStdout()
```

## Mock Command Execution

The `mockexec.go` file provides `MockExec` for mocking command execution without running real processes.

**Features**: Simulates `/bin/sh` shell behavior by responding to specific commands:
- `echo <text>` - outputs the text
- `whoami` - outputs `mockcmd[<program>]`
- `exit` - terminates the shell
- Other commands - outputs error message

**Usage**:
```go
mockExec := NewMockExec()
deps := &config.Dependencies{
    ExecCommand: mockExec.Command,
}
// Command will behave like a simple shell
```

## Dependency Injection

Uses `config.Dependencies` struct to inject mocks. Helper functions (`GetTCPDialerFunc`, `GetTCPListenerFunc`, `GetStdinFunc`, `GetStdoutFunc`, `GetExecCommandFunc`) return either mock or default implementations.

**Current mocks**: TCP network, stdin/stdout, command execution.

## Integration Tests

### TestEndToEndDataExchange
`TestEndToEndDataExchange` in `simple_test.go` demonstrates full end-to-end testing:
- Simulates "goncat master listen tcp://*:12345" and "goncat slave connect tcp://127.0.0.1:12345"
- Uses mocked TCP network and stdio on both sides
- Validates bidirectional data flow: master stdin → network → slave stdout and vice versa
- Tests complete master-slave handler lifecycle with mocked dependencies

### TestExecCommandExecution
`TestExecCommandExecution` in `exec_test.go` demonstrates command execution testing:
- Simulates "goncat master listen tcp://*:12345 --exec /bin/sh" and "goncat slave connect tcp://127.0.0.1:12345"
- Uses mocked TCP network, stdio, and command execution
- Validates specific shell commands: `echo`, `whoami`, unsupported commands, and `exit`
- Tests that the slave terminates when the shell exits
- Tests the --exec feature without spawning real processes

### TestLocalPortForwarding
`TestLocalPortForwarding` in `local_port_forward_test.go` demonstrates local port forwarding testing:
- Simulates "goncat master listen tcp://*:12345 -L 8000:127.0.0.1:9000" and "goncat slave connect tcp://127.0.0.1:12345"
- Uses mocked TCP network for all connections (master-slave tunnel, forwarded port, and remote server)
- Creates a mock remote server at 127.0.0.1:9000 that responds with unique, verifiable data
- Tests a mock client connecting to the forwarded port 8000 on the master side
- Validates complete bidirectional data flow through the port forwarding tunnel
- Tests multiple connections to ensure stability
- Demonstrates realistic port forwarding scenario entirely in-memory

### TestRemotePortForwarding
`TestRemotePortForwarding` in `remote_port_forward_test.go` demonstrates remote port forwarding testing:
- Simulates "goncat master listen tcp://*:12345 -R 127.0.0.1:8000:127.0.0.1:9000" and "goncat slave connect tcp://127.0.0.1:12345"
- Uses mocked TCP network for all connections (master-slave tunnel, forwarded port on slave, and remote server on master)
- Creates a mock remote server at 127.0.0.1:9000 on the master side that responds with unique, verifiable data
- Tests a mock client connecting to the forwarded port 8000 on the slave side
- Validates complete bidirectional data flow through the reverse port forwarding tunnel
- Tests multiple connections to ensure stability
- Demonstrates the reverse of local port forwarding: slave binds the port, master provides the destination

## Test Helpers

The `test/helpers/` directory contains utilities to reduce boilerplate in tests:

**SetupMockDependencies()**: Creates mock network and stdio dependencies  
**SetupMockDependenciesWithExec()**: Also includes mock exec for command testing  
**DefaultSharedConfig()**: Creates standard Shared config with sensible defaults  
**DefaultMasterConfig()**: Creates standard Master config with sensible defaults

These helpers allow tests to focus on test-specific configuration while reusing common setup code.
