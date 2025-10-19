#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 10

# Test 1: Plain client -> Plain server (should succeed)
puts "\n=== Test 1: Plain client -> Plain server (port 8080) ==="
spawn /opt/dist/goncat.elf master connect $transport://slave:8080 --timeout 2000
Expect::server_connected
Expect::close_and_wait
puts "✓ Test 1 passed: Plain connection established\n"

# Test 2: Plain client -> TLS server (should fail with timeout/EOF)
puts "\n=== Test 2: Plain client -> TLS server (port 8081) ==="
spawn /opt/dist/goncat.elf master connect $transport://slave:8081 --timeout 2000
set timeout 5
expect {
    "Error: Run: connecting:" {
        puts "✓ Test 2 passed: Plain client cannot connect to TLS server\n"
    }
    timeout {
        puts "✓ Test 2 passed: Plain client timed out connecting to TLS server\n"
    }
    -re "New.*connection" {
        puts "✗ Test 2 failed: Plain client unexpectedly connected to TLS server"
        exit 1
    }
}
catch {close}
catch {wait}

# Test 3: Plain client -> mTLS server (should fail with timeout/EOF)
puts "\n=== Test 3: Plain client -> mTLS server (port 8082) ==="
spawn /opt/dist/goncat.elf master connect $transport://slave:8082 --timeout 2000
set timeout 5
expect {
    "Error: Run: connecting:" {
        puts "✓ Test 3 passed: Plain client cannot connect to mTLS server\n"
    }
    timeout {
        puts "✓ Test 3 passed: Plain client timed out connecting to mTLS server\n"
    }
    -re "New.*connection" {
        puts "✗ Test 3 failed: Plain client unexpectedly connected to mTLS server"
        exit 1
    }
}
catch {close}
catch {wait}

puts "\n✓ All plain connection tests passed!"
