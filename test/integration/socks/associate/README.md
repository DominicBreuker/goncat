# SOCKS ASSOCIATE Integration Tests

This directory contains comprehensive integration tests for the SOCKS5 UDP ASSOCIATE feature (UDP proxy).

## Test Files

### `single_client_test.go`
Tests basic SOCKS5 UDP ASSOCIATE functionality with a single client:
- Creates a SOCKS5 UDP proxy
- Sends a UDP datagram through the proxy
- Verifies the response is correctly routed back

### `multiple_clients_test.go`
Tests concurrent client support:
- Creates multiple SOCKS5 clients simultaneously
- Each client gets its own UDP relay
- Verifies all clients can send/receive independently
- Tests that the SOCKS proxy correctly handles multiple concurrent UDP relays

### `helpers.go`
Shared test infrastructure:
- `TestSetup`: Sets up mock networks, master/slave processes, and destination server
- `SOCKSClient`: Manages SOCKS5 client connections (TCP control + UDP data)
- `CreateSOCKSClient()`: Performs SOCKS5 handshake and creates UDP relay
- `SendUDPDatagram()`: Sends SOCKS5 UDP datagram and receives response

## Running Tests

Run all SOCKS ASSOCIATE tests:
```bash
go test -v ./test/integration/socks/associate/
```

Run specific test:
```bash
go test -v -run TestSingleClient ./test/integration/socks/associate/
go test -v -run TestMultipleClients ./test/integration/socks/associate/
```

## Test Architecture

The tests use mocked networks (TCP and UDP) to simulate the complete SOCKS proxy flow:

```
[Client] --UDP--> [Master Relay] --TCP--> [Slave Relay] --UDP--> [Destination Server]
                                                                           |
[Client] <--UDP-- [Master Relay] <--TCP-- [Slave Relay] <--UDP-- [Response]
```

Each test:
1. Sets up master (with SOCKS proxy) and slave processes
2. Creates mock destination server on slave side
3. Creates SOCKS5 client(s) that connect to the proxy
4. Performs UDP ASSOCIATE handshake
5. Sends UDP datagrams through the proxy
6. Verifies responses are correctly routed back

## Key Features Tested

- **SOCKS5 handshake**: Method selection and UDP ASSOCIATE request/response
- **UDP relay creation**: Each client gets its own UDP relay on a unique port
- **Packet routing**: UDP packets are correctly wrapped/unwrapped with SOCKS5 headers
- **Bidirectional flow**: Both request and response packets flow correctly
- **Concurrent clients**: Multiple clients can use the proxy simultaneously
- **Address tracking**: Source addresses are correctly preserved in responses
