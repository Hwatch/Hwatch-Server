#!/bin/sh
set -eu
mkdir -p /data /media
touch /data/aria2.session
aria2c \
  --enable-rpc=true \
  --rpc-listen-all=false \
  --rpc-listen-port=6800 \
  --continue=true \
  --input-file=/data/aria2.session \
  --save-session=/data/aria2.session \
  --save-session-interval=30 \
  --auto-file-renaming=false \
  --allow-overwrite=true \
  --file-allocation=none \
  --max-concurrent-downloads=2 \
  --daemon=true
exec /usr/local/bin/r34watch
