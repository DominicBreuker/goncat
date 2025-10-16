#!/bin/sh
# TCP and UDP echo servers for master-companion
# TCP: Listens on port 9000
# UDP: Listens on port 9001

# Start TCP echo server in background
(
    while true; do
        socat TCP4-LISTEN:9000,reuseaddr,fork EXEC:"/opt/tests/helpers/echo-handler.sh" || sleep 1
    done
) &

# Start UDP echo server in foreground
while true; do
    socat UDP4-RECVFROM:9001,fork,reuseaddr EXEC:"/opt/tests/helpers/echo-handler.sh" || sleep 1
done
