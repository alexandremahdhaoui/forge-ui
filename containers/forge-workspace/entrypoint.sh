#!/bin/sh
# Generate host keys if not mounted from Secret
if [ ! -f /etc/ssh/ssh_host_ed25519_key ]; then
    ssh-keygen -A
fi
# Persist WORKSPACE_NAME for AuthorizedKeysCommand (sshd does not pass env vars).
echo "${WORKSPACE_NAME:-default}" > /etc/forge-workspace-name
# Start sshd in foreground
exec /usr/sbin/sshd -D -e
