#!/usr/bin/env bash
# Nightly encrypted off-site backup of Postgres and receipts. Runs as root from cron.
# Only the public key is on the droplet, so a hacked server can't read old backups.
# Settings in /etc/instantcrew/backup.env (chmod 600):
#   AGE_RECIPIENT=age1...   DB_NAME=instantcrew   RCLONE_DEST=spaces:your-bucket/instantcrew
set -euo pipefail
umask 077
. /etc/instantcrew/backup.env
: "${AGE_RECIPIENT:?}" "${DB_NAME:?}" "${RCLONE_DEST:?}"

DIR=/var/backups/instantcrew
STAMP=$(date +%F)
mkdir -p "$DIR"
sudo -u postgres pg_dump -Fc "$DB_NAME" | age -r "$AGE_RECIPIENT" > "$DIR/db-$STAMP.dump.age.tmp"
mv "$DIR/db-$STAMP.dump.age.tmp" "$DIR/db-$STAMP.dump.age"
tar -C /opt/instantcrew -cz uploads | age -r "$AGE_RECIPIENT" > "$DIR/uploads-$STAMP.tgz.age.tmp"
mv "$DIR/uploads-$STAMP.tgz.age.tmp" "$DIR/uploads-$STAMP.tgz.age"
rclone copy "$DIR" "$RCLONE_DEST"
find "$DIR" -type f -mtime +7 -delete
