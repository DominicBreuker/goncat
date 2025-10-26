# Integration Testing

This directory contains integration test utilities and examples for testing goncat functionality with mocked dependencies.

## Directory Structure

Tests are organized by feature:
- `plain/` - Basic end-to-end connection tests without special features
- `exec/` - Command execution tests (--exec flag)
- `portfwd/` - Port forwarding tests (local and remote)
- `socks/` - SOCKS proxy tests
  - `connect/` - SOCKS5 CONNECT tests
  - `associate/` - SOCKS5 UDP ASSOCIATE tests
- `udp/` - UDP transport tests with QUIC protocol

## Key Features

**Fast and Deterministic Tests**: The mock implementations provide `WaitForOutput` and `WaitForListener` methods that block until expected conditions are met, eliminating the need for arbitrary `time.Sleep` calls. This makes tests both faster (9x improvement) and more reliable.

## Unified Test Setup

Integration tests use a unified helper function `helpers.SetupMockDependenciesAndConfigs()` that provides all necessary mocks and default configurations in a single call.

**Usage**:
```go
func TestExample(t *testing.T) {
    // Setup all mocks and default configs
    setup := helpers.SetupMockDependenciesAndConfigs()
    defer setup.Close()

    // Modify configs as needed for this specific test
    setup.MasterCfg.Exec = "/bin/sh"
    setup.MasterCfg.LocalPortForwarding = []*config.LocalPortForwardingCfg{
        {LocalHost: "127.0.0.1", LocalPort: 8000, RemoteHost: "127.0.0.1", RemotePort: 9000},
    }

    // Use setup.TCPNetwork, setup.UDPNetwork, setup.MasterStdio, etc.
    // Access configs via setup.MasterSharedCfg, setup.SlaveSharedCfg, setup.MasterCfg
}
```

The `MockDependenciesAndConfigs` struct includes:
- **Mock Networks**: `TCPNetwork` (from `mocks/tcp`), `UDPNetwork`
- **Mock Stdio**: `MasterStdio`, `SlaveStdio` for master and slave processes
- **Mock Exec**: `MockExec` for command execution tests
- **Dependencies**: `MasterDeps`, `SlaveDeps` pre-configured with all mocks
- **Default Configs**: `MasterSharedCfg`, `SlaveSharedCfg`, `MasterCfg` ready to use or modify
- **Cleanup**: `Close()` method for proper resource cleanup

## Mock TCP Network

The `mocks/tcp` package provides a `MockTCPNetwork` that simulates TCP connections without using real network sockets.

**Features**: 
- In-memory connections via `net.Pipe()`
- Multiple listeners
- Automatic lifecycle management
- `WaitForListener(addr, timeoutMs)` - Blocks until a listener is bound to the specified address or timeout expires. Returns the listener and an error. Useful for synchronizing test flow without arbitrary sleeps.
- Helper types: `Client` and `Server` for easy test client/server creation

**Usage**:
```go
// Already available in setup
listener, err := setup.TCPNetwork.WaitForListener("127.0.0.1:12345", 2000)
if err != nil {
    t.Fatalf("Service failed to start: %v", err)
}

// Create a test client
client, err := mocks_tcp.NewClient(setup.TCPNetwork.DialTCP, "tcp", "127.0.0.1:8000")
if err != nil {
    t.Fatalf("failed to connect: %v", err)
}
defer client.Close()

// Create a test server
srv, err := mocks_tcp.NewServer(setup.TCPNetwork.ListenTCP, "tcp", "127.0.0.1:9000", "RESPONSE: ")
if err != nil {
    t.Fatalf("failed to start server: %v", err)
}
defer srv.Close()
```

## Mock Standard I/O

The `mockstdio.go` file provides `MockStdio` for mocking stdin and stdout streams.

**Features**: 
- Buffer-based stdin/stdout
- Thread-safe read/write operations
- `WaitForOutput(expected, timeoutMs)` - Blocks until the expected string appears in stdout or timeout expires. Useful for synchronizing test flow without arbitrary sleeps.

**Usage**:
```go
// Already available in setup
setup.MasterStdio.WriteToStdin([]byte("input data"))

// Wait for expected output instead of sleeping
if err := setup.MasterStdio.WaitForOutput("expected text", 2000); err != nil {
    t.Errorf("Expected output not found: %v", err)
}

// Or check all output collected so far
output := setup.MasterStdio.ReadFromStdout()
```

## Mock UDP Network

The `mockudp.go` file provides `MockUDPNetwork` for mocking UDP connections.

**Features**: 
- In-memory packet delivery via channels
- Multiple listeners
- `WaitForListener(addr, timeoutMs)` - Similar to TCP mock

**Usage**:
```go
// Already available in setup
listener, err := setup.UDPNetwork.ListenUDP("udp", udpAddr)
if err != nil {
    t.Fatalf("Failed to create UDP listener: %v", err)
}
defer listener.Close()
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
// Already available in setup via dependencies
// The mock exec is pre-configured in both master and slave dependencies
// Just set the Exec field in the master config
setup.MasterCfg.Exec = "/bin/sh"
```

## Integration Tests

### Plain Connection Tests (`plain/`)

**TestEndToEndDataExchange** (`plain_test.go`):
- Simulates "goncat master listen tcp://*:12345" and "goncat slave connect tcp://127.0.0.1:12345"
- Uses mocked TCP network and stdio on both sides
- Validates bidirectional data flow: master stdin → network → slave stdout and vice versa
- Tests complete master-slave handler lifecycle with mocked dependencies

### Command Execution Tests (`exec/`)

**TestExecCommandExecution** (`exec_test.go`):
- Simulates "goncat master listen tcp://*:12345 --exec /bin/sh" and "goncat slave connect tcp://127.0.0.1:12345"
- Uses mocked TCP network, stdio, and command execution
- Validates specific shell commands: `echo`, `whoami`, unsupported commands, and `exit`
- Tests that the slave terminates when the shell exits
- Tests the --exec feature without spawning real processes

### Port Forwarding Tests (`portfwd/`)

**TestLocalPortForwarding** (`portfwd/local_test.go`):
- Simulates "goncat master listen tcp://*:12345 -L 8000:127.0.0.1:9000" and "goncat slave connect tcp://127.0.0.1:12345"
- Uses mocked TCP network for all connections (master-slave tunnel, forwarded port, and remote server)
- Creates a mock remote server at 127.0.0.1:9000 that responds with unique, verifiable data
- Tests a mock client connecting to the forwarded port 8000 on the master side
- Validates complete bidirectional data flow through the port forwarding tunnel
- Demonstrates realistic port forwarding scenario entirely in-memory
- Uses `WaitForListener` and `WaitForNewConnection` to synchronize test flow

**TestRemotePortForwarding** (`portfwd/remote_test.go`):
- Simulates "goncat master listen tcp://*:12345 -R 127.0.0.1:8000:127.0.0.1:9000" and "goncat slave connect tcp://127.0.0.1:12345"
- Uses mocked TCP network for all connections (master-slave tunnel, forwarded port on slave, and remote server on master)
- Creates a mock remote server at 127.0.0.1:9000 on the master side that responds with unique, verifiable data
- Tests a mock client connecting to the forwarded port 8000 on the slave side
- Validates complete bidirectional data flow through the reverse port forwarding tunnel
- Demonstrates the reverse of local port forwarding: slave binds the port, master provides the destination
- Uses `WaitForListener` and `WaitForNewConnection` to synchronize test flow

### SOCKS Proxy Tests (`socks/`)

**TestSocksConnect** (`socks/connect/connect_test.go`):
- Simulates "goncat master listen tcp://*:12345 -D 127.0.0.1:1080" and "goncat slave connect tcp://127.0.0.1:12345"
- Uses mocked TCP network for all connections (master-slave tunnel, SOCKS proxy, and destination server)
- Creates a mock destination server at 127.0.0.1:8080 on the slave side using `mocks_tcp.NewServer` with line-oriented echo
- Implements full SOCKS5 client handshake (method selection, CONNECT request)
- Tests a mock SOCKS client connecting to the proxy at 127.0.0.1:1080 on the master side
- Validates complete bidirectional data flow through the SOCKS proxy tunnel
- Uses `WaitForListener` and `WaitForNewConnection` to synchronize test flow without arbitrary sleeps
- Tests multiple connections to ensure stability
- Demonstrates realistic SOCKS proxy scenario entirely in-memory
- Note: Only tests SOCKS CONNECT; ASSOCIATE (UDP) is tested separately

**SOCKS UDP ASSOCIATE Tests** (`socks/associate/`):

The `socks/associate/` directory contains tests for SOCKS5 UDP ASSOCIATE functionality:

- **TestSingleClient** (`single_client_test.go`): Tests SOCKS5 UDP ASSOCIATE with a single client, validates complete UDP packet flow through the SOCKS proxy

- **TestMultipleClients** (`multiple_clients_test.go`): Tests SOCKS5 UDP ASSOCIATE with multiple concurrent clients, each client gets its own UDP relay and can send/receive independently

- **Shared Setup** (`helpers.go`): `SetupTest()` creates all test infrastructure using the unified helper, `CreateSOCKSClient()` performs SOCKS5 handshake and UDP ASSOCIATE request, `SOCKSClient.SendUDPDatagram()` sends and receives SOCKS5 UDP datagrams
