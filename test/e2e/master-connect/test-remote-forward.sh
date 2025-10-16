#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 15

# Start goncat master with both remote port forwarding AND exec mode
# -R 7000:master-companion:9000 means:
# - Bind port 7000 on the slave side
# - Forward connections to master-companion:9000 (on master side)
# --exec sh allows us to run commands on the slave
spawn /opt/dist/goncat.elf master connect $transport://slave:8080 -R 7000:master-companion:9000 --exec sh

Expect::server_connected

# Give the remote port forwarding a moment to be ready
sleep 1

# Use shared helper to perform the remote-forward check
Expect::check_remote_forward 7000 master-companion "test message" 5

# Clean up
Expect::close_and_wait
