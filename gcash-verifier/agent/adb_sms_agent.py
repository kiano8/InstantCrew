#!/usr/bin/env python3
"""
GCash SMS agent: ADB -> verifier.

Watches the SMS inbox of an Android phone over ADB (USB or Wireless Debugging)
and forwards ONLY messages from the GCash sender to the verifier's webhook.
The verifier matches the exact amount to a pending payment, marks it paid, and
calls your website back (PAYMENT_CALLBACK_URL).

Nothing is installed on the phone, so there is no app for Android to block, and
SMS text never leaves your machine unless you point --verifier-url elsewhere.

    python3 gcash-verifier/agent/adb_sms_agent.py --list-senders
    SIGNING_KEY=... python3 gcash-verifier/agent/adb_sms_agent.py --sender GCash
"""
from __future__ import annotations

import hashlib
import hmac
import json
import re
import subprocess
import sys
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

import click

DEFAULT_STATE = Path(__file__).with_name(".agent_state.json")


class AdbError(Exception):
    pass


class ForwardError(Exception):
    pass


# --- ADB helpers -------------------------------------------------------------

def run_adb(args, device_id=None, timeout=25):
    cmd = ["adb"] + (["-s", device_id] if device_id else []) + args
    try:
        res = subprocess.run(cmd, capture_output=True, text=True, encoding="utf-8",
                             errors="replace", timeout=timeout)
    except FileNotFoundError:
        raise AdbError("adb not found. Install Android platform-tools and make sure `adb` is on your PATH.")
    except subprocess.TimeoutExpired:
        raise AdbError(f"adb timed out: {' '.join(cmd)}")
    return res.stdout, res.stderr, res.returncode


def list_devices():
    """Returns (ready_device_ids, [(id, state), ...] for devices that are not usable)."""
    out, _, _ = run_adb(["devices", "-l"])
    ready, bad = [], []
    for line in out.splitlines()[1:]:
        parts = line.split()
        if len(parts) < 2:
            continue
        if parts[1] == "device":
            ready.append(parts[0])
        else:
            bad.append((parts[0], parts[1]))
    return ready, bad


ROW_SPLIT = re.compile(r"(?m)^Row: \d+ ")
ROW_FIELDS = re.compile(r"_id=(\d+), address=(.*?), date=(\d+), body=(.*)\Z", re.S)


def fetch_inbox(device_id):
    """All inbox messages as dicts: _id, address, date (ms), body.
    body is requested last in the projection so multi-line bodies parse safely."""
    out, err, code = run_adb(
        ["shell", "content", "query", "--uri", "content://sms/inbox",
         "--projection", "_id:address:date:body"], device_id)
    if code != 0:
        raise AdbError((err or out).strip() or f"adb exited with code {code}")
    out = out.replace("\r", "")
    if "No result found" in out:
        return []
    msgs = []
    for block in ROW_SPLIT.split(out):
        m = ROW_FIELDS.match(block.strip())
        if m:
            msgs.append({"_id": int(m[1]), "address": m[2].strip(),
                         "date": int(m[3]), "body": m[4].strip()})
    if not msgs and out.strip():
        raise AdbError("unexpected output from the SMS provider: " + out.strip()[:200])
    return msgs


def norm(sender: str) -> str:
    return re.sub(r"[\s\-()+]", "", sender or "").upper()


# --- state -------------------------------------------------------------------

def load_state(path: Path) -> dict:
    try:
        return json.loads(path.read_text())
    except (OSError, ValueError):
        return {}


def save_state(path: Path, state: dict) -> None:
    tmp = path.with_suffix(".tmp")
    tmp.write_text(json.dumps(state))
    tmp.replace(path)


# --- forwarding --------------------------------------------------------------

def forward(url: str, secret: str, msg: dict, timeout: int = 10) -> dict:
    """POST one SMS to the verifier in the same envelope SMSGate uses, signed with
    hex(HMAC-SHA256(secret, body + X-Timestamp))."""
    date_ms = msg["date"] if msg["date"] > 10**11 else msg["date"] * 1000
    received_at = datetime.fromtimestamp(date_ms / 1000, tz=timezone.utc).isoformat()
    body = json.dumps({
        "event": "sms:received",
        "payload": {
            "messageId": f"adb-{msg['_id']}-{msg['date']}",  # stable id -> verifier dedupes retries
            "message": msg["body"],
            "sender": msg["address"],
            "receivedAt": received_at,
        },
    }).encode()
    headers = {"Content-Type": "application/json"}
    if secret:
        ts = str(int(time.time()))
        headers["X-Timestamp"] = ts
        headers["X-Signature"] = hmac.new(secret.encode(), body + ts.encode(), hashlib.sha256).hexdigest()
    req = urllib.request.Request(url, data=body, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            return json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        detail = e.read().decode(errors="replace")[:200]
        if e.code in (401, 403):
            raise ForwardError(f"verifier rejected the request ({e.code}): {detail}. "
                               "Make sure --secret matches SIGNING_KEY on the verifier.")
        raise ForwardError(f"verifier returned {e.code}: {detail}")
    except (urllib.error.URLError, TimeoutError, ConnectionError) as e:
        raise ForwardError(f"cannot reach verifier at {url}: {e}")


def log(text: str, color=None):
    click.secho(f"[{datetime.now():%H:%M:%S}] {text}", fg=color)


def report(msg: dict, out: dict):
    res = out.get("result", "?")
    color = {"paid": "green", "unmatched": "yellow", "duplicate_ref": "yellow",
             "duplicate": "cyan"}.get(res)
    if res == "paid":
        log(f"SMS #{msg['_id']} -> PAID {out.get('paymentId')}  PHP {out.get('amount')}  ref {out.get('ref') or '-'}", color)
    else:
        log(f"SMS #{msg['_id']} -> {res}  {out.get('note', '')}".rstrip(), color)


# --- CLI ---------------------------------------------------------------------

@click.command(context_settings={"help_option_names": ["-h", "--help"]})
@click.option("--verifier-url", envvar="VERIFIER_URL", default="http://localhost:4000/webhook/sms",
              show_default=True, help="Verifier webhook endpoint.")
@click.option("--secret", envvar="SIGNING_KEY", default="",
              help="Shared signing key (must equal SIGNING_KEY on the verifier). Also read from $SIGNING_KEY.")
@click.option("--sender", "senders", multiple=True, default=("GCash",), show_default=True,
              help="SMS sender(s) to forward, case-insensitive. Repeatable. Everything else stays on the phone.")
@click.option("--interval", default=3.0, show_default=True, help="Seconds between inbox checks.")
@click.option("--device", help="Specific ADB device id (default: first ready device).")
@click.option("--backfill", default=0, show_default=True,
              help="First run only: also forward the N most recent matching messages.")
@click.option("--once", is_flag=True, help="Check once and exit (handy for testing).")
@click.option("--list-senders", is_flag=True, help="Print the unique SMS sender addresses on the phone and exit.")
@click.option("--state-file", type=click.Path(path_type=Path), default=DEFAULT_STATE, show_default=True)
def main(verifier_url, secret, senders, interval, device, backfill, once, list_senders, state_file):
    """Forward GCash SMS from an Android phone (via ADB) to the payment verifier."""
    try:
        ready, bad = list_devices()
    except AdbError as e:
        click.secho(f"Error: {e}", fg="red", bold=True)
        sys.exit(1)
    if not ready:
        click.secho("Error: no usable ADB device.", fg="red", bold=True)
        for dev, state in bad:
            hint = "  (accept the debugging prompt on the phone)" if state == "unauthorized" else ""
            click.echo(f"  {dev}: {state}{hint}")
        click.echo("  Wireless: adb pair <ip:port>  then  adb connect <ip:port>   |   USB: enable USB debugging and plug in.")
        sys.exit(1)
    target = device or ready[0]

    if list_senders:
        try:
            addrs = sorted({m["address"] for m in fetch_inbox(target) if m["address"]})
        except AdbError as e:
            click.secho(f"Error: {e}", fg="red")
            sys.exit(1)
        click.secho("Unique SMS senders on this phone:", bold=True)
        for a in addrs:
            click.echo(f"  {a}")
        click.echo("\nUse --sender with the exact GCash address shown above.")
        return

    if not secret:
        click.secho("WARNING: no --secret / $SIGNING_KEY: requests are unsigned. Only OK while the verifier also runs without SIGNING_KEY (dev).",
                    fg="yellow")
    allowed = {norm(s) for s in senders}
    click.secho("GCash SMS agent", fg="green", bold=True)
    click.echo(f"  device   : {target}")
    click.echo(f"  verifier : {verifier_url}")
    click.echo(f"  senders  : {', '.join(senders)}")

    # First run: start AFTER what is already in the inbox, so old payments are never replayed.
    state = load_state(state_file)
    last_id = state.get("last_id")
    if last_id is None:
        try:
            msgs = fetch_inbox(target)
        except AdbError as e:
            click.secho(f"Error: {e}", fg="red")
            sys.exit(1)
        ids = [m["_id"] for m in msgs]
        recent = []
        if backfill > 0:
            recent = sorted((m["_id"] for m in msgs if norm(m["address"]) in allowed), reverse=True)[:backfill]
        last_id = (min(recent) - 1) if recent else (max(ids) if ids else 0)
        save_state(state_file, {"last_id": last_id})
        log(f"first run: watching for SMS after #{last_id}" + (f" (backfilling {len(recent)})" if recent else ""), "cyan")

    log("watching... (Ctrl+C to stop)", "magenta")
    last_warning = ""
    try:
        while True:
            try:
                msgs = fetch_inbox(target)
                for m in sorted((m for m in msgs if m["_id"] > last_id), key=lambda m: m["_id"]):
                    if norm(m["address"]) in allowed:
                        report(m, forward(verifier_url, secret, m))
                    # advance only after success (or after skipping a non-GCash message)
                    last_id = m["_id"]
                    save_state(state_file, {"last_id": last_id})
                last_warning = ""
            except AdbError as e:
                if str(e) != last_warning:
                    log(f"ADB problem: {e}", "red")
                    last_warning = str(e)
                if not device:  # wireless debugging often changes port; re-detect
                    try:
                        ready, _ = list_devices()
                        if ready and target not in ready:
                            target = ready[0]
                            log(f"switched to device {target}", "cyan")
                    except AdbError:
                        pass
                time.sleep(min(interval * 3, 15))
            except ForwardError as e:
                if str(e) != last_warning:
                    log(f"{e} - will retry", "red")
                    last_warning = str(e)
                time.sleep(min(interval * 3, 15))
            if once:
                return
            time.sleep(interval)
    except KeyboardInterrupt:
        click.echo("\nstopped.")


if __name__ == "__main__":
    main()
