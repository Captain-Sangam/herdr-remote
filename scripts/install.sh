#!/bin/sh
# Installs herdr-remote for this user (docs/deployment.md):
#   ~/.local/bin/herdr-remoted                    the daemon, built from daemon/
#   ~/Library/LaunchAgents/dev.herdr-remote.plist  runs `herdr-remoted serve`
#   ~/.config/herdr-app/mods/herdr-remote/         the overlay mod
# It changes no other mod and no Herdr setting.
set -eu

REPO=$(cd "$(dirname "$0")/.." && pwd)
BIN="$HOME/.local/bin/herdr-remoted"
STATE="$HOME/Library/Application Support/herdr-remote"
PLIST="$HOME/Library/LaunchAgents/dev.herdr-remote.plist"
LABEL="dev.herdr-remote"
MOD="${HERDR_APP_HOME:-$HOME/.config/herdr-app}/mods/herdr-remote"
MARKER=".herdr-remote-owned"

mkdir -p "$HOME/.local/bin" "$HOME/Library/LaunchAgents" "$STATE"
chmod 700 "$STATE"

echo "Building $BIN"
(cd "$REPO/daemon" && go build -o "$BIN" .)

cat >"$PLIST" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>$LABEL</string>
  <key>ProgramArguments</key>
  <array>
    <string>$BIN</string>
    <string>serve</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>$STATE/daemon.log</string>
  <key>StandardErrorPath</key><string>$STATE/daemon.log</string>
</dict>
</plist>
PLIST

launchctl bootout "gui/$(id -u)/$LABEL" 2>/dev/null || true
# bootout returns before the service is gone, and bootstrap fails with EIO until it is.
i=0
while [ $i -lt 50 ] && launchctl print "gui/$(id -u)/$LABEL" >/dev/null 2>&1; do
  sleep 0.1
  i=$((i + 1))
done
launchctl bootstrap "gui/$(id -u)" "$PLIST"
echo "LaunchAgent $LABEL loaded"

if [ -d "$MOD" ] && [ ! -f "$MOD/$MARKER" ]; then
  echo "error: $MOD exists and was not installed by this script; move it away first" >&2
  exit 1
fi
mkdir -p "$MOD"
sed "s|__HERDR_REMOTED__|$BIN|g" "$REPO/mod/herdr-remote/main.js" >"$MOD/main.js.tmp"
mv "$MOD/main.js.tmp" "$MOD/main.js"
cp "$REPO/mod/herdr-remote/mod.json" "$MOD/mod.json"
: >"$MOD/$MARKER"
echo "Mod installed at $MOD"
herdr-app check "$MOD"

cat <<NOTE

Done.
- Open the overlay with: herdr-app do mod.open --name herdr-remote
- macOS may ask once to allow incoming connections for herdr-remoted; click Allow.
- For full Codex transcripts, set Herdr's Codex launcher to ["codex", "--no-daemon"]
  (Herdr settings, new-menu.agents). This script does not change Herdr settings.
NOTE
