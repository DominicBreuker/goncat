#!/bin/sh
# Echo server for slave-companion
# Listens on port 9000 and echoes back messages with a prefix

nc -lk -p 9000 -e sh /opt/echo-handler.sh
