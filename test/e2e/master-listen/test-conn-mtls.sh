#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 10

# Test 1: mTLS server -> Plain client (should fail)
puts "\n=== Test 1: mTLS server -> Plain client (port 8080) ==="
spawn /opt/dist/goncat.elf master listen $transport://:8080 --ssl --key testsecret --timeout 2000
set timeout 8
expect {
    "New connection from" {
        # Connection message should NOT appear for failed TLS handshakes
        puts "✗ Test 1 failed: mTLS server accepted plain client connection\n"
        exit 1
    }
    "Error:" {
        puts "✓ Test 1 passed: mTLS server rejected plain client\n"
    }
    timeout {
        puts "✓ Test 1 passed: mTLS server did not accept plain client\n"
    }
}
catch {close}
catch {wait}

# Test 2: mTLS server -> TLS client (should fail with certificate error)
puts "\n=== Test 2: mTLS server -> TLS client (port 8081) ==="
spawn /opt/dist/goncat.elf master listen $transport://:8081 --ssl --key testsecret --timeout 2000
set timeout 8
expect {
    "New connection from" {
        # Connection message should NOT appear for failed TLS handshakes
        puts "✗ Test 2 failed: mTLS server accepted TLS client connection\n"
        exit 1
    }
    "Error:" {
        puts "✓ Test 2 passed: mTLS server rejected TLS client\n"
    }
    timeout {
        puts "✓ Test 2 passed: mTLS server did not accept TLS client\n"
    }
}
catch {close}
catch {wait}

# Test 3: mTLS server -> mTLS client with matching key (should succeed)
puts "\n=== Test 3: mTLS server -> mTLS client (port 8082, matching key) ==="
spawn /opt/dist/goncat.elf master listen $transport://:8082 --ssl --key testsecret --timeout 2000
Expect::client_connected
Expect::close_and_wait
puts "✓ Test 3 passed: mTLS connection established\n"

# Test 4: mTLS server -> mTLS client with different key (should fail)
# We need to connect with a different key than the slave is using
# The slave uses 'testsecret', so we'll use 'wrongsecret'
puts "\n=== Test 4: mTLS server -> mTLS client (port 8082, wrong key) ==="
spawn /opt/dist/goncat.elf master listen $transport://:8082 --ssl --key wrongsecret --timeout 2000
set timeout 8
expect {
    "New connection from" {
        # Connection message should NOT appear for failed TLS handshakes
        puts "✗ Test 4 failed: mTLS server accepted client with wrong key\n"
        exit 1
    }
    "Error:" {
        puts "✓ Test 4 passed: mTLS server rejected client with wrong key\n"
    }
    timeout {
        puts "✓ Test 4 passed: mTLS server did not accept client with wrong key\n"
    }
}
catch {close}
catch {wait}

puts "\n✓ All mTLS connection tests passed!"
