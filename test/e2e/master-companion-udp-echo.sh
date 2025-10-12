#!/bin/sh
# UDP echo server for master-companion
# Listens on UDP port 9001 and echoes back messages with a prefix

# Use socat to create UDP echo server that echoes back
# UDP4-RECVFROM:9001 receives UDP datagrams and provides sender address
# EXEC reads input, runs script, and sends output back to sender
while true; do
    socat UDP4-RECVFROM:9001,fork,reuseaddr EXEC:"/opt/udp-echo-handler.sh" || sleep 1
done
