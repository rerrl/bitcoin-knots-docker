FROM alpine:latest

# Install tor
RUN apk add --no-cache tor

# Copy configuration and entrypoint
COPY tor/torrc /etc/tor/torrc
COPY tor/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Tor control port (optional, useful for debugging)
EXPOSE 9050

ENTRYPOINT ["/entrypoint.sh"]