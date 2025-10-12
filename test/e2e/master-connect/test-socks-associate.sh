#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 15

# Start goncat master with SOCKS proxy
# -D 127.0.0.1:1080 enables SOCKS5 proxy including UDP ASSOCIATE support
spawn /opt/dist/goncat.elf master connect $transport://slave:8080 -D 127.0.0.1:1080

Expect::server_connected

# Give the SOCKS proxy a moment to be ready
sleep 2

# Save the spawn_id for the master connection
set spawn_id_master $spawn_id

# Now test UDP ASSOCIATE by using the Python SOCKS5 UDP test client
# This will send a UDP datagram through the SOCKS proxy to slave-companion:9001 (UDP echo server)
spawn python3 /opt/socks5-udp-test.py localhost 1080 slave-companion 9001
set spawn_id_test $spawn_id

# Enable logging to see all output
log_user 1

# Wait for the test result
expect {
    -i $spawn_id_test
    "*✓ UDP ASSOCIATE test successful!*" {
        puts "\n✓ SOCKS UDP ASSOCIATE test successful!"
    }
    "*✗ Error:*" {
        puts stderr "\n✗ SOCKS UDP ASSOCIATE test failed"
        exit 1
    }
    timeout {
        puts stderr "\n✗ Timeout waiting for UDP ASSOCIATE test result"
        exit 1
    }
    eof {
        puts stderr "\n✗ Unexpected EOF from test client"
        exit 1
    }
}

# Clean up the test client
close -i $spawn_id_test
wait -i $spawn_id_test

# Clean up the master connection
set spawn_id $spawn_id_master
Expect::close_and_wait
