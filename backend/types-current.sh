#!/bin/sh
# Fails if the web's generated protocol types are out of date with the go code.
set -e
files="../web/interfaces/holdem.ts ../web/interfaces/table.ts ../web/interfaces/server.ts"
before=$(cat $files | shasum)
go tool tygo generate
if [ "$(cat $files | shasum)" != "$before" ]; then
    echo "web/interfaces were stale and have been regenerated; commit them (just types)"
    exit 1
fi
