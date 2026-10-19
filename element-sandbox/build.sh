#!/usr/bin/env sh
# Builds a portable Windows .exe (no installer, no DLLs) from any OS with Go installed.
set -e
cd "$(dirname "$0")"
mkdir -p dist
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H=windowsgui -s -w" -o dist/ElementSandbox.exe .
echo "Built dist/ElementSandbox.exe"
