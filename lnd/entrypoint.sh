#!/bin/sh
set -e

# Everything is configured in /home/lnd/lnd.conf, which is baked into the image
# and passed via --configfile. To override settings, mount a custom conf over
# that path, e.g.:
#   - ./lnd/lnd.conf:/home/lnd/lnd.conf:ro

# The data dir is a bind mount and inherits host permissions. If the container
# happens to run as root (rather than the default non-root lnd user), make it
# writable by the lnd user first.
if [ "$(id -u)" = "0" ]; then
    chown -R lnd:lnd /home/lnd/.lnd 2>/dev/null || true
fi

exec "$@"