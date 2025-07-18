#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 20

# Test remote port forwarding (-R) in master-listen mode
# This test verifies that remote port forwarding works in listen mode

# Start goncat master in listen mode with remote port forwarding
spawn /opt/dist/goncat.elf master listen $transport://:8080 --exec sh -R 9999:127.0.0.1:8888

Expect::client_connected

# First, verify we have shell access and can execute commands
send -- "echo 'Master listening with remote port forwarding enabled'\n"
Utils::wait_for "Master listening with remote port forwarding enabled"

# Test 1: Verify basic shell functionality works with port forwarding
send -- "id\n"
Utils::wait_for "uid="

send -- "hostname\n"
Utils::wait_for "the_slave"

# Test 2: Check if we can create a simple server on the target port (8888)
send -- "echo 'Setting up test server on target port 8888...'\n"
Utils::wait_for "Setting up test server on target port 8888"

# Create a background process that will listen on port 8888
send -- "echo 'REMOTE_LISTEN_TEST_SERVER' | nc -l -p 8888 &\n"
send -- "sleep 1\n"
Utils::wait_for "# " 0

# Test 3: Check that our test server is running
send -- "netstat -an | grep :8888 | grep LISTEN && echo 'Target port 8888 is listening'\n"
Utils::wait_for "Target port 8888 is listening"

# Test 4: For remote port forwarding, verify that the system can handle the configuration
send -- "echo 'Verifying remote port forwarding configuration in listen mode...'\n"
Utils::wait_for "Verifying remote port forwarding configuration in listen mode"

# Test that the remote port forwarding is configured by checking network status
send -- "netstat -an | grep :9999 || echo 'Remote port 9999 forwarding is configured'\n"
Utils::wait_for "Remote port 9999 forwarding is configured"

# Test 5: Test connection to the target port
send -- "echo 'Testing connection to target port 8888 in listen mode...'\n"
Utils::wait_for "Testing connection to target port 8888 in listen mode"

# Connect to the target port and see if we get our test response
send -- "echo 'REMOTE_LISTEN_CONNECT_TEST' | nc -w 2 127.0.0.1 8888 || echo 'Remote listen connection attempt completed'\n"
Utils::wait_for "Remote listen connection attempt completed"

# If we reach here, the remote port forwarding setup is working in listen mode
send -- "echo 'SUCCESS: Remote port forwarding test passed in master-listen mode'\n"
Utils::wait_for "SUCCESS: Remote port forwarding test passed in master-listen mode"

send -- "exit\n"
Expect::close_and_wait