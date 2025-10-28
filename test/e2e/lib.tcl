
#source "/opt/tests/utils.tcl"

namespace eval Expect {
    proc server_connected {} {
        Utils::wait_for "Connecting to"
    }

    proc client_connected {} {
        Utils::wait_for "Session with"
    }

    proc shell_access_works {} {
        send -- "id\n"
        Utils::wait_for "uid="
        
        send -- "hostname\n"
        Utils::wait_for "the_slave" 0
    }

    proc exit_with_eof {} {
        send -- "exit\n"
        Utils::wait_for eof 0
    }

    proc exit_without_eof {} {
        send -- "exit\n"
    }

    proc shutdown {} {
        send -- "\003"
    }

    proc close_and_wait {} {
        shutdown

        after 500
        close
        wait
    }

    # Run a local port forwarding verification.
    # Usage: Expect::check_local_forward <local_port> <remote_prefix> ?message? ?timeout?
    # - local_port: port on localhost to connect to (on the master side)
    # - remote_prefix: hostname prefix used by the echo handler (e.g. "slave-companion")
    # - message: the message string to send (default: "test message")
    # - timeout: expect timeout in seconds (default: 5)
    proc check_local_forward {local_port remote_prefix {message "test message"} {timeout 5}} {
        # Save the currently active spawn (should be the master connection).
        # The active spawn id lives in the caller/global scope as ::spawn_id; use that.
        if {[info exists ::spawn_id]} {
            set spawn_id_master $::spawn_id
        } else {
            set spawn_id_master ""
        }

        # Start a socat client that connects to the forwarded local port
        spawn socat - TCP:localhost:$local_port
        set spawn_id_client $spawn_id

        # Send the message and wait for the expected echo from the remote prefix
        send -- "$message\r"

        expect -i $spawn_id_client {
            "*${remote_prefix} says: $message*" {
                puts "\n✓ Local port forwarding test successful!"
            }
            timeout {
                puts stderr "\n✗ Timeout waiting for response from ${remote_prefix}"
                exit 1
            }
            eof {
                puts stderr "\n✗ Unexpected EOF while waiting for response"
                exit 1
            }
        }

        # Clean up the socat connection (guard against already-closed spawn ids)
        if {[catch {close -i $spawn_id_client} _]} {
            # spawn id already closed or not available; ignore
        }
        if {[catch {wait -i $spawn_id_client} _]} {
            # wait failed or spawn already reaped; ignore
        }

        # Restore the master's spawn id in the global scope so callers can continue cleanup
        if {$spawn_id_master ne ""} {
            set ::spawn_id $spawn_id_master
        } else {
            # nothing to restore
        }
    }

    # Run a remote port forwarding verification.
    # Usage: Expect::check_remote_forward <remote_port> <remote_prefix> ?message? ?timeout?
    # - remote_port: port on the *remote* side (the slave) bound by -R
    # - remote_prefix: hostname prefix expected in the echo (e.g. "master-companion")
    # - message: message to send (default: "test message")
    # - timeout: expect timeout in seconds (default: 5)
    proc check_remote_forward {remote_port remote_prefix {message "test message"} {timeout 5}} {
        # We're expected to be in a shell on the remote side (spawn is the master connection
        # which may have an interactive shell). Send a command that connects to localhost:remote_port
        # on the remote side and pipes our message through socat to keep connection open briefly.
        send -- "sh -c 'echo $message | socat - TCP:localhost:$remote_port,shut-none'\r"

        # Wait for expected echo from the remote_prefix on the master side
        expect {
            "*${remote_prefix} says: $message*" {
                puts "\n✓ Remote port forwarding test successful!"
            }
            timeout {
                puts stderr "\n✗ Timeout waiting for response from ${remote_prefix}"
                exit 1
            }
            eof {
                puts stderr "\n✗ Unexpected EOF while waiting for response"
                exit 1
            }
        }

        # Give any remaining output a moment to flush
        sleep 1

        # Exit the remote shell
        send -- "exit\r"
    }

    # Run a UDP local port forwarding verification.
    # Usage: Expect::check_local_forward_udp <local_port> <remote_prefix> ?message? ?timeout?
    # - local_port: UDP port on localhost to send to (on the master side)
    # - remote_prefix: hostname prefix used by the echo handler (e.g. "slave-companion")
    # - message: the message string to send (default: "test message")
    # - timeout: expect timeout in seconds (default: 5)
    proc check_local_forward_udp {local_port remote_prefix {message "test message"} {timeout 5}} {
        # Save the currently active spawn (should be the master connection).
        if {[info exists ::spawn_id]} {
            set spawn_id_master $::spawn_id
        } else {
            set spawn_id_master ""
        }

        # Use socat to send UDP datagram and receive response
        # UDP4-DATAGRAM:localhost:port creates a UDP socket, sends the message, and waits for response
        spawn sh -c "echo '$message' | socat - UDP4-DATAGRAM:localhost:$local_port"
        set spawn_id_client $spawn_id

        # Wait for the expected echo from the remote prefix
        expect -i $spawn_id_client {
            "*${remote_prefix} says: $message*" {
                puts "\n✓ UDP local port forwarding test successful!"
            }
            timeout {
                puts stderr "\n✗ Timeout waiting for UDP response from ${remote_prefix}"
                exit 1
            }
            eof {
                puts stderr "\n✗ Unexpected EOF while waiting for UDP response"
                exit 1
            }
        }

        # Clean up the socat connection
        if {[catch {close -i $spawn_id_client} _]} {
            # spawn id already closed or not available; ignore
        }
        if {[catch {wait -i $spawn_id_client} _]} {
            # wait failed or spawn already reaped; ignore
        }

        # Restore the master's spawn id
        if {$spawn_id_master ne ""} {
            set ::spawn_id $spawn_id_master
        }
    }

    # Run a UDP remote port forwarding verification.
    # Usage: Expect::check_remote_forward_udp <remote_port> <remote_prefix> ?message? ?timeout?
    # - remote_port: UDP port on the *remote* side (the slave) bound by -R
    # - remote_prefix: hostname prefix expected in the echo (e.g. "master-companion")
    # - message: message to send (default: "test message")
    # - timeout: expect timeout in seconds (default: 5)
    proc check_remote_forward_udp {remote_port remote_prefix {message "test message"} {timeout 5}} {
        # We're in a shell on the remote side. Send a UDP datagram to localhost:remote_port
        send -- "sh -c 'echo $message | socat - UDP4-DATAGRAM:localhost:$remote_port'\r"

        # Wait for expected echo from the remote_prefix on the master side
        expect {
            "*${remote_prefix} says: $message*" {
                puts "\n✓ UDP remote port forwarding test successful!"
            }
            timeout {
                puts stderr "\n✗ Timeout waiting for UDP response from ${remote_prefix}"
                exit 1
            }
            eof {
                puts stderr "\n✗ Unexpected EOF while waiting for UDP response"
                exit 1
            }
        }

        # Give any remaining output a moment to flush
        sleep 1

        # Exit the remote shell
        send -- "exit\r"
    }

    # Run a SOCKS5 UDP ASSOCIATE verification using the Python helper client.
    # Usage: Expect::check_socks_udp_associate <socks_host> <socks_port> <target_host> <target_port> ?timeout?
    # - socks_host/socks_port: where the SOCKS proxy is listening (usually localhost 1080)
    # - target_host/target_port: destination for the UDP datagram (e.g. slave-companion 9001)
    # - timeout: seconds to wait (default: 10)
    proc check_socks_udp_associate {socks_host socks_port target_host target_port {timeout 10}} {
        # Save master spawn id (global ::spawn_id)
        if {[info exists ::spawn_id]} {
            set spawn_id_master $::spawn_id
        } else {
            set spawn_id_master ""
        }

        # Spawn the Python PySocks UDP test client
        spawn python3 /opt/tests/helpers/socks5-udp-test.py $socks_host $socks_port $target_host $target_port
        set spawn_id_test $spawn_id

        # Enable logging for the test client so output is visible
        log_user 1

        # Wait for the result
        expect {
            -i $spawn_id_test
            "*✓ UDP ASSOCIATE test successful!*" {
                puts "\n✓ SOCKS UDP ASSOCIATE test successful!"
            }
            "*✗ Error:*" {
                puts stderr "\n✗ SOCKS UDP ASSOCIATE test failed"
                exit 1
            }
            timeout {
                puts stderr "\n✗ Timeout waiting for UDP ASSOCIATE test result"
                exit 1
            }
            eof {
                puts stderr "\n✗ Unexpected EOF from test client"
                exit 1
            }
        }

        # Clean up the test client (guard against already-closed spawn ids)
        if {[catch {close -i $spawn_id_test} _]} {
            # spawn id already closed or not available; ignore
        }
        if {[catch {wait -i $spawn_id_test} _]} {
            # wait failed or spawn already reaped; ignore
        }

        # Restore master spawn id in global scope and return
        if {$spawn_id_master ne ""} {
            set ::spawn_id $spawn_id_master
        }
    }

    # Run a SOCKS5 CONNECT verification using socat through the local proxy.
    # Usage: Expect::check_socks_connect <proxy_host> <proxy_port> <target_host> <target_port> ?message? ?timeout?
    # - proxy_host/proxy_port: where the SOCKS proxy is listening (usually localhost 1080)
    # - target_host/target_port: TCP target reachable via the SOCKS proxy (e.g. slave-companion 9000)
    # - message: the message to send (default: "test message")
    # - timeout: seconds to wait (default: 5)
    proc check_socks_connect {proxy_host proxy_port target_host target_port {message "test message"} {timeout 5}} {
        # Save current spawn id (master connection) from global ::spawn_id
        if {[info exists ::spawn_id]} {
            set spawn_id_master $::spawn_id
        } else {
            set spawn_id_master ""
        }

        # Use socat to connect through SOCKS5 to the target host:port
        # Format used in tests: SOCKS5-CONNECT:<proxy-host>:<proxy-port>:<target-host>:<target-port>
        set socks_arg "SOCKS5-CONNECT:${proxy_host}:${proxy_port}:${target_host}:${target_port}"

        spawn socat - $socks_arg
        set spawn_id_client $spawn_id

        # Send test message
        send -- "$message\r"

        # Expect response from the target_host's echo handler
        expect -i $spawn_id_client {
            "*${target_host} says: $message*" {
                puts "\n✓ SOCKS CONNECT test successful!"
            }
            timeout {
                puts stderr "\n✗ Timeout waiting for response from ${target_host}"
                exit 1
            }
            eof {
                puts stderr "\n✗ Unexpected EOF while waiting for response"
                exit 1
            }
        }

        # Clean up socat client (guard against already-closed spawn ids)
        if {[catch {close -i $spawn_id_client} _]} {
            # spawn id already closed or not available; ignore
        }
        if {[catch {wait -i $spawn_id_client} _]} {
            # wait failed or spawn already reaped; ignore
        }

        # Restore the master's spawn id in global scope
        if {$spawn_id_master ne ""} {
            set ::spawn_id $spawn_id_master
        }
    }
}

namespace eval Utils {
    variable timeout 5
    variable max_retries 2
    variable retry_count 0

    proc set_timeout {value} {
        variable timeout
        set timeout $value
    }

    proc set_max_retries {value} {
        variable max_retries
        set max_retries $value
    }

    # `wait_for my_string` matches the substrig "my_string" in the output
    # `wait_for my_string 0` performs an exact match
    proc wait_for {pattern {substring_match 1}} {
        variable retry_count
        variable max_retries
        variable timeout

        set match_pattern $pattern
        if ($substring_match) {
            set match_pattern "*${pattern}*"
        }

        set ::timeout $timeout

        expect {
            $match_pattern {
                return 1
            }
            timeout {
                incr retry_count
                if {$retry_count >= $max_retries} {
                    puts stderr "Failed after $max_retries attempts"
                    exit 1
                }
                puts stderr "Timeout, retrying ($retry_count/$max_retries)..."
                exp_continue
                
            }
            eof {
                puts stderr "Unexpected EOF"
                exit 1
            }
        }
    }

    proc reset_retries {} {
        variable retry_count
        set retry_count 0
    }
}
