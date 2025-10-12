#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 15

# Start goncat master with both remote port forwarding AND exec mode
# -R 7000:master-companion:9000 means:
# - Bind port 7000 on the slave side
# - Forward connections to master-companion:9000 (on master side)
# --exec sh allows us to run commands on the slave
spawn /opt/dist/goncat.elf master listen $transport://:8080 -R 7000:master-companion:9000 --exec sh

Expect::client_connected

# Give the remote port forwarding a moment to be ready
sleep 3

# Now execute a command on the slave that connects to the forwarded port
# We need to ensure the shell and socat command complete before the connection closes
send "(printf 'test message' | socat - TCP:localhost:7000; sleep 1) & wait\r"

# Wait for the response from master-companion
expect {
    "*master-companion says: test message*" {
        puts "\n✓ Remote port forwarding test successful!"
    }
    timeout {
        puts stderr "\n✗ Timeout waiting for response from master-companion"
        exit 1
    }
    eof {
        puts stderr "\n✗ Unexpected EOF while waiting for response"
        exit 1
    }
}

# Wait a moment for any remaining output
sleep 1

# Exit the shell
send "exit\r"

# Clean up
Expect::close_and_wait
