FROM alpine:3

RUN apk add --no-cache expect

# Copy echo server scripts
COPY server-companion-echo.sh /opt/server-companion-echo.sh
COPY client-companion-echo.sh /opt/client-companion-echo.sh
COPY echo-handler.sh /opt/echo-handler.sh

# Make scripts executable
RUN chmod +x /opt/server-companion-echo.sh /opt/client-companion-echo.sh /opt/echo-handler.sh
