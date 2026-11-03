#!/usr/bin/env bash
# Starts a headless X11 desktop for the live tests: Xvfb + openbox + a D-Bus session with the
# AT-SPI accessibility bus, and exports DISPLAY / DBUS_SESSION_BUS_ADDRESS for later steps.
set -euo pipefail
export DISPLAY=:99
Xvfb :99 -screen 0 1280x800x24 -nolisten tcp > /tmp/xvfb.log 2>&1 &
for i in $(seq 1 50); do xdpyinfo > /dev/null 2>&1 && break; sleep 0.1; done
eval "$(dbus-launch --sh-syntax)"
# toolkit accessibility on, so GTK apps load the AT-SPI bridge and expose their trees
gsettings set org.gnome.desktop.interface toolkit-accessibility true || true
dbus-send --session --dest=org.a11y.Bus --print-reply /org/a11y/bus org.freedesktop.DBus.Properties.Set \
  string:org.a11y.Status string:IsEnabled variant:boolean:true > /dev/null || true
openbox > /tmp/openbox.log 2>&1 &
sleep 1
{
  echo "DISPLAY=$DISPLAY"
  echo "DBUS_SESSION_BUS_ADDRESS=$DBUS_SESSION_BUS_ADDRESS"
  echo "GTK_MODULES=gail:atk-bridge"
  echo "NO_AT_BRIDGE=0"
  echo "XDG_STATE_HOME=$RUNNER_TEMP/state"
} >> "$GITHUB_ENV"
