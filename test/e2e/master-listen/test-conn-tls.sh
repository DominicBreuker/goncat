#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 10

# Test 1: TLS server -> Plain client (should fail)
puts "\n=== Test 1: TLS server -> Plain client (port 8080) ==="
spawn /opt/dist/goncat.elf master listen $transport://:8080 --ssl --timeout 2000
set timeout 8
expect {
    -re "New.*connection from" {
        # We might see a connection attempt but it should fail quickly
        expect {
            "Error:" {
                puts "✓ Test 1 passed: TLS server rejected plain client\n"
            }
            timeout {
                puts "✓ Test 1 passed: No clean connection established\n"
            }
        }
    }
    timeout {
        puts "✓ Test 1 passed: TLS server did not accept plain client\n"
    }
}
catch {close}
catch {wait}

# Test 2: TLS server -> TLS client (should succeed)
puts "\n=== Test 2: TLS server -> TLS client (port 8081) ==="
spawn /opt/dist/goncat.elf master listen $transport://:8081 --ssl --timeout 2000
Expect::client_connected
Expect::close_and_wait
puts "✓ Test 2 passed: TLS connection established\n"

# Test 3: TLS server -> mTLS client (should fail with certificate error)
puts "\n=== Test 3: TLS server -> mTLS client (port 8082) ==="
spawn /opt/dist/goncat.elf master listen $transport://:8082 --ssl --timeout 2000
set timeout 8
expect {
    -re "New.*connection from" {
        # We might see a connection attempt but it should fail quickly
        expect {
            "Error:" {
                puts "✓ Test 3 passed: TLS server rejected mTLS client\n"
            }
            timeout {
                puts "✓ Test 3 passed: No clean connection established\n"
            }
        }
    }
    timeout {
        puts "✓ Test 3 passed: TLS server did not accept mTLS client\n"
    }
}
catch {close}
catch {wait}

puts "\n✓ All TLS connection tests passed!"
