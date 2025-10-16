FROM alpine:3

RUN apk add --no-cache expect socat python3 py3-pip

# Install PySocks for SOCKS5 UDP ASSOCIATE testing
RUN pip3 install --no-cache-dir PySocks --break-system-packages

# Compose files mount the repo folder `/test/e2e/ to `/opt/tests` in the container
# Do not copy files here!