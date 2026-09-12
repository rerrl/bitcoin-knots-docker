#!/bin/sh
set -e

# Fix permissions on the mounted host directories — Tor requires 700
# owned by the tor user, but Docker volumes inherit host permissions
chown -R tor:tor /var/lib/tor/ 2>/dev/null || true
chmod 700 /var/lib/tor/ 2>/dev/null || true
chmod 700 /var/lib/tor/hidden_service/ 2>/dev/null || true

# Shared logs volume: make sure our log file exists and is readable by the
# dashboard (tor drops privileges to the `tor` user after startup, so open the
# file up enough that both the tor user and the dashboard's uid can use it).
mkdir -p /logs
touch /logs/tor.log
chown tor:tor /logs/tor.log 2>/dev/null || true
chmod 0666 /logs/tor.log 2>/dev/null || true

# Tor cannot resolve hostnames for HiddenServicePort targets — it only accepts
# numeric IP addresses. Docker injects the fulcrum container's address into
# /etc/hosts under the service name, so resolve it and rewrite the config with a
# real IP target before starting (avoids "Unparseable address in hidden service
# port configuration" -> config parse failure -> restart loop).
FULCRUM_IP=$(awk '$2=="fulcrum"{print $1; exit}' /etc/hosts)
if [ -z "$FULCRUM_IP" ]; then
  echo "ERROR: could not resolve fulcrum container address" >&2
  exit 1
fi
sed "s/^HiddenServicePort 50001 .*/HiddenServicePort 50001 ${FULCRUM_IP}:50001/" \
    /etc/tor/torrc > /tmp/torrc

# Start Tor in the background, logging to the shared file (visible in the
# dashboard) instead of to stdout. -f uses the rewritten config above.
tor -f /tmp/torrc >>/logs/tor.log 2>&1 &

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

# Publish a readable copy of the onion into the shared /logs volume so the
# dashboard (a separate container/uid) can read it. The hidden-service dir
# itself stays 0700-owned-by-tor (Tor refuses to serve it otherwise), which is
# exactly why other uids can't read hostname directly from there. /logs is
# world-usable, so this copy is the clean handoff channel.
cp /var/lib/tor/hidden_service/hostname /logs/tor-hostname
chmod 0644 /logs/tor-hostname

# Display the onion address
ONION_ADDRESS=$(cat /var/lib/tor/hidden_service/hostname)
echo "=========================================="
echo "  Tor Hidden Service: $ONION_ADDRESS"
echo "  Connect with Sparrow Wallet:"
echo "    Server: $ONION_ADDRESS"
echo "    Port:   50001"
echo "    Protocol: TCP (SSL disabled)"
echo "  See it in the dashboard under Overview -> Connect with Sparrow."
echo "=========================================="

# Wait for the tor background process
wait