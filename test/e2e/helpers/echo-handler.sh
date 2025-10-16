#!/bin/sh
# Handler script that reads input and echoes it back with hostname prefix

read line && echo "$(hostname) says: $line"
