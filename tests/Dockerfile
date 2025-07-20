FROM ubuntu:22.04

# Install required packages including networking tools for testing
RUN apt-get update && apt-get install -y \
    expect \
    netcat-openbsd \
    curl \
    python3 \
    && rm -rf /var/lib/apt/lists/*
