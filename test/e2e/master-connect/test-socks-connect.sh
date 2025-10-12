#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 10

# Start goncat master with SOCKS proxy
# -D 127.0.0.1:1080 means:
# - Listen on 127.0.0.1:1080 with a SOCKS5 proxy server
# - Tunnel connections through the slave side
spawn /opt/dist/goncat.elf master connect $transport://slave:8080 -D 127.0.0.1:1080

Expect::server_connected

# Give the SOCKS proxy a moment to be ready
sleep 1

# Now test the SOCKS proxy by using socat with socks4a to connect to slave-companion:9000
# socat will bind a local port and forward it through the SOCKS proxy
set spawn_id_master $spawn_id

# Use socat to connect through SOCKS proxy to slave-companion:9000
# SOCKS5-CONNECT format: SOCKS5-CONNECT:<proxy-host>:<proxy-port>:<target-host>:<target-port>
spawn socat - SOCKS5-CONNECT:localhost:1080:slave-companion:9000
set spawn_id_client $spawn_id

# Send a test message through the SOCKS proxy
send "test message\r"

# Wait for the response from slave-companion
expect {
    "*slave-companion says: test message*" {
        puts "\n✓ SOCKS proxy test successful!"
    }
    timeout {
        puts stderr "\n✗ Timeout waiting for response from slave-companion"
        exit 1
    }
    eof {
        puts stderr "\n✗ Unexpected EOF while waiting for response"
        exit 1
    }
}

# Clean up the socat connection
close -i $spawn_id_client
wait -i $spawn_id_client

# Clean up the master connection
set spawn_id $spawn_id_master
Expect::close_and_wait
