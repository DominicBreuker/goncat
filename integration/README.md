# Integration Testing

This directory contains integration test utilities and examples for testing goncat functionality with mocked dependencies.

## Mock TCP Network

The `mocktcp.go` file provides a `MockTCPNetwork` that simulates TCP connections without using real network sockets. This enables fast, reliable integration tests that don't require actual network access.

### Features

- **In-memory connections**: Uses `net.Pipe()` to create connected pairs of connections
- **Multiple listeners**: Supports multiple listeners on different addresses
- **Connection tracking**: Automatically manages connection lifecycle
- **Interface compatibility**: Returns `net.Listener` and `net.Conn` interfaces for seamless integration

### Usage

```go
// Create a mock network
mockNet := NewMockTCPNetwork()

// Create dependencies with the mock
deps := &config.Dependencies{
    TCPDialer:   mockNet.DialTCP,
    TCPListener: mockNet.ListenTCP,
}

// Use in configurations
cfg := &config.Shared{
    Protocol: config.ProtoTCP,
    Host:     "127.0.0.1",
    Port:     12345,
    Deps:     deps,
}

// Now server.New() and client.New() will use the mock network
```

### Example Test

See `simple_test.go` for a complete example that demonstrates:
- Creating a master server that listens for connections
- Creating a slave client that connects to the server
- Sending and receiving data through the mocked connection
- Proper cleanup and goroutine coordination

## Dependency Injection

The dependency injection system is implemented through the `config.Dependencies` struct in `pkg/config/config.go`. This allows injecting mock implementations for:

- **TCP Dialer**: Mock `net.DialTCP` functionality
- **TCP Listener**: Mock `net.ListenTCP` functionality

Future enhancements may include mocking other dependencies like:
- Standard input/output
- File system operations
- Time functions

## Adding New Mocks

To add a new mockable dependency:

1. Add a new function type to `config.Dependencies` struct
2. Update the relevant package to use the injected dependency if provided
3. Create a mock implementation in the `integration/` directory
4. Write tests demonstrating the mock usage

## Notes

- All mocks should implement standard Go interfaces when possible
- Mocks should be minimal and focused on the behavior needed for testing
- Production code should use `nil` for `Deps` field to use default implementations
- Tests should clean up resources properly (close connections, cancel contexts, etc.)
