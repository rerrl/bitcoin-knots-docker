#!/bin/bash
set -e

# Always use the bitcoin.conf from the repository if it exists
if [ -f /tmp/bitcoin.conf ]; then
    echo "Using custom bitcoin.conf from repository..."
    cp /tmp/bitcoin.conf /home/bitcoin/.bitcoin/bitcoin.conf
elif [ ! -f /home/bitcoin/.bitcoin/bitcoin.conf ]; then
    echo "ERROR: No bitcoin.conf file found!"
    echo "Please create a bitcoin.conf file in the repository root directory."
    echo "The file should contain your Bitcoin Knots configuration."
    exit 1
fi

# Execute the command
exec "$@"
