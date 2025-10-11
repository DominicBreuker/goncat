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

**Current mocks**: TCP network, stdin/stdout. See `simple_test.go` for usage examples.

**Adding mocks**: (1) Add function type to `Dependencies`, (2) Add helper function, (3) Update code to use helper, (4) Create mock implementation.
