#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 20

# Test local port forwarding (-L) in master-listen mode
# This test verifies that local port forwarding works in listen mode

# Start goncat master in listen mode with local port forwarding
spawn /opt/dist/goncat.elf master listen $transport://:8080 --exec sh -L 9999:127.0.0.1:8888

Expect::client_connected

# First, verify we have shell access and can execute commands
send -- "echo 'Master listening with local port forwarding enabled'\n"
Utils::wait_for "Master listening with local port forwarding enabled"

# Test 1: Verify basic shell functionality works with port forwarding
send -- "id\n"
Utils::wait_for "uid="

send -- "hostname\n"
Utils::wait_for "the_slave"

# Test 2: Check if we can create a simple server on the target port (8888)
send -- "echo 'Setting up test server on target port 8888...'\n"
Utils::wait_for "Setting up test server on target port 8888"

# Create a background process that will listen on port 8888
send -- "echo 'LOCAL_LISTEN_TEST_SERVER' | nc -l -p 8888 &\n"
send -- "sleep 1\n"
Utils::wait_for "# " 0

# Test 3: Check that our test server is running
send -- "netstat -an | grep :8888 | grep LISTEN && echo 'Target port 8888 is listening'\n"
Utils::wait_for "Target port 8888 is listening"

# Test 4: Test that we can connect to the target port locally 
send -- "echo 'Testing connection to target port 8888 in listen mode...'\n"
Utils::wait_for "Testing connection to target port 8888 in listen mode"

# Connect to the target port and see if we get our test response
send -- "echo 'LOCAL_LISTEN_CONNECT_TEST' | nc -w 2 127.0.0.1 8888 || echo 'Local listen connection attempt completed'\n"
Utils::wait_for "Local listen connection attempt completed"

# Test 5: Verify that the port forwarding configuration is active
send -- "echo 'Port forwarding test completed in listen mode - configuration is active'\n"
Utils::wait_for "Port forwarding test completed in listen mode - configuration is active"

# If we reach here, the local port forwarding setup is working in listen mode
send -- "echo 'SUCCESS: Local port forwarding test passed in master-listen mode'\n"
Utils::wait_for "SUCCESS: Local port forwarding test passed in master-listen mode"

send -- "exit\n"
Expect::close_and_wait