#!/bin/sh
# UDP echo handler script that reads input and echoes it back with hostname prefix

# Read input and echo it with hostname prefix
# For UDP, we just read stdin and write to stdout
read line && echo "$(hostname) says: $line"
