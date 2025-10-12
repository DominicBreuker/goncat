#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 10

# Start goncat master with local port forwarding
# -L 7000:client-companion:9000 means:
# - Listen on local port 7000 (on master side)
# - Forward connections to client-companion:9000 (via slave side)
# Note: in master-connect mode, the slave is listening and master connects to it
spawn /opt/dist/goncat.elf master connect $transport://server:8080 -L 7000:client-companion:9000

Expect::server_connected

# Give the port forwarding a moment to be ready
sleep 1

# Now test the port forwarding by connecting to localhost:7000
# This should forward through the slave to client-companion:9000
set spawn_id_master $spawn_id

spawn nc localhost 7000
set spawn_id_client $spawn_id

# Send a test message through the forwarded port
send "test message\r"

# Wait for the response from client-companion
expect {
    "*client-companion says: test message*" {
        puts "\n✓ Local port forwarding test successful!"
    }
    timeout {
        puts stderr "\n✗ Timeout waiting for response from client-companion"
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
