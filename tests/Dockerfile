FROM ubuntu:22.04

# Install required packages including networking tools for testing
RUN apt-get update && apt-get install -y \
    expect \
    netcat-openbsd \
    curl \
    python3 \
    && rm -rf /var/lib/apt/lists/*

# Create a simple HTTP server script that returns the hostname
RUN echo '#!/usr/bin/env python3' > /usr/local/bin/test-http-server.py && \
    echo 'import http.server, socketserver, socket' >> /usr/local/bin/test-http-server.py && \
    echo 'class HostnameHandler(http.server.BaseHTTPRequestHandler):' >> /usr/local/bin/test-http-server.py && \
    echo '    def do_GET(self):' >> /usr/local/bin/test-http-server.py && \
    echo '        hostname = socket.gethostname()' >> /usr/local/bin/test-http-server.py && \
    echo '        self.send_response(200)' >> /usr/local/bin/test-http-server.py && \
    echo '        self.send_header("Content-type", "text/plain")' >> /usr/local/bin/test-http-server.py && \
    echo '        self.end_headers()' >> /usr/local/bin/test-http-server.py && \
    echo '        self.wfile.write(f"HTTP_RESPONSE_FROM_{hostname}".encode())' >> /usr/local/bin/test-http-server.py && \
    echo '    def log_message(self, format, *args): pass' >> /usr/local/bin/test-http-server.py && \
    echo 'with socketserver.TCPServer(("127.0.0.1", 8888), HostnameHandler) as server:' >> /usr/local/bin/test-http-server.py && \
    echo '    print(f"HTTP server listening on {socket.gethostname()}:127.0.0.1:8888")' >> /usr/local/bin/test-http-server.py && \
    echo '    server.serve_forever()' >> /usr/local/bin/test-http-server.py && \
    chmod +x /usr/local/bin/test-http-server.py

# Start the HTTP server in the background by default
RUN echo '#!/bin/bash' > /usr/local/bin/start-test-server.sh && \
    echo 'python3 /usr/local/bin/test-http-server.py &' >> /usr/local/bin/start-test-server.sh && \
    echo 'exec "$@"' >> /usr/local/bin/start-test-server.sh && \
    chmod +x /usr/local/bin/start-test-server.sh

ENTRYPOINT ["/usr/local/bin/start-test-server.sh"]
