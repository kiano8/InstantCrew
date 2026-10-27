#!/usr/bin/env bash
# Build from the git checkout and publish into /opt/instantcrew, which the sandboxed
# service runs from. Lives at ~/deploy.sh on the droplet; CI runs it over SSH.
set -euo pipefail
export PATH=$PATH:/usr/local/go/bin
SRC=/home/instantcrew/DEPLOYMENT/InstantCrew
APP=/opt/instantcrew
cd "$SRC"

# Back up data (db + wal/shm files) before touching git
STAMP=$(date +%Y%m%d-%H%M%S)
BACKUP=~/backups/data-$STAMP
mkdir -p ~/backups data
cp -a data "$BACKUP"
touch "$BACKUP"

git fetch origin main
git reset --hard origin/main

# Restore live data in case git overwrote or deleted it
mkdir -p data
cp -a "$BACKUP/." data/

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
ls -d ~/backups/data-* | sort | head -n -10 | xargs -r rm -rf
echo "Deploy OK"
