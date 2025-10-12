# Local Port Forwarding E2E Tests

This document describes the local port forwarding e2e tests added to the goncat test suite.

## Overview

The e2e test setup has been extended to support testing local port forwarding functionality. This allows verification that goncat can forward connections from the master side through the slave to remote servers that are only accessible from the slave side.

## Architecture

### Network Topology

Each docker-compose configuration now includes four services:

1. **server** / **client** - The main goncat services (master or slave depending on test mode)
2. **server-companion** - A simple echo server accessible only from `server` via `server_private` network
3. **client-companion** - A simple echo server accessible only from `client` via `client_private` network

### Networks

- **common**: Bridge network connecting server and client (allows them to communicate)
- **server_private**: Internal bridge network connecting server and server-companion only
- **client_private**: Internal bridge network connecting client and client-companion only

The `internal: true` flag on private networks prevents cross-access between companions.

### Echo Servers

Each companion service runs a simple echo server using netcat (nc):

```bash
while true; do 
  nc -l -p 9000 -e sh -c "read line; echo <companion-name> says: \$line"
done
```

This creates a TCP server on port 9000 that:
- Reads one line of input
- Echoes it back with a prefix identifying the companion
- Loops to accept the next connection

## Test Scenarios

### Master-Listen Mode (test/e2e/master-listen/test-local-forward.sh)

Tests local port forwarding when master is listening for slave connections:

1. Master starts: `goncat master listen tcp://:8080 -L 7000:server-companion:9000`
2. Slave connects to master
3. Test connects to `localhost:7000` on master side
4. Connection is forwarded through slave to `server-companion:9000`
5. Test verifies response contains "server-companion says:"

This demonstrates that the master can access a server only reachable from the slave's network.

### Master-Connect Mode (test/e2e/master-connect/test-local-forward.sh)

Tests local port forwarding when master connects to listening slave:

1. Master starts: `goncat master connect tcp://server:8080 -L 7000:client-companion:9000`
2. Slave is listening on port 8080
3. Test connects to `localhost:7000` on master side
4. Connection is forwarded through slave to `client-companion:9000`
5. Test verifies response contains "client-companion says:"

Note: In this mode, "client" is where the master runs, and the forwarding reaches `client-companion` through the slave.

## Running the Tests

### Prerequisites

- Docker and Docker Compose installed
- Network access to Alpine package repositories (for building test images)
- Built goncat Linux binary in `dist/goncat.elf`

### Build Binary

```bash
make build-linux
```

### Run All E2E Tests

```bash
make test-integration
```

This runs:
- Existing plain connection tests
- Existing exec mode tests (with and without PTY)
- New local port forwarding tests

### Run Specific Test

```bash
# Test slave-listen (master-connect tests)
TRANSPORT='tcp' TEST_SET='master-connect' docker compose -f test/e2e/docker-compose.slave-listen.yml up --exit-code-from client

# Test slave-connect (master-listen tests)
TRANSPORT='tcp' TEST_SET='master-listen' docker compose -f test/e2e/docker-compose.slave-connect.yml up --exit-code-from server
```

### Supported Transports

Tests run with three transport protocols:
- `tcp` - Plain TCP
- `ws` - WebSocket
- `wss` - WebSocket with TLS

## Implementation Details

### Test Scripts

Both test scripts follow this pattern:

1. Start goncat with `-L` flag for local port forwarding
2. Wait for connection establishment
3. Sleep 1 second to ensure port forwarding is ready
4. Spawn nc to connect to forwarded port
5. Send test message
6. Verify response contains expected companion prefix
7. Clean up connections

### Expect Library Usage

Tests use the expect library in `test/e2e/lib.tcl`:
- `Expect::client_connected` - Wait for "New * connection from" message
- `Expect::server_connected` - Wait for "Connecting to" message
- `Expect::close_and_wait` - Clean shutdown

### Key Testing Points

1. **Network Isolation**: Companions are only accessible via their private networks
2. **Port Forwarding**: Master accesses companions only through forwarded ports
3. **Bidirectional Data Flow**: Request and response both traverse the tunnel
4. **Message Integrity**: Response includes correct companion identifier

## Troubleshooting

### Build Failures

If you see "Permission denied" errors when building the Docker image:
- This is a network connectivity issue accessing Alpine package repositories
- The issue should not occur in CI environments (GitHub Actions)
- The base Alpine image includes nc (netcat) which is all we need besides expect

### Test Timeouts

If tests timeout waiting for companion responses:
- Check that companion services are healthy: `docker compose ps`
- Verify companions are listening: `docker compose exec server-companion netstat -an | grep 9000`
- Check goncat logs for port forwarding errors

### Connection Failures

If port forwarding doesn't work:
- Verify the companion is reachable from the slave: `docker compose exec server nc -v server-companion 9000`
- Check that master cannot reach companion directly (should fail): `docker compose exec client nc -v server-companion 9000`

## Future Enhancements

Potential improvements to these tests:

1. Test multiple simultaneous forwarded connections
2. Test bidirectional forwarding (both -L and -R in same session)
3. Test forwarding to multiple different destinations
4. Test error cases (unreachable destination, port conflicts)
5. Verify that companions are truly isolated (negative test)
