# SOCKS ASSOCIATE Integration Test - TODO

## Overview

This document describes the current state of the SOCKS ASSOCIATE integration test and what remains to be debugged.

## What's Been Completed

### 1. Mock UDP Network (`mocks/mockudp.go`)
- ✅ Full UDP network mocking implementation similar to `MockTCPNetwork`
- ✅ Supports `ListenUDP` and `ListenPacket` functions
- ✅ Handles ephemeral port allocation (port 0 → 40000-50000 range)
- ✅ Packet routing between listeners via in-memory channels
- ✅ `WaitForListener` support for synchronization
- ✅ Tested independently and works correctly

### 2. Dependency Injection
- ✅ Added `UDPListenerFunc` and `PacketListenerFunc` to `pkg/config/dependencies.go`
- ✅ Added `GetUDPListenerFunc` and `GetPacketListenerFunc` helper functions
- ✅ Updated `pkg/handler/socks/master/relay.go`:
  - Changed `Conn` field from `*net.UDPConn` to `net.PacketConn` to support mocking
  - Modified `NewUDPRelay` to accept `*config.Dependencies` parameter
  - Updated `LocalToRemote` to use `ReadFrom` instead of `ReadFromUDPAddrPort`
  - Created helper function `writeUDPRequest` to work with `PacketConn`
- ✅ Updated `pkg/handler/socks/slave/associate.go`:
  - Modified `NewUDPRelay` to accept `*config.Dependencies` parameter
  - Uses `GetPacketListenerFunc` for UDP binding
- ✅ Updated calling code in:
  - `pkg/handler/socks/master/associate.go` to pass dependencies
  - `pkg/handler/slave/socks.go` to pass dependencies
- ✅ Updated all unit tests to pass `nil` for dependencies (uses defaults)

### 3. Integration Test Structure
- ✅ Created `test/integration/socks_associate_test.go`
- ✅ Follows the same pattern as `socks_connect_test.go`
- ✅ Test setup:
  - Mock TCP network for control connection (master-slave)
  - Mock UDP network for data plane (client-relay-destination)
  - UDP destination server at 127.0.0.1:8080
  - SOCKS proxy at 127.0.0.1:1080
  - Complete SOCKS5 handshake (method selection + ASSOCIATE request)
  - UDP datagram with SOCKS5 header format
  - Response verification

## Current Issue

The test is currently **skipped** because UDP packets are not flowing through the complete relay chain. Symptoms:

1. ✅ Master starts and listens on TCP 12345
2. ✅ Slave connects to master
3. ✅ SOCKS proxy binds on TCP 1080
4. ✅ Client connects to SOCKS proxy
5. ✅ ASSOCIATE handshake completes successfully
6. ✅ Master UDP relay binds on port 40000
7. ✅ Client sends UDP datagram to relay at 127.0.0.1:40000
8. ❌ No response received - test times out waiting for UDP response

### Debug Output
```
[+] Listening on 127.0.0.1:12345
[+] Connecting to 127.0.0.1:12345
[+] New TCP connection from 127.0.0.1:XXXXX
UDP relay address: 127.0.0.1:40000
Sending UDP datagram to relay at 127.0.0.1:40000
UDP datagram sent successfully
Waiting for UDP response...
[!] Error: UDP Relay: context deadline exceeded
[!] Error: SOCKS UDP Relay: reading from local conn: connection closed
```

## Expected Packet Flow

1. **Client → Master Relay (UDP)**
   - Client sends SOCKS5 UDP datagram to 127.0.0.1:40000
   - Master relay's `LocalToRemote` reads packet via `ReadFrom`
   - Parses SOCKS5 header, extracts destination (127.0.0.1:8080)

2. **Master → Slave (TCP)**
   - Master relay sends `msg.SocksDatagram` via TCP control channel
   - Uses gob encoder over multiplexed connection

3. **Slave → Destination (UDP)**
   - Slave relay's `remoteToLocal` receives from TCP
   - Slave relay sends raw UDP to 127.0.0.1:8080
   - Slave's local address is 0.0.0.0:40001 (ephemeral)

4. **Destination → Slave (UDP)**
   - Destination receives UDP, responds to sender (0.0.0.0:40001)
   - Slave relay's `localToRemote` reads response

5. **Slave → Master (TCP)**
   - Slave relay sends response via TCP control channel

6. **Master → Client (UDP)**
   - Master relay's `RemoteToLocal` receives from TCP
   - Sends SOCKS5 UDP datagram to client

## Possible Root Causes

### Theory 1: Goroutine Initialization Timing
- The relay goroutines (`LocalToRemote` and `RemoteToLocal`) may not be fully started before the client sends data
- Evidence: Adding sleep delays doesn't help
- Counter-evidence: TCP-based tests work without delays

### Theory 2: Mock Network Address Resolution
- Slave binds to "0.0.0.0:0" which becomes "0.0.0.0:40001" in mock
- Destination server responds to "0.0.0.0:40001"
- Mock network might not properly route packets to "0.0.0.0" addresses
- Test shows `WriteTo` with "0.0.0.0:XXXXX" should work, but needs verification in full flow

### Theory 3: Packet Format Issues
- SOCKS5 UDP datagram format might be incorrect
- Evidence: Format follows RFC 1928 specification
- Counter-evidence: `ReadUDPDatagram` parsing should fail with clear error

### Theory 4: Channel/Context Issues
- UDP relay creation on slave side might be failing silently
- Multiplexed channel creation might be blocking or failing
- Evidence: Similar pattern works for SOCKS CONNECT
- Counter-evidence: No error messages from relay creation

## Debugging Steps to Try

### 1. Add Extensive Logging
Add debug logging to:
- `pkg/handler/socks/master/relay.go` - `LocalToRemote` and `RemoteToLocal`
- `pkg/handler/socks/slave/associate.go` - `localToRemote` and `remoteToLocal`
- Track packet flow at each stage

### 2. Test Simplified Scenarios
Create minimal test cases:
- Test A: Client → Master Relay (verify master receives UDP)
- Test B: Master → Slave → Destination (verify destination receives)
- Test C: Destination → Slave → Master (verify master receives from slave)
- Test D: Master → Client (verify client receives response)

### 3. Verify Mock Network Behavior
- Test UDP packets between "0.0.0.0:X" and "127.0.0.1:Y" addresses
- Verify packet source address is correctly preserved
- Check if "0.0.0.0" addresses are properly routed

### 4. Check Goroutine Status
- Add synchronization channels to verify goroutines start
- Use `runtime.NumGoroutine()` to verify goroutines are running
- Check for potential deadlocks with `pprof`

### 5. Compare with SOCKS CONNECT
- Review differences between TCP relay (working) and UDP relay (broken)
- Check if there are timing dependencies in UDP code that don't exist in TCP code

## Test File Location

`test/integration/socks_associate_test.go` - Currently contains:
- Full test implementation (currently skipped)
- Complete SOCKS5 handshake
- UDP datagram construction
- Response parsing
- Multiple iterations test (for stability verification)

## How to Re-enable Test

Remove the `t.Skip()` line in `TestSocksAssociate`:
```go
func TestSocksAssociate(t *testing.T) {
    // t.Skip("Test infrastructure in place but UDP packet flow needs debugging")
    // ... rest of test
}
```

## Notes

- All other integration tests pass
- Unit tests pass
- Code compiles and builds successfully
- No linting errors
- The infrastructure is solid - just need to fix the packet flow issue
