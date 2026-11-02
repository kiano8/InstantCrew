#!/usr/bin/env bash
# Back up, test, build and publish into /opt/instantcrew, which the sandboxed service
# runs from. Lives at ~/deploy.sh on the droplet and replaces itself with the repo's
# copy after each successful deploy, so the two can't drift apart.
set -euo pipefail
umask 077 # backups hold personal data; only this user may read them
export PATH=$PATH:/usr/local/go/bin
SRC=/home/instantcrew/DEPLOYMENT/InstantCrew
APP=/opt/instantcrew
BACKUPS=~/backups
cd "$SRC"

# Optional settings: DB_USER, DB_NAME, APP_PORT, TEST_DATABASE_URL.
if [ -f ~/.instantcrew-deploy.env ]; then
  set -a
  . ~/.instantcrew-deploy.env
  set +a
fi

# Dump Postgres before the new binary runs its migrations. set -e stops the deploy
# if the dump fails, so we never migrate without a fresh backup.
# Needs ~/.pgpass (chmod 600) with: 127.0.0.1:5432:<db>:<user>:<password>
STAMP=$(date +%Y%m%d-%H%M%S)
mkdir -p "$BACKUPS"
chmod 700 "$BACKUPS"
pg_dump -w -h 127.0.0.1 -U "${DB_USER:-instantcrew}" -Fc "${DB_NAME:-instantcrew}" > "$BACKUPS/db-$STAMP.dump.tmp"
mv "$BACKUPS/db-$STAMP.dump.tmp" "$BACKUPS/db-$STAMP.dump"

# Receipts live outside the database. A failure here warns instead of stopping a
# deploy: this user may not be able to read every upload, and they aren't at risk from one.
if tar -C "$APP" -czf "$BACKUPS/uploads-$STAMP.tgz.tmp" uploads; then
  mv "$BACKUPS/uploads-$STAMP.tgz.tmp" "$BACKUPS/uploads-$STAMP.tgz"
else
  rm -f "$BACKUPS/uploads-$STAMP.tgz.tmp"
  echo "WARNING: could not back up $APP/uploads (is this user in the instantcrew-app group?)" >&2
fi

# Anything kept in the checkout's data/ folder (git reset leaves it alone).
if [ -n "$(ls -A data 2>/dev/null)" ]; then
  cp -a data "$BACKUPS/data-$STAMP"
fi

git fetch origin main
git reset --hard origin/main

# Broken code stops here, before anything live changes. Only the app's own packages:
# ./... also compiles the verifier's SQLite library, which a 1 GB droplet can't handle.
if [ -n "${TEST_DATABASE_URL:-}" ]; then
  go vet ./cmd/... ./internal/...
  go test -p 1 ./cmd/... ./internal/...
else
  echo "WARNING: TEST_DATABASE_URL is not set, so tests are skipped" >&2
fi

go build -o instantcrew.new ./cmd/server

# APP and its subdirectories are setgid instantcrew-app, so new files land in that group
# and the service can read them. Never touch uploads/ or write a .env here.
install -m 750 instantcrew.new "$APP/instantcrew.new"
mv "$APP/instantcrew.new" "$APP/instantcrew"
rm instantcrew.new
rsync -rlt --delete --chmod=D2750,F640 css js images "$APP/"
rsync -lt --chmod=F640 ./*.html sw.js manifest.json "$APP/"

sudo systemctl restart instantcrew
sleep 2
systemctl is-active --quiet instantcrew || { echo "Service failed to start"; sudo systemctl status instantcrew --no-pager; exit 1; }

# The PWA needs these exact files; a running service doesn't prove they were published.
for p in / /manifest.json /sw.js /js/pwa-install.js /images/icon-192.png; do
  curl -fsS -o /dev/null "http://127.0.0.1:${APP_PORT:-3000}$p" || { echo "Smoke test failed: $p"; exit 1; }
done

# Keep the newest backups (sort by name = timestamp, not mtime). Receipts are the big ones.
find "$BACKUPS" -maxdepth 1 -name 'db-2*.dump' | sort | head -n -10 | xargs -r rm -f
find "$BACKUPS" -maxdepth 1 -name 'uploads-*.tgz' | sort | head -n -3 | xargs -r rm -f
find "$BACKUPS" -maxdepth 1 -name 'data-*' | sort | head -n -10 | xargs -r rm -rf

# mv swaps in a new file, so the copy bash is reading right now stays intact.
install -m 700 deploy/deploy.sh ~/deploy.sh.new
mv ~/deploy.sh.new ~/deploy.sh
echo "Deploy OK"
exit 0
