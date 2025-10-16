#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 15

# Start goncat master with SOCKS proxy
# -D 127.0.0.1:1080 enables SOCKS5 proxy including UDP ASSOCIATE support
spawn /opt/dist/goncat.elf master listen $transport://:8080 -D 127.0.0.1:1080

Expect::client_connected

# Give the SOCKS proxy a moment to be ready
sleep 1

# Use shared helper to run the UDP ASSOCIATE check
Expect::check_socks_udp_associate localhost 1080 slave-companion 9001 10

# Clean up the master connection
Expect::close_and_wait
