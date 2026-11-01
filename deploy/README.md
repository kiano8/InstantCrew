# Production hardening (DigitalOcean droplet)

Run these once, in order, on the droplet. Keep one SSH session open the whole time
so a mistake in step 2 can't lock you out.

## 0. In the DigitalOcean console

- **Networking → Firewalls**: create a Cloud Firewall for the droplet with inbound
  TCP `30022` (ideally only from your IP), `80`, `443`. Nothing else. This sits in
  front of `ufw`, so a mistake on the box still can't expose Postgres.
- **Backups**: turn on droplet backups (or weekly snapshots).
- **Account**: enable 2FA on the DO account; remove unused SSH keys and API tokens.

## 1. A non-root sudo user with your key

The login/deploy user is `instantcrew` (already exists, in the `sudo` group, with
your SSH key). If you ever rebuild the droplet, create it as root first:

```sh
adduser instantcrew && usermod -aG sudo instantcrew
rsync --archive --chown=instantcrew:instantcrew ~/.ssh /home/instantcrew
```

Log in as `instantcrew` from here on. Copy this `deploy/` folder to the droplet.

## 2. OS hardening — `sudo SSH_PORT=30022 ./harden.sh`

Auto security updates, `ufw` (30022/80/443 only), keys-only SSH with no root login,
fail2ban, kernel network sysctls. Before closing your old session, test
`ssh -p 30022 instantcrew@<droplet-ip>` from a second terminal.

## 3. Secrets — `sudo ./install-env.sh /opt/instantcrew/.env [client_secret_*.json]`

Moves `.env` to `/etc/instantcrew/instantcrew.env` (`root:root 600`, folder `700`)
and shreds the original. systemd reads it as root and passes the values to the app
as environment variables, so:

- the app user `instantcrew-app` cannot read the file, even if the app is compromised;
- there is no `.env` in the app directory for the web server or a bad deploy to expose;
- the Google OAuth JSON (optional second argument) gets the same treatment and is
  handed to the app through systemd `LoadCredential`.

To change a secret later: `sudo -e /etc/instantcrew/instantcrew.env && sudo systemctl restart instantcrew`.

Set strong values (e.g. `openssl rand -base64 32`) for `DB_PASSWORD`, `ADMIN_PASSWORD`,
`ACCOUNTING_PASSWORD`. If the old `.env` was ever world-readable, copied around or
pasted anywhere, **rotate every value in it**: DB passwords (`ALTER USER … PASSWORD`),
admin passwords, and the Google client secret in Google Cloud Console.

## 4. Run the app as a sandboxed service — `sudo ./install-service.sh`

Creates the `instantcrew-app` system user (no shell, no login), installs
`instantcrew.service` (read-only filesystem except `uploads/`, no capabilities,
syscall filter) and lets `instantcrew` restart only this service without a password.
Check the sandbox with `systemd-analyze security instantcrew`.

`deploy.sh` backs up the database, runs the tests, publishes into `/opt/instantcrew`
and restarts the service. It must never write a `.env` there. It stops before touching
the live app if the backup or tests fail, and updates `~/deploy.sh` from the repo after
each success. Settings go in `~/.instantcrew-deploy.env` (chmod 600): `DB_USER`,
`DB_NAME`, `APP_PORT` and `TEST_DATABASE_URL`. Point that last one at its own database,
never the live one; without it the database tests are skipped.

## 5. HTTPS — Cloudflare and nginx

Traffic goes Cloudflare → nginx on the droplet → the app on `127.0.0.1:3000`. The live
config is `/etc/nginx/sites-enabled/instant-crew.com.conf`. Keep backup copies of it
out of `sites-enabled`, because nginx may load them too.

Outside `ENV=dev` the app only listens on `127.0.0.1` unless `HOST` says otherwise.
Keep `COOKIE_SECURE=true`. In the DigitalOcean firewall, limit ports 80 and 443 to
[Cloudflare's IP ranges](https://www.cloudflare.com/ips/), so login limits can't be
dodged by reaching the droplet directly.

## 6. Postgres

- In `postgresql.conf`: `listen_addresses = 'localhost'`.
- In `pg_hba.conf`: only `host … 127.0.0.1/32 scram-sha-256` lines; no `trust`.
- Give `instantcrew` its own DB role that owns only its database (not a superuser).

## 7. Backups

`deploy.sh` keeps a dump on the droplet from before each deploy. That doesn't survive
losing the droplet, so also run `backup.sh` nightly: it encrypts the database and
receipts and copies them off the server. See the top of the script for its settings.

```sh
sudo apt install age rclone && sudo rclone config     # create the off-site remote
sudo install -m 700 backup.sh /usr/local/sbin/instantcrew-backup
sudo /usr/local/sbin/instantcrew-backup               # check the files reach the bucket
echo '30 18 * * * root /usr/local/sbin/instantcrew-backup' | sudo tee /etc/cron.d/instantcrew-backup
```

Keep the `age` private key off the droplet. To test a restore on your own machine:
`age -d -i key.txt db-DATE.dump.age | pg_restore --no-owner -d instantcrew_restore`.

## Monitoring

Sign in as admin and open `/telemetry`: traffic, errors, response times, uptime and
activity, stored in Postgres for 35 days. Uptime comes from a per-minute heartbeat, so
outages shorter than about 2 minutes don't appear, and an outage only appears once the
app is back. Nothing can alert you while the droplet itself is down.

## Verify

```sh
sudo ss -ltnp                       # only :30022, :80, :443 on 0.0.0.0; 3000/5432 on 127.0.0.1
sudo ufw status verbose
sudo fail2ban-client status sshd
sudo -u instantcrew cat /etc/instantcrew/instantcrew.env   # must say Permission denied
curl -I https://<domain>/.env       # 404
curl -I https://<domain>/uploads/   # 404 (no directory listing; a receipt without login is 401)
```
