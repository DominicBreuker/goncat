#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 15

# Start goncat master with UDP remote port forwarding AND exec mode in connect mode
# -R U:7001:master-companion:9001 means:
# - Bind UDP port 7001 on the slave side
# - Forward datagrams to master-companion:9001 (on master side)
# --exec sh allows us to run commands on the slave
spawn /opt/dist/goncat.elf master connect $transport://slave:8080 -R U:7001:master-companion:9001 --exec sh

Expect::server_connected

# Give the remote port forwarding a moment to be ready
sleep 1

# Use shared helper to perform the remote-forward check for UDP
Expect::check_remote_forward_udp 7001 master-companion "test udp message" 5

# Clean up
Expect::close_and_wait
