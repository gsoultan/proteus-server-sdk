#!/usr/bin/env sh
# Copies the API definition from a proteus-server checkout and regenerates
# both clients. Usage: scripts/sync-proto.sh [path-to-proteus-server]
set -eu
src="${1:-../proteus-server}/proto/proteus/v1"
cd "$(dirname "$0")/.."
cp "$src"/*.proto proto/proteus/v1/
buf lint
buf generate
