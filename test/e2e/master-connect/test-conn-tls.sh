#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 10

# Test 1: TLS client -> Plain server (should fail with handshake error)
puts "\n=== Test 1: TLS client -> Plain server (port 8080) ==="
spawn /opt/dist/goncat.elf master connect $transport://slave:8080 --ssl --timeout 2000
set timeout 5
expect {
    "Error: Run: connecting: upgrade to tls: tls handshake:" {
        puts "✓ Test 1 passed: TLS client cannot connect to plain server\n"
    }
    timeout {
        puts "✓ Test 1 passed: TLS client timed out connecting to plain server\n"
    }
    -re "New.*connection" {
        puts "✗ Test 1 failed: TLS client unexpectedly connected to plain server"
        exit 1
    }
}
catch {close}
catch {wait}

# Test 2: TLS client -> TLS server (should succeed)
puts "\n=== Test 2: TLS client -> TLS server (port 8081) ==="
spawn /opt/dist/goncat.elf master connect $transport://slave:8081 --ssl --timeout 2000
Expect::server_connected
Expect::close_and_wait
puts "✓ Test 2 passed: TLS connection established\n"

# Test 3: TLS client -> mTLS server (should fail with certificate error)
puts "\n=== Test 3: TLS client -> mTLS server (port 8082) ==="
spawn /opt/dist/goncat.elf master connect $transport://slave:8082 --ssl --timeout 2000
set timeout 5
expect {
    "Error: Run: connecting: upgrade to tls: tls handshake: verify certificate:" {
        puts "✓ Test 3 passed: TLS client cannot connect to mTLS server (cert verification failed)\n"
    }
    "Error: Run: connecting: upgrade to tls: tls handshake:" {
        puts "✓ Test 3 passed: TLS client cannot connect to mTLS server (handshake failed)\n"
    }
    timeout {
        puts "✓ Test 3 passed: TLS client timed out connecting to mTLS server\n"
    }
    -re "New.*connection" {
        puts "✗ Test 3 failed: TLS client unexpectedly connected to mTLS server"
        exit 1
    }
}
catch {close}
catch {wait}

puts "\n✓ All TLS connection tests passed!"
