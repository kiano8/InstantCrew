#!/usr/bin/env bash
# Create the unprivileged instantcrew-app user, install the systemd unit, and let the
# deploy user restart the service (and nothing else) without a password.
# Usage: sudo ./install-service.sh   (run from the deploy/ directory)
set -euo pipefail

APP_DIR=/opt/instantcrew
DEPLOY_USER="${DEPLOY_USER:-${SUDO_USER:-}}"

die() { echo "error: $*" >&2; exit 1; }
[[ $EUID -eq 0 ]] || die "run with sudo"
[[ -n $DEPLOY_USER && $DEPLOY_USER != root ]] || die "set DEPLOY_USER to the account deploy.sh runs as"
[[ -f /etc/instantcrew/instantcrew.env ]] || die "run install-env.sh first"

id instantcrew-app &>/dev/null || useradd --system --home-dir "$APP_DIR" --shell /usr/sbin/nologin instantcrew-app

# Code is owned by the deploy user and only readable by the service; uploads are the reverse.
install -d -m 750 -o "$DEPLOY_USER" -g instantcrew-app "$APP_DIR"
install -d -m 750 -o instantcrew-app -g instantcrew-app "$APP_DIR/uploads"

install -m 644 "$(dirname "$0")/instantcrew.service" /etc/systemd/system/instantcrew.service

sudoers=/etc/sudoers.d/instantcrew-deploy
echo "$DEPLOY_USER ALL=(root) NOPASSWD: /usr/bin/systemctl restart instantcrew, /usr/bin/systemctl status instantcrew" > "$sudoers.tmp"
chmod 440 "$sudoers.tmp"
visudo -cf "$sudoers.tmp" && mv "$sudoers.tmp" "$sudoers"

systemctl daemon-reload
systemctl enable instantcrew
echo "Installed. Put the built binary and static files in $APP_DIR, then: sudo systemctl restart instantcrew"
echo "Sandbox score (lower is better): systemd-analyze security instantcrew"
