#!/usr/bin/env bash
# Remove the macOS Forgejo runner LaunchAgent, binary and state.
# The server-side runner entry is NOT deleted — remove it in the Forgejo UI
# (Site administration → Actions → Runners) or via the API afterwards.
set -euo pipefail

STATE_DIR="${STATE_DIR:-$HOME/.local/share/forgejo-runner}"
PLIST="$HOME/Library/LaunchAgents/com.rezuscloud.forgejo-runner.plist"

launchctl bootout "gui/$(id -u)" "$PLIST" 2>/dev/null || true
rm -f "$PLIST"
rm -f "$HOME/.local/bin/forgejo-runner"

echo "Removed agent + binary."
read -r -p "Also delete state ($STATE_DIR: config, registration, cache, logs)? [y/N] " ans
case "$ans" in
  y|Y) rm -rf "$STATE_DIR"; echo "State removed." ;;
  *)   echo "State kept at $STATE_DIR" ;;
esac
echo "Remember to delete the runner entry server-side."
