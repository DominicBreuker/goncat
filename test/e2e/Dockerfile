FROM alpine:3

RUN apk add --no-cache expect socat

# Copy echo server scripts
COPY slave-companion-echo.sh /opt/slave-companion-echo.sh
COPY master-companion-echo.sh /opt/master-companion-echo.sh
COPY echo-handler.sh /opt/echo-handler.sh

# Make scripts executable
RUN chmod +x /opt/slave-companion-echo.sh /opt/master-companion-echo.sh /opt/echo-handler.sh
