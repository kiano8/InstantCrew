#!/usr/bin/env bash
# Baseline hardening for the InstantCrew VPS (Ubuntu 22.04+/Debian 12+).
# Usage: sudo SSH_PORT=30022 ./harden.sh
#
# Keep your current SSH session open until you have confirmed a NEW login works.
set -euo pipefail

SSH_PORT="${SSH_PORT:-30022}"
LOGIN_USER="${SUDO_USER:-}"

die() { echo "error: $*" >&2; exit 1; }
[[ $EUID -eq 0 ]] || die "run with sudo"
[[ -n $LOGIN_USER && $LOGIN_USER != root ]] || die "run via sudo from your normal (non-root) user"

# Refuse to turn off password logins unless key login is already set up, or we lock ourselves out.
home=$(getent passwd "$LOGIN_USER" | cut -d: -f6)
[[ -s $home/.ssh/authorized_keys ]] || die "$home/.ssh/authorized_keys is empty; add your SSH key first"

echo "==> Packages and automatic security updates"
export DEBIAN_FRONTEND=noninteractive
apt-get update -q
apt-get install -yq ufw fail2ban python3-systemd unattended-upgrades
cat > /etc/apt/apt.conf.d/20auto-upgrades <<'EOF'
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
EOF

echo "==> Firewall: allow SSH ($SSH_PORT), HTTP, HTTPS; deny everything else"
ufw default deny incoming
ufw default allow outgoing
ufw limit "$SSH_PORT/tcp"
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable

echo "==> SSH: keys only, no root, port $SSH_PORT"
# sshd uses the first value it sees, so this must sort before cloud-init's 50-cloud-init.conf.
cat > /etc/ssh/sshd_config.d/00-hardening.conf <<EOF
Port $SSH_PORT
PermitRootLogin no
PasswordAuthentication no
KbdInteractiveAuthentication no
PubkeyAuthentication yes
PermitEmptyPasswords no
MaxAuthTries 3
LoginGraceTime 30
X11Forwarding no
AllowAgentForwarding no
ClientAliveInterval 300
ClientAliveCountMax 2
EOF
sshd -t || die "sshd config invalid; not restarting"
systemctl daemon-reload
if systemctl is-active --quiet ssh.socket; then systemctl restart ssh.socket; fi
systemctl restart ssh

echo "==> fail2ban for sshd"
cat > /etc/fail2ban/jail.d/sshd.local <<EOF
[sshd]
enabled  = true
port     = $SSH_PORT
backend  = systemd
maxretry = 5
findtime = 10m
bantime  = 1h
EOF
systemctl enable --now fail2ban
systemctl restart fail2ban

echo "==> Kernel network hardening"
cat > /etc/sysctl.d/99-hardening.conf <<'EOF'
net.ipv4.tcp_syncookies = 1
net.ipv4.conf.all.rp_filter = 1
net.ipv4.conf.default.rp_filter = 1
net.ipv4.conf.all.accept_redirects = 0
net.ipv4.conf.default.accept_redirects = 0
net.ipv6.conf.all.accept_redirects = 0
net.ipv6.conf.default.accept_redirects = 0
net.ipv4.conf.all.send_redirects = 0
net.ipv4.conf.all.accept_source_route = 0
net.ipv6.conf.all.accept_source_route = 0
net.ipv4.icmp_echo_ignore_broadcasts = 1
net.ipv4.conf.all.log_martians = 1
kernel.kptr_restrict = 2
kernel.dmesg_restrict = 1
fs.protected_hardlinks = 1
fs.protected_symlinks = 1
EOF
sysctl --system >/dev/null

echo
echo "Done. Ports listening on non-loopback addresses (Postgres should NOT be here):"
ss -ltnH | awk '$4 !~ /^(127\.|\[::1\])/ {print "  " $4}'
echo
echo "Now, from another terminal, confirm you can still log in:  ssh -p $SSH_PORT $LOGIN_USER@<host>"
