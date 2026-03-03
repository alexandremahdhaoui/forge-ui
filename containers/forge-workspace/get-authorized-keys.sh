#!/bin/sh
# Fetch dynamically registered SSH public keys from the wss-proxy.
# Called by sshd AuthorizedKeysCommand on each authentication attempt.
# Falls back silently so static keys in AuthorizedKeysFile still work
# if the proxy is unreachable.
#
# Read workspace name from file (written by entrypoint.sh) because sshd
# does not pass container environment variables to AuthorizedKeysCommand.
WORKSPACE="$(cat /etc/forge-workspace-name 2>/dev/null || echo default)"
wget -qO- "http://forge-wss-proxy:8080/internal/authorized-keys/${WORKSPACE}" 2>/dev/null || true
