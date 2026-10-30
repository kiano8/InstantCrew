#!/usr/bin/env bash
# Build from the git checkout and publish into /opt/instantcrew, which the sandboxed
# service runs from. Lives at ~/deploy.sh on the droplet; CI runs it over SSH.
set -euo pipefail
export PATH=$PATH:/usr/local/go/bin
SRC=/home/instantcrew/DEPLOYMENT/InstantCrew
APP=/opt/instantcrew
cd "$SRC"

# Dump Postgres before the new binary runs its migrations. set -e stops the deploy
# if the dump fails, so we never migrate without a fresh backup.
# Needs ~/.pgpass (chmod 600) with: 127.0.0.1:5432:<db>:<user>:<password>
STAMP=$(date +%Y%m%d-%H%M%S)
mkdir -p ~/backups
pg_dump -w -h 127.0.0.1 -U "${DB_USER:-instantcrew}" -Fc "${DB_NAME:-instantcrew}" > ~/backups/db-$STAMP.dump.tmp
mv ~/backups/db-$STAMP.dump.tmp ~/backups/db-$STAMP.dump

git fetch origin main
git reset --hard origin/main

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

# Keep last 10 backups (sort by name = timestamp, not mtime)
ls -d ~/backups/db-*.dump | sort | head -n -10 | xargs -r rm -f
echo "Deploy OK"
