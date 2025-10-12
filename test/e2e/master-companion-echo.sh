#!/bin/sh
# TCP and UDP echo servers for master-companion
# TCP: Listens on port 9000
# UDP: Listens on port 9001

# Start TCP echo server in background
(
    while true; do
        nc -lk -p 9000 -e sh /opt/echo-handler.sh || sleep 1
    done
) &

# Start UDP echo server in foreground
while true; do
    socat UDP4-RECVFROM:9001,fork,reuseaddr EXEC:"/opt/udp-echo-handler.sh" || sleep 1
done
