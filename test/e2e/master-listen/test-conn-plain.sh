#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 10

# Test 1: Plain server -> Plain client (should succeed)
puts "\n=== Test 1: Plain server -> Plain client (port 8080) ==="
spawn /opt/dist/goncat.elf master listen $transport://:8080 --timeout 2000

# Wait for session establishment message
expect {
    -re "Session with .* established \\(slave\\[" {
        # Good - session established
    }
    -re "Error:" {
        puts "✗ Test 1 FAILED: Got error instead of session establishment\n"
        exit 1
    }
    timeout {
        puts "✗ Test 1 FAILED: Timeout waiting for session establishment\n"
        exit 1
    }
}

puts "✓ Test 1 passed: Plain connection established (session message seen)\n"
Expect::close_and_wait

# Test 2: Plain server -> TLS client (should fail - no session, has error)
puts "\n=== Test 2: Plain server -> TLS client (port 8081) ==="
spawn /opt/dist/goncat.elf master listen $transport://:8081 --timeout 2000
set test2_session_seen 0
set test2_error_seen 0
set timeout 8

expect {
    -re "Session with .* established" {
        set test2_session_seen 1
        exp_continue
    }
    -re "Error:" {
        set test2_error_seen 1
        exp_continue
    }
    timeout {
        # Timeout is acceptable
    }
    eof {
        # EOF is acceptable
    }
}
catch {close}
catch {wait}

if {$test2_session_seen == 1} {
    puts "✗ Test 2 FAILED: Session establishment message should not appear\n"
    exit 1
}
# For plain server with TLS client, errors typically appear quickly
if {$test2_error_seen == 0} {
    puts "✗ Test 2 FAILED: Error message should appear\n"
    exit 1
}
puts "✓ Test 2 passed: Plain server rejected TLS client (no session, has error)\n"

# Test 3: Plain server -> mTLS client (should fail - no session, has error)
puts "\n=== Test 3: Plain server -> mTLS client (port 8082) ==="
spawn /opt/dist/goncat.elf master listen $transport://:8082 --timeout 2000
set test3_session_seen 0
set test3_error_seen 0
set timeout 8

expect {
    -re "Session with .* established" {
        set test3_session_seen 1
        exp_continue
    }
    -re "Error:" {
        set test3_error_seen 1
        exp_continue
    }
    timeout {
        # Timeout is acceptable
    }
    eof {
        # EOF is acceptable
    }
}
catch {close}
catch {wait}

if {$test3_session_seen == 1} {
    puts "✗ Test 3 FAILED: Session establishment message should not appear\n"
    exit 1
}
# For plain server with mTLS client, errors typically appear quickly
if {$test3_error_seen == 0} {
    puts "✗ Test 3 FAILED: Error message should appear\n"
    exit 1
}
puts "✓ Test 3 passed: Plain server rejected mTLS client (no session, has error)\n"

puts "\n✓ All plain connection tests passed!"
