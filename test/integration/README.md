# Integration Testing

This directory contains integration test utilities and examples for testing goncat functionality with mocked dependencies.

## Mock TCP Network

The `mocktcp.go` file provides a `MockTCPNetwork` that simulates TCP connections without using real network sockets.

**Features**: In-memory connections via `net.Pipe()`, multiple listeners, automatic lifecycle management.

**Usage**:
```go
mockNet := NewMockTCPNetwork()
deps := &config.Dependencies{
    TCPDialer:   mockNet.DialTCP,
    TCPListener: mockNet.ListenTCP,
}
```

## Mock Standard I/O

The `mockstdio.go` file provides `MockStdio` for mocking stdin and stdout streams.

**Features**: Buffer-based stdin/stdout, thread-safe read/write operations.

**Usage**:
```go
mockStdio := NewMockStdio()
mockStdio.WriteToStdin([]byte("input data"))
deps := &config.Dependencies{
    Stdin:  func() io.Reader { return mockStdio.GetStdin() },
    Stdout: func() io.Writer { return mockStdio.GetStdout() },
}
// Check output: mockStdio.ReadFromStdout()
```

## Mock Command Execution

The `mockexec.go` file provides `MockExec` for mocking command execution without running real processes.

**Features**: Simulates command execution by echoing stdin to stdout, no real process spawning.

**Usage**:
```go
mockExec := NewMockExec()
deps := &config.Dependencies{
    ExecCommand: mockExec.Command,
}
// Command will echo input back to output
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
- Validates that commands sent from master are executed on slave and output is returned
- Tests the --exec feature without spawning real processes
