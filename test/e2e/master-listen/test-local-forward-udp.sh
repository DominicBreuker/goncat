#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 10

# Start goncat master with UDP local port forwarding
# -L U:7001:slave-companion:9001 means:
# - Listen on local UDP port 7001 (on master side)
# - Forward datagrams to slave-companion:9001 (via slave side)
# Note: in master-listen mode, the slave can reach slave-companion
spawn /opt/dist/goncat.elf master listen $transport://:8080 -L U:7001:slave-companion:9001

Expect::client_connected

# Give the port forwarding a moment to be ready
sleep 1

# Verify UDP local forwarding using the shared helper
Expect::check_local_forward_udp 7001 slave-companion "test udp message" 5

# Clean up the master connection
Expect::close_and_wait
