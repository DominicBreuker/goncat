#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 5

spawn /opt/dist/goncat.elf master connect $transport://slave:8080 --exec sh

Expect::server_connected
Expect::shell_access_works
Expect::exit_with_eof

wait
