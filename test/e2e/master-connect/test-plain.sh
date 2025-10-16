#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 5

spawn /opt/dist/goncat.elf master connect $transport://slave:8080

Expect::server_connected

Expect::close_and_wait
