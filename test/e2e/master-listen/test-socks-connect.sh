#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 10

# Start goncat master with SOCKS proxy
# -D 127.0.0.1:1080 means:
# - Listen on 127.0.0.1:1080 with a SOCKS5 proxy server
# - Tunnel connections through the slave side
spawn /opt/dist/goncat.elf master listen $transport://:8080 -D 127.0.0.1:1080

Expect::client_connected

# Give the SOCKS proxy a moment to be ready
sleep 1

# Use shared helper to verify SOCKS CONNECT functionality
Expect::check_socks_connect localhost 1080 slave-companion 9000 "test message" 5

# Clean up the master connection
Expect::close_and_wait
