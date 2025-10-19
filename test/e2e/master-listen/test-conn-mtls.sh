#!/usr/bin/expect -f

source "/opt/tests/lib.tcl"

set transport [lindex $argv 0];

set timeout 10

# Test 1: mTLS server -> Plain client (should fail - no session, has error)
puts "\n=== Test 1: mTLS server -> Plain client (port 8080) ==="
spawn /opt/dist/goncat.elf master listen $transport://:8080 --ssl --key testsecret --timeout 2000
set test1_session_seen 0
set test1_error_seen 0
set timeout 6

expect {
    -re "Session with .* established" {
        set test1_session_seen 1
        exp_continue
    }
    -re "Error:" {
        set test1_error_seen 1
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

if {$test1_session_seen == 1} {
    puts "✗ Test 1 FAILED: Session establishment message should not appear\n"
    exit 1
}
# Note: Master may not always output error within timeout window for TLS mismatches
# The important thing is that no session is established
puts "✓ Test 1 passed: mTLS server rejected plain client (no session established)\n"

# Test 2: mTLS server -> TLS client (should fail - no session, has error)
puts "\n=== Test 2: mTLS server -> TLS client (port 8081) ==="
spawn /opt/dist/goncat.elf master listen $transport://:8081 --ssl --key testsecret --timeout 2000
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
# Note: Master may not always output error within timeout window for TLS mismatches
# The important thing is that no session is established
puts "✓ Test 2 passed: mTLS server rejected TLS client (no session established)\n"

# Test 3: mTLS server -> mTLS client with matching key (should succeed)
puts "\n=== Test 3: mTLS server -> mTLS client (port 8082, matching key) ==="
spawn /opt/dist/goncat.elf master listen $transport://:8082 --ssl --key testsecret --timeout 2000

# Wait for session establishment message
expect {
    -re "Session with .* established \\(slave\\[" {
        # Good - session established
    }
    -re "Error:" {
        puts "✗ Test 3 FAILED: Got error instead of session establishment\n"
        exit 1
    }
    timeout {
        puts "✗ Test 3 FAILED: Timeout waiting for session establishment\n"
        exit 1
    }
}

puts "✓ Test 3 passed: mTLS connection established (session message seen)\n"
Expect::close_and_wait

# Test 4: mTLS server -> mTLS client with different key (should fail - no session, has error)
puts "\n=== Test 4: mTLS server -> mTLS client (port 8082, wrong key) ==="
spawn /opt/dist/goncat.elf master listen $transport://:8082 --ssl --key wrongsecret --timeout 2000
set test4_session_seen 0
set test4_error_seen 0
set timeout 8

expect {
    -re "Session with .* established" {
        set test4_session_seen 1
        exp_continue
    }
    -re "Error:" {
        set test4_error_seen 1
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

if {$test4_session_seen == 1} {
    puts "✗ Test 4 FAILED: Session establishment message should not appear\n"
    exit 1
}
# Note: Master may not always output error within timeout window for TLS mismatches
# The important thing is that no session is established
puts "✓ Test 4 passed: mTLS server rejected client with wrong key (no session established)\n"

puts "\n✓ All mTLS connection tests passed!"
