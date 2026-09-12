#!/bin/bash
set -e

# Everything below is configured in /home/fulcrum/.fulcrum/fulcrum.conf
# which is baked into the image. If you want to override settings, mount a
# custom conf file over it, e.g.:
#   - ./fulcrum/fulcrum.conf:/home/fulcrum/.fulcrum/fulcrum.conf

# This entrypoint runs as root (see Dockerfile) so we can fix ownership of the
# host bind mounts before dropping to the fulcrum user. When `docker compose
# up` is run as root (common in production), Docker creates these host dirs
# root-owned, and the non-root fulcrum user can't write them.
mkdir -p /logs
chown fulcrum:fulcrum /logs 2>/dev/null || true
chown fulcrum:fulcrum /home/fulcrum/db 2>/dev/null || true

# Shared logs volume: make sure our log file exists and is world-readable so
# the dashboard (a separate container/uid) can tail it. Keep it owned by the
# fulcrum user so `tee -a` below can append to it after we drop privileges.
touch /logs/fulcrum.log
chown fulcrum:fulcrum /logs/fulcrum.log 2>/dev/null || true
chmod 0644 /logs/fulcrum.log

# Drop to the fulcrum user and run, teeing output to the shared log file for
# the dashboard while keeping it visible in `docker compose logs`. setpriv is
# part of util-linux (in the base image) — no extra package needed.
exec setpriv --reuid=fulcrum --regid=fulcrum --init-groups "$@" 2>&1 | tee -a /logs/fulcrum.log