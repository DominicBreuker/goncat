# Plan for UDP Transport Implementation

Add UDP transport support to goncat using KCP protocol for reliable communication over UDP. The UDP transport will create a `net.PacketConn` using Go stdlib, then upgrade it to a reliable connection using the kcp-go library's `UDPSession`, which provides a `net.Conn`-compatible interface.

## Overview

The UDP transport needs to integrate with goncat's existing transport architecture while providing reliable, ordered delivery on top of UDP. The implementation will:

1. Add `udp` protocol support alongside existing `tcp`, `ws`, and `wss` protocols
2. Create UDP packet connections using Go stdlib (`net.ListenPacket`, `net.DialUDP`)
3. Upgrade the packet connection to a reliable KCP session using `github.com/xtaci/kcp-go`
4. Provide standard `transport.Dialer` and `transport.Listener` interfaces that return `net.Conn`
5. Handle KCP configuration for optimal performance (block cipher, FEC parameters, etc.)

Key architectural principle: We create the raw UDP `net.PacketConn` ourselves using Go stdlib (not using kcp-go's dial/listen functions directly), then only use kcp-go to upgrade it to a reliable connection. This allows future flexibility to create PacketConns through other mechanisms.

## Implementation Plan

- [ ] Step 1: Add KCP library dependency
  - **Task**: Add `github.com/xtaci/kcp-go` to project dependencies
  - **Files**: 
    - `go.mod`: Add dependency via `go get github.com/xtaci/kcp-go/v5@latest`
    - `go.sum`: Auto-generated checksum file
  - **Dependencies**: None
  - **Validation**: Run `go mod tidy` and verify dependency is added

- [ ] Step 2: Add UDP protocol constant and parsing
  - **Task**: Extend config package to support UDP protocol type
  - **Files**:
    - `pkg/config/config.go`: Add `ProtoUDP = 4` constant and update `String()` method
      ```go
      const (
          ProtoTCP = 1
          ProtoWS  = 2
          ProtoWSS = 3
          ProtoUDP = 4  // New constant
      )
      
      func (p Protocol) String() string {
          // Add case for ProtoUDP returning "udp"
      }
      ```
    - `cmd/shared/parsers.go`: Update regex to accept `udp` protocol
      ```go
      // Change regex from `^(tcp|ws|wss)://` to `^(tcp|ws|wss|udp)://`
      // Add case for "udp" in switch statement
      case "udp":
          proto = config.ProtoUDP
      ```
    - `cmd/shared/parsers_test.go`: Add test cases for UDP protocol parsing
      ```go
      // Add test cases:
      {"udp protocol", "udp://localhost:12345", config.ProtoUDP, "localhost", 12345, false},
      {"udp wildcard", "udp://*:12345", config.ProtoUDP, "", 12345, false},
      ```
  - **Dependencies**: None
  - **Validation**: Run `go test ./cmd/shared/...` to verify parsing tests pass

- [ ] Step 3: Create UDP transport package structure
  - **Task**: Create new package `pkg/transport/udp` with dialer and listener implementations
  - **Files**:
    - `pkg/transport/udp/dialer.go`: UDP dialer implementation
    - `pkg/transport/udp/listener.go`: UDP listener implementation
    - `pkg/transport/udp/dialer_test.go`: Unit tests for dialer
    - `pkg/transport/udp/listener_test.go`: Unit tests for listener
  - **Dependencies**: Step 1, Step 2
  - **Validation**: Files created, compile check with `go build ./pkg/transport/udp/...`

- [ ] Step 4: Implement UDP Dialer
  - **Task**: Create UDP dialer that establishes KCP session over UDP connection
  - **Files**:
    - `pkg/transport/udp/dialer.go`: Complete implementation
      ```go
      package udp
      
      import (
          "context"
          "dominicbreuker/goncat/pkg/config"
          "fmt"
          "net"
          
          kcp "github.com/xtaci/kcp-go/v5"
      )
      
      type Dialer struct {
          remoteAddr *net.UDPAddr
          packetConnFn config.PacketListenerFunc  // For creating PacketConn
      }
      
      func NewDialer(addr string, deps *config.Dependencies) (*Dialer, error) {
          // Parse UDP address
          udpAddr, err := net.ResolveUDPAddr("udp", addr)
          
          // Get PacketListener function from dependencies or use default
          packetConnFn := config.GetPacketListenerFunc(deps)
          
          return &Dialer{
              remoteAddr: udpAddr,
              packetConnFn: packetConnFn,
          }, nil
      }
      
      func (d *Dialer) Dial(ctx context.Context) (net.Conn, error) {
          // Create UDP packet connection using stdlib
          // Use ":0" for local address to let OS choose port
          conn, err := d.packetConnFn("udp", ":0")
          
          // Upgrade to KCP session using kcp.NewConn
          // Parameters: remoteAddr, block cipher (nil for now), dataShards (0), parityShards (0), conn
          kcpConn, err := kcp.NewConn(d.remoteAddr.String(), nil, 0, 0, conn)
          
          // Configure KCP for optimal performance
          // SetNoDelay(nodelay, interval, resend, nc)
          kcpConn.SetNoDelay(1, 10, 2, 1)
          kcpConn.SetStreamMode(true)
          kcpConn.SetWindowSize(1024, 1024)
          
          return kcpConn, nil
      }
      ```
    - `pkg/transport/udp/dialer_test.go`: Unit tests similar to TCP dialer tests
      ```go
      // Test cases:
      // - Valid address parsing
      // - Invalid address (no port, bad format)
      // - IPv4 and IPv6 addresses
      ```
  - **Dependencies**: Step 1, Step 2, Step 3
  - **Validation**: Run `go test ./pkg/transport/udp/...` to verify dialer tests pass

- [ ] Step 5: Implement UDP Listener
  - **Task**: Create UDP listener that accepts KCP sessions over UDP
  - **Files**:
    - `pkg/transport/udp/listener.go`: Complete implementation
      ```go
      package udp
      
      import (
          "dominicbreuker/goncat/pkg/config"
          "dominicbreuker/goncat/pkg/log"
          "dominicbreuker/goncat/pkg/transport"
          "fmt"
          "net"
          
          kcp "github.com/xtaci/kcp-go/v5"
      )
      
      type Listener struct {
          kcpListener *kcp.Listener
          sem         chan struct{} // capacity 1 for single connection
      }
      
      func NewListener(addr string, deps *config.Dependencies) (*Listener, error) {
          // Parse UDP address
          udpAddr, err := net.ResolveUDPAddr("udp", addr)
          
          // Create UDP packet connection using stdlib
          packetConnFn := config.GetPacketListenerFunc(deps)
          conn, err := packetConnFn("udp", addr)
          
          // Create KCP listener from packet connection
          // ServeConn wraps a PacketConn and returns a Listener
          kcpListener, err := kcp.ServeConn(nil, 0, 0, conn)
          
          l := &Listener{
              kcpListener: kcpListener,
              sem:         make(chan struct{}, 1),
          }
          l.sem <- struct{}{} // initially allow one connection
          return l, nil
      }
      
      func (l *Listener) Serve(handle transport.Handler) error {
          for {
              // Accept KCP session (blocks until connection)
              kcpConn, err := l.kcpListener.AcceptKCP()
              
              // Configure KCP session
              kcpConn.SetNoDelay(1, 10, 2, 1)
              kcpConn.SetStreamMode(true)
              kcpConn.SetWindowSize(1024, 1024)
              
              // Semaphore logic similar to TCP listener
              select {
              case <-l.sem:
                  go func(conn *kcp.UDPSession) {
                      defer func() {
                          conn.Close()
                          l.sem <- struct{}{}
                      }()
                      
                      if err := handle(conn); err != nil {
                          log.ErrorMsg("Handling connection: %s\n", err)
                      }
                  }(kcpConn)
              default:
                  kcpConn.Close()
              }
          }
      }
      
      func (l *Listener) Close() error {
          return l.kcpListener.Close()
      }
      ```
    - `pkg/transport/udp/listener_test.go`: Unit tests similar to TCP listener tests
      ```go
      // Test cases:
      // - Valid listener creation
      // - Invalid address handling
      // - Single connection acceptance
      // - Listener close
      ```
  - **Dependencies**: Step 1, Step 2, Step 3, Step 4
  - **Validation**: Run `go test ./pkg/transport/udp/...` to verify all tests pass

- [ ] Step 6: Integrate UDP transport into server
  - **Task**: Update server package to support UDP protocol
  - **Files**:
    - `pkg/server/server.go`: Add UDP case in `Serve()` method
      ```go
      import "dominicbreuker/goncat/pkg/transport/udp"
      
      func (s *Server) Serve() error {
          // ... existing code ...
          
          switch s.cfg.Protocol {
          case config.ProtoWS, config.ProtoWSS:
              l, err = ws.NewListener(s.ctx, addr, s.cfg.Protocol == config.ProtoWSS)
          case config.ProtoUDP:
              l, err = udp.NewListener(addr, s.cfg.Deps)  // New case
          default:
              l, err = tcp.NewListener(addr, s.cfg.Deps)
          }
          
          // ... rest of code ...
      }
      ```
  - **Dependencies**: Step 5
  - **Validation**: Run `go test ./pkg/server/...` to verify server tests still pass

- [ ] Step 7: Integrate UDP transport into client
  - **Task**: Update client package to support UDP protocol
  - **Files**:
    - `pkg/client/client.go`: Add UDP case in `connect()` method
      ```go
      import "dominicbreuker/goncat/pkg/transport/udp"
      
      type dependencies struct {
          newTCPDialer func(string, *config.Dependencies) (transport.Dialer, error)
          newWSDialer  func(context.Context, string, config.Protocol) transport.Dialer
          newUDPDialer func(string, *config.Dependencies) (transport.Dialer, error)  // New field
          tlsUpgrader  func(net.Conn, string, time.Duration) (net.Conn, error)
      }
      
      func (c *Client) Connect() error {
          deps := &dependencies{
              // ... existing dialers ...
              newUDPDialer: func(addr string, deps *config.Dependencies) (transport.Dialer, error) {
                  return udp.NewDialer(addr, deps)
              },
              tlsUpgrader: upgradeToTLS,
          }
          return c.connect(deps)
      }
      
      func (c *Client) connect(deps *dependencies) error {
          // ... existing code ...
          
          var d transport.Dialer
          var err error
          switch c.cfg.Protocol {
          case config.ProtoWS, config.ProtoWSS:
              d = deps.newWSDialer(c.ctx, addr, c.cfg.Protocol)
          case config.ProtoUDP:
              d, err = deps.newUDPDialer(addr, c.cfg.Deps)  // New case
          default:
              d, err = deps.newTCPDialer(addr, c.cfg.Deps)
          }
          
          // ... rest of code ...
      }
      ```
    - `pkg/client/client_test.go`: Add test cases for UDP protocol
      ```go
      // Add test case for ProtoUDP similar to TCP test case
      ```
  - **Dependencies**: Step 4, Step 6
  - **Validation**: Run `go test ./pkg/client/...` to verify client tests pass

- [ ] Step 8: Add UDP mock for integration tests
  - **Task**: Create mock KCP connection for integration testing
  - **Files**:
    - `mocks/udp/kcp.go`: Mock KCP network using in-memory connections
      ```go
      package mockudp
      
      import (
          "net"
          "sync"
      )
      
      // MockKCPNetwork provides in-memory KCP-like connections for testing
      type MockKCPNetwork struct {
          listeners map[string]*MockKCPListener
          mu        sync.Mutex
          cond      *sync.Cond
      }
      
      func NewMockKCPNetwork() *MockKCPNetwork {
          m := &MockKCPNetwork{
              listeners: make(map[string]*MockKCPListener),
          }
          m.cond = sync.NewCond(&m.mu)
          return m
      }
      
      // ListenPacket creates a mock packet connection
      func (m *MockKCPNetwork) ListenPacket(network, address string) (net.PacketConn, error) {
          // Return a mock PacketConn that integrates with mock network
          // This will be wrapped by KCP in real code, but for tests we provide
          // a simpler mock that directly returns net.Conn from Dial/Accept
      }
      
      // WaitForListener waits for a listener to be bound
      func (m *MockKCPNetwork) WaitForListener(addr string, timeoutMs int) (*MockKCPListener, error) {
          // Similar to MockTCPNetwork implementation
      }
      ```
    - Note: For integration tests, we may want to use a simpler approach where the mock PacketListener directly creates paired net.Pipe() connections rather than actually using KCP, since we're testing the tool's logic, not KCP itself
  - **Dependencies**: Step 5
  - **Validation**: Create and verify mock compiles

- [ ] Step 9: Add integration tests for UDP transport
  - **Task**: Create integration tests validating UDP transport works end-to-end
  - **Files**:
    - `test/integration/plain/udp_test.go`: Test basic UDP connection
      ```go
      func TestUDPEndToEndDataExchange(t *testing.T) {
          // Setup similar to TestEndToEndDataExchange in plain_test.go
          // Use ProtoUDP instead of ProtoTCP
          // Configure master and slave with UDP protocol
          // Verify bidirectional data flow works
      }
      ```
    - Update `test/integration/helpers/helpers.go` if needed to support UDP mocks
  - **Dependencies**: Step 7, Step 8
  - **Validation**: Run `go test ./test/integration/plain/...` to verify UDP integration test passes

- [ ] Step 10: Update CLI help text and documentation
  - **Task**: Document UDP protocol in user-facing documentation
  - **Files**:
    - `cmd/shared/shared.go`: Update `GetBaseDescription()` to mention UDP
      ```go
      "Specify transport like this: tcp://127.0.0.1:123 (supports tcp|ws|wss|udp)"
      ```
    - `docs/USAGE.md`: Add UDP transport examples
      ```markdown
      ### Plain UDP with KCP
      ```bash
      goncat master listen 'udp://*:12345' --exec /bin/sh
      goncat slave connect udp://192.168.1.100:12345
      ```
      
      **Best for**: Situations where UDP is available but TCP is blocked, or for testing UDP-based protocols.
      **Note**: Uses KCP protocol for reliable, ordered delivery over UDP.
      ```
    - `docs/ARCHITECTURE.md`: Document UDP transport implementation
      ```markdown
      - `udp/`: KCP over UDP connections
        - Uses `github.com/xtaci/kcp-go` for reliable delivery
        - Creates raw UDP PacketConn, then upgrades to KCP session
        - Provides net.Conn interface compatible with rest of system
      ```
    - `README.md`: Mention UDP in quick feature list
      ```markdown
      Supported protocols include `tcp`, `ws`, `wss`, and `udp` (UDP with KCP for reliability).
      ```
  - **Dependencies**: All previous steps
  - **Validation**: Review documentation changes for accuracy

- [ ] Step 11: Build and manual testing
  - **Task**: Build binaries and manually test UDP transport
  - **Files**: None (compilation and manual testing)
  - **Commands**:
    ```bash
    # Build Linux binary
    make build-linux
    
    # Terminal 1: Start UDP listener
    ./dist/goncat.elf slave listen 'udp://*:12345'
    
    # Terminal 2: Connect and get shell
    ./dist/goncat.elf master connect udp://localhost:12345 --exec /bin/sh
    
    # Test bidirectional communication
    echo "hello from master"
    whoami
    exit
    ```
  - **Dependencies**: All previous steps
  - **Validation**: 
    - Binary builds successfully
    - UDP connection establishes
    - Bidirectional data flow works
    - Shell commands execute correctly
    - Clean exit on both sides

- [ ] Step 12: Run full test suite
  - **Task**: Verify all tests pass with UDP transport added
  - **Commands**:
    ```bash
    # Run linting
    make lint
    
    # Run unit tests
    make test-unit
    
    # Run integration tests
    make test-integration
    
    # Optionally run E2E tests (takes ~8-9 minutes)
    # Note: E2E tests don't need to be updated for UDP initially
    # as they test real binaries and UDP will work through entrypoint
    ```
  - **Dependencies**: All previous steps
  - **Validation**: All tests pass without errors

## Notes and Considerations

### KCP Configuration Parameters

The KCP library requires several configuration parameters:

1. **Block Cipher** (`BlockCrypt`): Set to `nil` for no encryption (TLS handles encryption in goncat)
2. **FEC Parameters** (`dataShards`, `parityShards`): Set to `0, 0` to disable Forward Error Correction initially for simplicity
3. **NoDelay Parameters**: `(nodelay, interval, resend, nc)` - Use `(1, 10, 2, 1)` for low-latency mode
4. **Stream Mode**: Enable with `SetStreamMode(true)` for TCP-like streaming behavior
5. **Window Size**: `SetWindowSize(1024, 1024)` for send and receive windows

These parameters can be tuned later if performance issues arise.

### Packet Connection vs Stream Connection

- **PacketConn**: Go's `net.PacketConn` interface for UDP - allows reading/writing packets
- **UDPSession**: KCP's wrapper that provides `net.Conn` interface (streaming) over PacketConn
- We create PacketConn ourselves, then let KCP wrap it into a streaming connection

### Mock Strategy for Testing

Two approaches for mocking:

1. **Real KCP in tests**: Use actual kcp-go library with in-memory PacketConn mocks
   - Pro: Tests real KCP behavior
   - Con: More complex, slower tests

2. **Simple net.Pipe() mocks**: Bypass KCP in tests, use direct net.Conn pairs
   - Pro: Simpler, faster tests
   - Con: Doesn't test KCP integration

Recommendation: Use approach #2 for integration tests (we're testing goncat logic, not KCP). The manual testing and potential future E2E tests will cover real KCP behavior.

### Error Handling

Similar to TCP listener, handle:
- Connection closed errors gracefully
- Timeout errors with retry backoff
- Handler panics with recovery
- Semaphore logic to limit concurrent connections

### Compatibility with Existing Features

UDP transport should work with:
- ✅ TLS encryption (`--ssl` flag)
- ✅ Mutual authentication (`--key` flag)
- ✅ PTY mode (`--pty` flag)
- ✅ Port forwarding (`-L`, `-R` flags)
- ✅ SOCKS proxy (`-D` flag)
- ✅ Session logging (`--log` flag)
- ✅ Cleanup (`--cleanup` flag)

All of these work at a higher layer than transport, so UDP integration should be transparent.

### Future Enhancements (Out of Scope)

- Configurable KCP parameters via CLI flags
- FEC (Forward Error Correction) configuration
- Custom block ciphers for KCP-level encryption
- UDP-specific optimizations (MTU tuning, congestion control)
- E2E tests specifically for UDP (initial implementation relies on manual testing)

## Review Checklist

Before considering implementation complete:

- [ ] All unit tests pass (`go test ./pkg/transport/udp/...`)
- [ ] Parser tests include UDP protocol (`go test ./cmd/shared/...`)
- [ ] Server and client tests pass (`go test ./pkg/server/...`, `go test ./pkg/client/...`)
- [ ] Integration test for UDP passes (`go test ./test/integration/plain/...`)
- [ ] Full test suite passes (`make test-unit`, `make test-integration`)
- [ ] Linting passes (`make lint`)
- [ ] Binary builds successfully (`make build-linux`)
- [ ] Manual testing confirms UDP connection works
- [ ] Documentation updated (USAGE.md, ARCHITECTURE.md, README.md)
- [ ] Code follows existing style and patterns in the codebase
- [ ] No race conditions (`go test -race ./pkg/transport/udp/...`)

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| KCP library compatibility issues | Use stable version (v5), test thoroughly |
| Performance degradation vs TCP | Use recommended KCP parameters, tune if needed |
| Mock complexity for testing | Use simple net.Pipe() mocks in integration tests |
| Breaking existing functionality | Run full test suite, verify TCP/WS/WSS still work |
| KCP session management edge cases | Follow KCP examples, handle errors like TCP listener |

## Success Criteria

Implementation is successful when:

1. ✅ UDP protocol can be specified in CLI: `udp://host:port`
2. ✅ Master can listen on UDP and slave can connect
3. ✅ Slave can listen on UDP and master can connect
4. ✅ Bidirectional data flow works correctly
5. ✅ Shell execution works over UDP (PTY and non-PTY)
6. ✅ TLS encryption works with UDP transport
7. ✅ All existing tests continue to pass
8. ✅ New unit and integration tests pass
9. ✅ Documentation is updated
10. ✅ Code quality matches existing transport implementations
