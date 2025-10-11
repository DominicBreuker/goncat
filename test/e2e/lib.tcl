
#source "/opt/tests/utils.tcl"

namespace eval Expect {
    proc server_connected {} {
        Utils::wait_for "Connecting to"
    }

    proc client_connected {} {
        Utils::wait_for "New * connection from"
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

    proc close_and_wait {} {
        close
        wait
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
