#!/usr/bin/env sh
# Groundbreak launcher (macOS / Linux): get the latest version, then start the game.
cd "$(dirname "$0")" || exit 1
echo "Checking for updates..."
git pull --ff-only || echo "Could not update right now - starting the version you already have."
npm install --no-audit --no-fund --loglevel=error && npm start
