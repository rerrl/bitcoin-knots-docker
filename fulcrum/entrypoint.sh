#!/bin/bash
set -e

# Everything below is configured in /home/fulcrum/.fulcrum/fulcrum.conf
# which is baked into the image. If you want to override settings, mount a
# custom conf file over it, e.g.:
#   - ./fulcrum/fulcrum.conf:/home/fulcrum/.fulcrum/fulcrum.conf

exec "$@"