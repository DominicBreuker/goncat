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

## Dependency Injection

Uses `config.Dependencies` struct to inject mocks. Helper functions (`GetTCPDialerFunc`, `GetTCPListenerFunc`, `GetStdinFunc`, `GetStdoutFunc`) return either mock or default implementations.

**Current mocks**: TCP network, stdin/stdout.

## Integration Test

`TestEndToEndDataExchange` in `simple_test.go` demonstrates full end-to-end testing:
- Simulates "goncat master listen tcp://*:12345" and "goncat slave connect tcp://127.0.0.1:12345"
- Uses mocked TCP network and stdio on both sides
- Validates bidirectional data flow: master stdin → network → slave stdout and vice versa
- Tests complete master-slave handler lifecycle with mocked dependencies
