#!/bin/sh
# TCP and UDP echo servers for slave-companion
# TCP: Listens on port 9000
# UDP: Listens on port 9001

# Start TCP echo server in background
(
    while true; do
        # replaced nc with socat: listen on TCP 9000, reuseaddr, fork per connection,
        # and execute the echo handler for each incoming connection
        socat TCP-LISTEN:9000,reuseaddr,fork EXEC:"/opt/tests/helpers/echo-handler.sh" || sleep 1
    done
) &

# Start UDP echo server in foreground
while true; do
    socat UDP4-RECVFROM:9001,fork,reuseaddr EXEC:"/opt/tests/helpers/echo-handler.sh" || sleep 1
done
