#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 5

spawn /opt/dist/goncat.elf master listen $transport://:8080

Expect::client_connected

Expect::close_and_wait
