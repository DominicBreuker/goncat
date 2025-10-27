# UDP Port Forwarding Examples

This document demonstrates how to use UDP port forwarding with goncat.

## Syntax

UDP port forwarding uses the same `-L` (local) and `-R` (remote) flags as TCP, but with an optional `U:` or `u:` prefix:

```bash
# Local UDP port forwarding
-L U:8080:target-host:9000

# Remote UDP port forwarding
-R U:8080:target-host:9000

# TCP forwarding (for comparison)
-L 8080:target-host:9000        # Implicit TCP
-L T:8080:target-host:9000      # Explicit TCP
```

The protocol prefix is case-insensitive (`U:` or `u:`, `T:` or `t:`).

## Example: UDP Echo Server

### Setup

1. Start a UDP echo server on the slave side:
   ```bash
   # Simple Python UDP echo server
   python3 -c '
   import socket
   s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
   s.bind(("127.0.0.1", 9000))
   print("UDP Echo Server listening on 127.0.0.1:9000")
   while True:
       data, addr = s.recvfrom(4096)
       print(f"Received from {addr}: {data.decode()}")
       s.sendto(b"ECHO: " + data, addr)
   '
   ```

2. Start goncat slave:
   ```bash
   goncat slave listen 'tcp://*:12345'
   ```

3. Start goncat master with UDP local port forwarding:
   ```bash
   goncat master connect 'tcp://slave-host:12345' -L U:127.0.0.1:8000:127.0.0.1:9000
   ```

4. Test with a UDP client:
   ```bash
   # Using netcat
   echo "Hello UDP!" | nc -u 127.0.0.1 8000

   # Using Python
   python3 -c '
   import socket
   s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
   s.sendto(b"Hello UDP!", ("127.0.0.1", 8000))
   data, addr = s.recvfrom(4096)
   print(f"Response: {data.decode()}")
   '
   ```

## Example: DNS Forwarding

Forward DNS queries through the tunnel:

```bash
# On master side, forward local UDP 5353 to slave's DNS server (port 53)
goncat master connect 'tcp://slave-host:12345' -L U:127.0.0.1:5353:8.8.8.8:53

# Test DNS query
dig @127.0.0.1 -p 5353 example.com
```

## Example: Mixed TCP and UDP Forwarding

You can forward both TCP and UDP ports simultaneously:

```bash
goncat master connect 'tcp://slave-host:12345' \
  -L T:8080:web-server:80 \
  -L U:5353:dns-server:53
```

## Remote Port Forwarding

Remote UDP port forwarding works similarly to local forwarding:

```bash
# On slave side, port 8000 will forward to master's 127.0.0.1:9000
goncat master listen 'tcp://*:12345' -R U:127.0.0.1:8000:127.0.0.1:9000

# Slave connects
goncat slave connect 'tcp://master-host:12345'

# Now UDP clients on the slave side can connect to 127.0.0.1:8000
# and traffic will be forwarded to master's 127.0.0.1:9000
```

## Important Notes

### UDP Session Management

- Each unique client address (IP:port combination) gets its own session
- Sessions are automatically cleaned up after 60 seconds of inactivity (or the value specified by `--timeout`)
- Multiple UDP clients can communicate through the same forwarded port simultaneously

### Differences from TCP

- UDP is connectionless - datagrams are forwarded individually
- No connection establishment or teardown
- Responses are routed back to the correct client based on the source address
- Packet ordering is not guaranteed (though the tunnel itself is reliable)

### Timeout Configuration

Use the `--timeout` flag to control UDP session lifetime:

```bash
# Sessions expire after 30 seconds of inactivity
goncat master connect 'tcp://slave:12345' -L U:8000:target:9000 --timeout 30s
```

## Troubleshooting

### No response received

1. Verify the UDP service is running on the destination
2. Check that the destination accepts UDP packets from 127.0.0.1
3. Ensure no firewall is blocking UDP traffic
4. Try with `--timeout` set to a longer value

### Testing UDP forwarding

Use these simple commands to test:

```bash
# Test with netcat (if available)
echo "test" | nc -u -w1 127.0.0.1 8000

# Test with Python
python3 -c 'import socket; s=socket.socket(socket.AF_INET, socket.SOCK_DGRAM); s.settimeout(5); s.sendto(b"test", ("127.0.0.1", 8000)); print(s.recvfrom(4096)[0].decode())'
```
