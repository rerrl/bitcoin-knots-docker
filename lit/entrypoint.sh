#!/bin/bash
set -e

# Validate required env vars
if [ -z "$LIT_UI_PASSWORD" ]; then
    echo "ERROR: LIT_UI_PASSWORD is not set."
    echo "Set it in your .env file, e.g.: LIT_UI_PASSWORD=yourpassword"
    exit 1
fi

# Generate lit.conf from environment
cat > /root/.lit/lit.conf << EOF
# Remote mode — connect to the LND container
lnd-mode=remote
remote.lnd.rpcserver=lnd:10009
remote.lnd.tlscertpath=/lnd/tls.cert
remote.lnd.macaroonpath=/lnd/data/chain/bitcoin/mainnet/admin.macaroon

# Network
network=mainnet

# Web UI
httpslisten=0.0.0.0:8443
uipassword=${LIT_UI_PASSWORD}
EOF

echo "Generated /root/.lit/lit.conf"

# Execute litd
exec litd "$@"
