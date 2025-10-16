#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 10

# Start goncat master with local port forwarding
# -L 7000:slave-companion:9000 means:
# - Listen on local port 7000 (on master side)
# - Forward connections to slave-companion:9000 (via slave side)
# Note: in master-connect mode, the slave can reach slave-companion
spawn /opt/dist/goncat.elf master connect $transport://slave:8080 -L 7000:slave-companion:9000

Expect::server_connected

# Give the port forwarding a moment to be ready
sleep 1

# Verify local forwarding using the shared helper
Expect::check_local_forward 7000 slave-companion "test message" 5

# Clean up the master connection
Expect::close_and_wait
