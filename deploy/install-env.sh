#!/usr/bin/env bash
# Move the production .env out of the app directory into a root-only file that
# systemd loads before dropping to the unprivileged service user.
# Usage: sudo ./install-env.sh /path/to/.env [/path/to/client_secret_*.json]
set -euo pipefail

SRC="${1:?usage: sudo ./install-env.sh /path/to/.env}"
DEST=/etc/instantcrew/instantcrew.env

die() { echo "error: $*" >&2; exit 1; }
[[ $EUID -eq 0 ]] || die "run with sudo"
[[ -f $SRC ]] || die "$SRC not found"
# systemd EnvironmentFile does not understand shell syntax.
! grep -qE '^\s*export\s' "$SRC" || die "remove 'export ' prefixes from $SRC"

install -d -m 700 -o root -g root /etc/instantcrew
install -m 600 -o root -g root "$SRC" "$DEST"
shred -u "$SRC"
echo "Installed $DEST (root:root 600) and shredded $SRC"

# Optional: the Google OAuth client_secret_*.json. The service user cannot read
# /etc/instantcrew, so systemd hands it over as a credential at start-up.
if [[ -n ${2:-} ]]; then
  [[ -f $2 ]] || die "$2 not found"
  install -m 600 -o root -g root "$2" /etc/instantcrew/google-oauth.json
  shred -u "$2"
  install -d -m 755 /etc/systemd/system/instantcrew.service.d
  cat > /etc/systemd/system/instantcrew.service.d/google-oauth.conf <<'EOF'
[Service]
LoadCredential=google-oauth.json:/etc/instantcrew/google-oauth.json
Environment=GOOGLE_OAUTH_CREDENTIALS=%d/google-oauth.json
EOF
  systemctl daemon-reload
  echo "Installed /etc/instantcrew/google-oauth.json and shredded $2"
fi

value() { sed -n "s/^$1=//p" "$DEST" | tail -1 | tr -d '"'"'"; }
[[ $(value ENV) == prod ]] || echo "warning: ENV is not 'prod'"
[[ $(value COOKIE_SECURE) == true ]] || echo "warning: COOKIE_SECURE is not 'true' (session cookies sent over plain HTTP)"
[[ $(value HOST) == 127.0.0.1 ]] || echo "warning: HOST is not 127.0.0.1; set it if a reverse proxy fronts the app"
for k in DB_SSLMODE VERIFIER_DB_SSLMODE; do
  [[ $(value $k) != disable ]] || echo "note: $k=disable is only OK when Postgres is on this same host"
done
