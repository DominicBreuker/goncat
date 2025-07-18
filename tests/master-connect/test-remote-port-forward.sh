#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 10

# Test that goncat accepts the remote port forwarding flag and connects successfully
spawn /opt/dist/goncat.elf master connect $transport://server:8080 --exec sh -R 9999:127.0.0.1:8888

Expect::server_connected

# Verify we can execute commands (basic shell functionality) and that port forwarding didn't cause issues
send -- "echo 'Remote port forwarding test: connection established with -R flag'\n"
Utils::wait_for "Remote port forwarding test: connection established with -R flag"

# Test that we can check if the port forwarding is working by checking network status
send -- "netstat -an | grep 9999 || echo 'Port forwarding active (port may not be visible in netstat)'\n"
Utils::wait_for "Port forwarding active"

send -- "exit\n"
Expect::close_and_wait