#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 10

# Test 1: Plain server -> Plain client (should succeed)
puts "\n=== Test 1: Plain server -> Plain client (port 8080) ==="
spawn /opt/dist/goncat.elf master listen $transport://:8080 --timeout 2000
Expect::client_connected
Expect::close_and_wait
puts "✓ Test 1 passed: Plain connection established\n"

# Test 2: Plain server -> TLS client (should fail with handshake error on server side)
puts "\n=== Test 2: Plain server -> TLS client (port 8081) ==="
# The slave will be trying to connect with TLS to our plain server
# We expect NO successful connection message
spawn /opt/dist/goncat.elf master listen $transport://:8081 --timeout 2000
set timeout 8
expect {
    "New connection from" {
        # Connection message should NOT appear for failed TLS handshakes
        puts "✗ Test 2 failed: Plain server accepted TLS client connection\n"
        exit 1
    }
    "Error:" {
        puts "✓ Test 2 passed: Plain server rejected TLS client\n"
    }
    timeout {
        puts "✓ Test 2 passed: Plain server did not accept TLS client\n"
    }
}
catch {close}
catch {wait}

# Test 3: Plain server -> mTLS client (should fail with handshake error on server side)
puts "\n=== Test 3: Plain server -> mTLS client (port 8082) ==="
spawn /opt/dist/goncat.elf master listen $transport://:8082 --timeout 2000
set timeout 8
expect {
    "New connection from" {
        # Connection message should NOT appear for failed TLS handshakes
        puts "✗ Test 3 failed: Plain server accepted mTLS client connection\n"
        exit 1
    }
    "Error:" {
        puts "✓ Test 3 passed: Plain server rejected mTLS client\n"
    }
    timeout {
        puts "✓ Test 3 passed: Plain server did not accept mTLS client\n"
    }
}
catch {close}
catch {wait}

puts "\n✓ All plain connection tests passed!"
