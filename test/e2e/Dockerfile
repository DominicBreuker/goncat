FROM alpine:3

RUN apk add --no-cache expect socat

# Copy TCP echo server scripts
COPY slave-companion-echo.sh /opt/slave-companion-echo.sh
COPY master-companion-echo.sh /opt/master-companion-echo.sh
COPY echo-handler.sh /opt/echo-handler.sh

# Copy UDP echo server scripts
COPY slave-companion-udp-echo.sh /opt/slave-companion-udp-echo.sh
COPY master-companion-udp-echo.sh /opt/master-companion-udp-echo.sh
COPY udp-echo-handler.sh /opt/udp-echo-handler.sh

# Make scripts executable
RUN chmod +x /opt/slave-companion-echo.sh /opt/master-companion-echo.sh /opt/echo-handler.sh \
             /opt/slave-companion-udp-echo.sh /opt/master-companion-udp-echo.sh /opt/udp-echo-handler.sh
