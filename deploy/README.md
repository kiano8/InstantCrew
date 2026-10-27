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

`~/deploy.sh` should copy the new binary and static files into `/opt/instantcrew`
and run `sudo systemctl restart instantcrew`. It must never write a `.env` there.

## 5. HTTPS — Caddy

```sh
sudo apt install caddy
sudo cp Caddyfile /etc/caddy/Caddyfile   # set your domain first
sudo systemctl reload caddy
```

The app only listens on `127.0.0.1:8080` (`HOST=127.0.0.1`), so all traffic goes
through Caddy's TLS. Keep `COOKIE_SECURE=true` and `ENV=prod`.

## 6. Postgres

- In `postgresql.conf`: `listen_addresses = 'localhost'`.
- In `pg_hba.conf`: only `host … 127.0.0.1/32 scram-sha-256` lines; no `trust`.
- Give `instantcrew` its own DB role that owns only its database (not a superuser).
- Nightly backup, kept off the droplet:
  `pg_dump -Fc instantcrew > /var/backups/instantcrew-$(date +%F).dump` → DO Spaces.

## Verify

```sh
sudo ss -ltnp                       # only :30022, :80, :443 on 0.0.0.0; 8080/5432 on 127.0.0.1
sudo ufw status verbose
sudo fail2ban-client status sshd
sudo -u instantcrew cat /etc/instantcrew/instantcrew.env   # must say Permission denied
curl -I https://<domain>/.env       # 404
curl -I https://<domain>/uploads/   # 404 (no directory listing)
```
