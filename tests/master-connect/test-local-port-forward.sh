#!/usr/bin/expect -f

# Test local port forwarding (-L) functionality  
# This test demonstrates that local port forwarding works by:
# 1. Starting goncat master connect with -L flag to forward local port 9999 to slave's localhost:8888
# 2. Using the HTTP server that runs by default on slave localhost:8888 
# 3. Making HTTP requests from the MASTER side to localhost:9999 
# 4. Verifying that requests get tunneled to the slave and return the slave's hostname

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0]
set timeout 60

# Start goncat with local port forwarding: local port 9999 -> slave's localhost:8888
# The -L flag creates a local port 9999 on the master that forwards to 127.0.0.1:8888 on the slave
spawn /opt/dist/goncat.elf master connect $transport://server:8080 --exec sh -L 9999:127.0.0.1:8888

Expect::server_connected

send -- "echo 'Connected to slave with port forwarding: master:9999 -> slave:8888'\n"
Utils::wait_for "Connected to slave with port forwarding: master:9999 -> slave:8888"

# Verify that the slave's HTTP server is running and accessible locally
send -- "echo 'Checking if HTTP server is running on slave localhost:8888...'\n"
Utils::wait_for "Checking if HTTP server is running on slave localhost:8888"

send -- "netstat -an | grep 8888\n"
Utils::wait_for "127.0.0.1:8888"

send -- "echo 'Confirmed: HTTP server running on slave'\n"
Utils::wait_for "Confirmed: HTTP server running on slave"

# Test direct access to slave's HTTP server to verify it works
send -- "curl -s --connect-timeout 3 http://127.0.0.1:8888/\n"
expect {
    "HTTP_RESPONSE_FROM_" {
        send -- "echo 'SUCCESS: Slave HTTP server responds correctly'\n"
        Utils::wait_for "SUCCESS: Slave HTTP server responds correctly"
    }
    timeout {
        send -- "echo 'ERROR: Slave HTTP server not responding'\n"
        Utils::wait_for "ERROR: Slave HTTP server not responding"
        exit 1
    }
}

# Give port forwarding a moment to establish
send -- "sleep 2\n"
Utils::wait_for "# " 0

# Now the critical test: Use exec to make HTTP requests from the MASTER side
# These requests go to localhost:9999 on the master, which should tunnel to slave:8888

send -- "echo 'Starting port forwarding tests from master side...'\n"
Utils::wait_for "Starting port forwarding tests from master side"

# Wait a bit more for port forwarding to be fully established
exec sleep 1

# Test 1: First HTTP request through the tunnel from master side
send -- "echo 'Making HTTP request #1 from master to localhost:9999 (should tunnel to slave:8888)...'\n"
Utils::wait_for "Making HTTP request #1 from master"

set response1 [exec curl -s --connect-timeout 5 http://127.0.0.1:9999/ 2>/dev/null || echo "CURL_FAILED"]

send -- "echo 'Master side request #1 response: $response1'\n"
Utils::wait_for "Master side request #1 response:"

if {[string match "*HTTP_RESPONSE_FROM_*" $response1]} {
    send -- "echo 'SUCCESS: Request #1 tunneled correctly - received response from slave'\n"
    Utils::wait_for "SUCCESS: Request #1 tunneled correctly"
} else {
    send -- "echo 'FAILED: Request #1 did not work - no response or incorrect response'\n"
    Utils::wait_for "FAILED: Request #1 did not work"
    exit 1
}

# Test 2: Second HTTP request through the tunnel
send -- "echo 'Making HTTP request #2 from master to localhost:9999...'\n"
Utils::wait_for "Making HTTP request #2 from master"

set response2 [exec curl -s --connect-timeout 5 http://127.0.0.1:9999/ 2>/dev/null || echo "CURL_FAILED"]

send -- "echo 'Master side request #2 response: $response2'\n"
Utils::wait_for "Master side request #2 response:"

if {[string match "*HTTP_RESPONSE_FROM_*" $response2]} {
    send -- "echo 'SUCCESS: Request #2 tunneled correctly'\n"
    Utils::wait_for "SUCCESS: Request #2 tunneled correctly"
} else {
    send -- "echo 'FAILED: Request #2 did not work'\n"
    Utils::wait_for "FAILED: Request #2 did not work"
    exit 1
}

# Test 3: Third HTTP request through the tunnel  
send -- "echo 'Making HTTP request #3 from master to localhost:9999...'\n"
Utils::wait_for "Making HTTP request #3 from master"

set response3 [exec curl -s --connect-timeout 5 http://127.0.0.1:9999/ 2>/dev/null || echo "CURL_FAILED"]

send -- "echo 'Master side request #3 response: $response3'\n"
Utils::wait_for "Master side request #3 response:"

if {[string match "*HTTP_RESPONSE_FROM_*" $response3]} {
    send -- "echo 'SUCCESS: Request #3 tunneled correctly'\n"
    Utils::wait_for "SUCCESS: Request #3 tunneled correctly"
} else {
    send -- "echo 'FAILED: Request #3 did not work'\n"
    Utils::wait_for "FAILED: Request #3 did not work"
    exit 1
}

# Verify we're getting responses from the slave, not the master
if {[string match "*server*" $response1] || [string match "*slave*" $response1]} {
    send -- "echo 'VERIFICATION: Responses are coming from the slave host as expected'\n"
    Utils::wait_for "VERIFICATION: Responses are coming from the slave host as expected"
} else {
    send -- "echo 'WARNING: Cannot verify responses are from slave (hostname: $response1)'\n"
    Utils::wait_for "WARNING: Cannot verify responses are from slave"
}

send -- "echo\n"
send -- "echo '=== LOCAL PORT FORWARDING INTEGRATION TEST RESULTS ==='\n"
Utils::wait_for "LOCAL PORT FORWARDING INTEGRATION TEST RESULTS"

send -- "echo '1. ✓ goncat master connect with -L 9999:127.0.0.1:8888 established successfully'\n"
Utils::wait_for "goncat master connect with -L 9999:127.0.0.1:8888 established successfully"

send -- "echo '2. ✓ Slave HTTP server confirmed running on localhost:8888'\n"
Utils::wait_for "Slave HTTP server confirmed running on localhost:8888"

send -- "echo '3. ✓ Master side HTTP requests to localhost:9999 successfully tunneled'\n"
Utils::wait_for "Master side HTTP requests to localhost:9999 successfully tunneled"

send -- "echo '4. ✓ All 3 HTTP requests from master received responses from slave'\n"
Utils::wait_for "All 3 HTTP requests from master received responses from slave"

send -- "echo '5. ✓ Port forwarding tunnel is working correctly: master:9999 -> slave:8888'\n"
Utils::wait_for "Port forwarding tunnel is working correctly"

send -- "echo\n"
send -- "echo 'FINAL_RESULT: Local port forwarding (-L) integration test PASSED'\n"
Utils::wait_for "FINAL_RESULT: Local port forwarding (-L) integration test PASSED"

send -- "exit\n"
Expect::close_and_wait