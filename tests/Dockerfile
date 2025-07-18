FROM ubuntu:22.04

# Install required packages
RUN apt-get update && apt-get install -y \
    expect \
    netcat-openbsd \
    telnet \
    socat \
    && rm -rf /var/lib/apt/lists/*
