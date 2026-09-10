#!/bin/bash
set -e

# Everything below is configured in /home/fulcrum/.fulcrum/fulcrum.conf
# which is baked into the image. If you want to override settings, mount a
# custom conf file over it, e.g.:
#   - ./fulcrum/fulcrum.conf:/home/fulcrum/.fulcrum/fulcrum.conf

# Shared logs volume: make sure our log file exists and is world-readable so
# the dashboard (a separate container/uid) can tail it.
mkdir -p /logs
touch /logs/fulcrum.log
chmod 0644 /logs/fulcrum.log

# Run Fulcrum, teeing its output to the shared log file for the dashboard
# while keeping it visible in `docker compose logs`.
exec "$@" 2>&1 | tee -a /logs/fulcrum.log