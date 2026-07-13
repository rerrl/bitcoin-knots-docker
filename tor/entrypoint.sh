#!/bin/sh
set -e

# Fix permissions on the mounted host directories — Tor requires 700
# owned by the tor user, but Docker volumes inherit host permissions
chown -R tor:tor /var/lib/tor/ 2>/dev/null || true
chmod 700 /var/lib/tor/ 2>/dev/null || true
chmod 700 /var/lib/tor/hidden_service/ 2>/dev/null || true

# Start Tor in the background
tor -f /etc/tor/torrc &

# Wait for the hidden service hostname to be generated
echo "Waiting for Tor hidden service to be ready..."
i=0
while [ ! -f /var/lib/tor/hidden_service/hostname ]; do
  sleep 1
  i=$((i + 1))
  if [ $i -ge 30 ]; then
    echo "ERROR: Tor hidden service did not become ready within 30 seconds"
    exit 1
  fi
done

# Display the onion address
ONION_ADDRESS=$(cat /var/lib/tor/hidden_service/hostname)
echo "=========================================="
echo "  Tor Hidden Service: $ONION_ADDRESS"
echo "  Connect with Sparrow Wallet:"
echo "    Server: $ONION_ADDRESS"
echo "    Port:   50001"
echo "    Protocol: TCP (SSL disabled)"
echo "=========================================="

# Wait for the tor background process
wait