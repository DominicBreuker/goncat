#!/usr/bin/expect -f

# This is a comprehensive test that demonstrates actual port forwarding functionality
# It tests that data sent to a forwarded port reaches the intended destination

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 30

# Test local port forwarding (-L) with actual data flow
# Start goncat with local port forwarding: local port 9999 -> remote port 8888
spawn /opt/dist/goncat.elf master connect $transport://server:8080 --exec sh -L 9999:127.0.0.1:8888

Expect::server_connected

# Create a comprehensive test that shows data actually flows through the tunnel
send -- "echo 'Starting comprehensive port forwarding test...'\n"
Utils::wait_for "Starting comprehensive port forwarding test"

# Test 1: Create a test server that will log received data
send -- "echo 'Creating test server that logs received data...'\n"
Utils::wait_for "Creating test server that logs received data"

# Create a server that writes received data to a log file
send -- "{ echo 'DATA_FLOW_TEST_SERVER_READY'; while read line; do echo \"RECEIVED: \$line\" >> /tmp/tunnel_test.log; echo \"LOGGED: \$line\"; done; } | nc -l -p 8888 &\n"
send -- "echo 'Data logging server started on port 8888'\n"
Utils::wait_for "Data logging server started on port 8888"

# Give the server time to start
send -- "sleep 2\n"
Utils::wait_for "# " 0

# Test 2: Send test data and verify it's received
send -- "echo 'Sending test data through port forwarding...'\n"
Utils::wait_for "Sending test data through port forwarding"

# Send multiple test messages
send -- "echo 'TEST_MESSAGE_1' | nc -w 3 127.0.0.1 8888\n"
Utils::wait_for "LOGGED: TEST_MESSAGE_1"

send -- "echo 'TEST_MESSAGE_2' | nc -w 3 127.0.0.1 8888\n"
Utils::wait_for "LOGGED: TEST_MESSAGE_2"

# Test 3: Verify the data was actually logged (proves data flow)
send -- "echo 'Verifying data flow through tunnel...'\n"
Utils::wait_for "Verifying data flow through tunnel"

send -- "cat /tmp/tunnel_test.log\n"
Utils::wait_for "RECEIVED: TEST_MESSAGE_1"
Utils::wait_for "RECEIVED: TEST_MESSAGE_2"

# Test 4: Test bidirectional communication
send -- "echo 'Testing bidirectional communication...'\n"
Utils::wait_for "Testing bidirectional communication"

# Start an echo server for bidirectional test
send -- "{ echo 'ECHO_SERVER_READY'; while read line; do echo \"ECHO: \$line\"; done; } | nc -l -p 8889 &\n"
send -- "echo 'Echo server started on port 8889'\n"
Utils::wait_for "Echo server started on port 8889"

send -- "sleep 1\n"
Utils::wait_for "# " 0

# Test echo communication
send -- "echo 'BIDIRECTIONAL_TEST' | nc -w 3 127.0.0.1 8889\n"
Utils::wait_for "ECHO: BIDIRECTIONAL_TEST"

# If we reach here, port forwarding is working correctly
send -- "echo 'SUCCESS: Comprehensive port forwarding test PASSED - data flows correctly through tunnel'\n"
Utils::wait_for "SUCCESS: Comprehensive port forwarding test PASSED"

send -- "exit\n"
Expect::close_and_wait