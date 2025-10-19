#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 10

# Test 1: mTLS client -> Plain server (should fail with handshake error)
puts "\n=== Test 1: mTLS client -> Plain server (port 8080) ==="
spawn /opt/dist/goncat.elf master connect $transport://slave:8080 --ssl --key testsecret --timeout 2000
set timeout 5
expect {
    "Error: Run: connecting: upgrade to tls: tls handshake:" {
        puts "✓ Test 1 passed: mTLS client cannot connect to plain server\n"
    }
    -re "Error:" {
        puts "✓ Test 1 passed: mTLS client got error connecting to plain server\n"
    }
    timeout {
        puts "✓ Test 1 passed: mTLS client timed out connecting to plain server\n"
    }
    eof {
        puts "✓ Test 1 passed: Connection closed\n"
    }
}
catch {close}
catch {wait}

# Test 2: mTLS client -> TLS server (should fail with certificate error)
puts "\n=== Test 2: mTLS client -> TLS server (port 8080) ==="
spawn /opt/dist/goncat.elf master connect $transport://slave-tls:8080 --ssl --key testsecret --timeout 2000
set timeout 5
expect {
    "Error: Run: connecting: upgrade to tls: tls handshake: verify certificate:" {
        puts "✓ Test 2 passed: mTLS client cannot connect to TLS server (cert verification failed)\n"
    }
    "Error: Run: connecting: upgrade to tls: tls handshake:" {
        puts "✓ Test 2 passed: mTLS client cannot connect to TLS server (handshake failed)\n"
    }
    -re "Error:" {
        puts "✓ Test 2 passed: mTLS client got error connecting to TLS server\n"
    }
    timeout {
        puts "✓ Test 2 passed: mTLS client timed out connecting to TLS server\n"
    }
    eof {
        puts "✓ Test 2 passed: Connection closed\n"
    }
}
catch {close}
catch {wait}

# Test 3: mTLS client -> mTLS server with matching key (should succeed)
puts "\n=== Test 3: mTLS client -> mTLS server (port 8080, matching key) ==="
spawn /opt/dist/goncat.elf master connect $transport://slave-mtls:8080 --ssl --key testsecret --timeout 2000
Expect::server_connected
Expect::close_and_wait
puts "✓ Test 3 passed: mTLS connection established\n"

# Test 4: mTLS client -> mTLS server with different key (should fail)
puts "\n=== Test 4: mTLS client -> mTLS server (port 8080, wrong key) ==="
spawn /opt/dist/goncat.elf master connect $transport://slave-mtls:8080 --ssl --key wrongsecret --timeout 2000
set timeout 5
expect {
    "Error: Run: connecting: upgrade to tls: tls handshake: verify certificate: x509: certificate signed by unknown authority" {
        puts "✓ Test 4 passed: mTLS client with wrong key cannot connect\n"
    }
    "Error: Run: connecting: upgrade to tls: tls handshake: verify certificate:" {
        puts "✓ Test 4 passed: mTLS client with wrong key cannot connect (cert verification failed)\n"
    }
    "Error: Run: connecting: upgrade to tls: tls handshake:" {
        puts "✓ Test 4 passed: mTLS client with wrong key cannot connect (handshake failed)\n"
    }
    -re "Error:" {
        puts "✓ Test 4 passed: mTLS client with wrong key got error\n"
    }
    timeout {
        puts "✓ Test 4 passed: mTLS client with wrong key timed out\n"
    }
    eof {
        puts "✓ Test 4 passed: Connection closed\n"
    }
}
catch {close}
catch {wait}

puts "\n✓ All mTLS connection tests passed!"
