# gcash-verifier

Standalone GCash payment verification for InstantCrew. An Android phone with the
GCash SIM forwards incoming SMS to this service as a webhook. No payment gateway.

```
payer pays GCash number -> SMS arrives on the phone -> httpSMS (or SMSGate) POSTs a webhook
  -> match exact amount to a pending payment -> status = paid -> website polls GET /api/payments/{id}
```

Each payment gets a unique amount (your price + 1..99 centavos, e.g. `100.37`).
The payer must send exactly that amount; it is how an SMS is matched to a payment
even when GCash masks the sender name.

Two SMS forwarders are supported by the same matching logic:

| Provider | Webhook endpoint | Auth |
|---|---|---|
| **httpSMS** (https://httpsms.com) | `POST /webhook/httpsms` | `Authorization: Bearer <HS256 JWT>` signed with the webhook's signing key |
| SMSGate | `POST /webhook/sms` | `X-Signature` + `X-Timestamp` HMAC |

## Run

From the project root (`InstantCrew/`):

```bash
go test ./gcash-verifier/...

GCASH_NUMBER=09XXXXXXXXX go run ./gcash-verifier/cmd/verifier
# open http://localhost:4000
```

| Var | Default | Purpose |
|---|---|---|
| `PORT` | `4000` | listen port |
| `GCASH_NUMBER` | - | shown to payers on the test page |
| `SIGNING_KEY` | - | webhook signing key. Required when `ENV=prod` |
| `HTTPSMS_SIGNING_KEY` | `SIGNING_KEY` | optional separate key for the httpSMS webhook |
| `SENDER_ALLOWLIST` | - | comma-separated allowed SMS senders, e.g. `GCash`. Empty = any (testing only) |
| `PAYMENT_TTL_MINUTES` | `30` | how long a pending payment stays valid |
| `VERIFIER_DB_PATH` | `data/gcash-verifier.db` | SQLite file (`data/` is gitignored) |
| `CORS_ORIGIN` | `http://localhost:3000` | origin allowed to call `/api/*` from the browser |
| `ENV` | `dev` | `prod` disables `/api/dev/*` and `/api/sms-log`, and requires `SIGNING_KEY` |
| `PAYMENT_CALLBACK_URL` | - | where to POST `payment.paid` events |
| `CALLBACK_SECRET` | - | signs callback requests (HMAC-SHA256) |

## Test plan with httpSMS

httpSMS is a hosted service, so **its servers call your webhook**. `localhost` will
not work: you need a public HTTPS URL.

**A. No phone (2 min).** Open http://localhost:4000, *Create payment*, *Send simulated SMS*.
Status flips to `paid`; sending it again shows `duplicate_ref`.

**B. Real phone.**
1. Set up the phone in httpSMS: create an account at httpsms.com, install the Android app
   from the link in the dashboard, log in with your API key, grant SMS permission, and
   exempt the app from battery optimisation.
2. Expose your verifier publicly:
   ```bash
   cloudflared tunnel --url http://localhost:4000   # prints https://xxxx.trycloudflare.com
   ```
3. In the httpSMS dashboard go to **Settings > Webhooks > Add**:
   - Callback URL: `https://xxxx.trycloudflare.com/webhook/httpsms`
   - Events: `message.phone.received` (also tick `phone.heartbeat.offline` and `phone.heartbeat.online`)
   - Signing key: make up a long random string (e.g. `openssl rand -hex 32`)
4. Start the verifier with that key:
   ```bash
   SIGNING_KEY=<the same string> GCASH_NUMBER=09XXXXXXXXX go run ./gcash-verifier/cmd/verifier
   ```
5. Create a payment on the test page and pay the exact amount from another GCash account.
   The terminal prints `httpsms webhook from ... type=message.phone.received: {...}`
   and the page's SMS log shows the result.
6. If the result is `ignored_not_payment`, copy the real SMS text from the log into
   `internal/parser/parser_test.go` and adjust the regexes in `internal/parser/parser.go`.
7. Look at the sender shown in the SMS log, then restart with `SENDER_ALLOWLIST=<that sender>`.

Phone health: `GET /api/phone-status` reports whether the phone is online, based on httpSMS
heartbeat events (it flags offline after about an hour without a ping), and when the last SMS arrived.

## ADB agent (no app on the phone, nothing to get blocked)

`agent/adb_sms_agent.py` reads the phone's SMS inbox over ADB and forwards only GCash
messages to `POST /webhook/sms` on the verifier (signed with `SIGNING_KEY`).
Everything runs on your computer, so no tunnel and no third-party service are needed.

```bash
pip install click

# phone: Developer options > Wireless debugging (or USB debugging)
adb pair <ip:port>        # wireless, once
adb connect <ip:port>
adb devices               # should say "device"

# terminal 1: the verifier
SIGNING_KEY=<long random string> SENDER_ALLOWLIST=GCash GCASH_NUMBER=09XXXXXXXXX \
  go run ./gcash-verifier/cmd/verifier

# terminal 2: find GCash's exact sender address on the phone, then run the agent
python3 gcash-verifier/agent/adb_sms_agent.py --list-senders
SIGNING_KEY=<same string> python3 gcash-verifier/agent/adb_sms_agent.py --sender GCash
```

The agent prints `SMS #42 -> PAID pay_... PHP 100.37 ref ...` when a payment is verified.
On first run it starts *after* the existing inbox (old payments are never replayed);
use `--backfill N` to also forward the last N GCash messages. It remembers its position in
`agent/.agent_state.json`. Add that file to `.gitignore`.

Replay protection: the verifier only lets an SMS settle a payment that already existed when
the SMS arrived (5 min clock-skew allowance), so an old "received PHP 100.37" can't pay a new
payment with the same amount.

## Callback to your website when a payment is verified

Set `PAYMENT_CALLBACK_URL` (and `CALLBACK_SECRET`) and the verifier POSTs this as soon as a payment is settled:

```
POST <PAYMENT_CALLBACK_URL>
X-Timestamp: <unix seconds>
X-Signature: hex(HMAC-SHA256(CALLBACK_SECRET, rawBody + X-Timestamp))
{"event":"payment.paid","payment":{"id":"pay_..","label":"job-123","payAmount":"100.37",
 "status":"paid","gcashRef":"1234567890123","payerNumber":"09171234567","paidAt":"..."}}
```

Put your own id (job/booking) in `label` when calling `POST /api/payments` and read it back from the
callback. Delivery is retried 5 times with backoff (2s, 4s, 8s, 16s); answer with any 2xx. Make the
receiver idempotent (key on `payment.id`). If delivery fails, the payment is still paid in the verifier:
reconcile with `GET /api/payments/{id}`.

Try it without wiring your site first (dev only):

```bash
PAYMENT_CALLBACK_URL=http://localhost:4000/api/dev/callback-sink CALLBACK_SECRET=test123 \
  SIGNING_KEY=... go run ./gcash-verifier/cmd/verifier
```

The terminal shows `callback-sink: received {...}` right after the payment flips to paid.

## API

| Method | Path | Notes |
|---|---|---|
| POST | `/api/payments` | `{"amount": 100, "label": "job-123"}` -> `{payment, gcashNumber}` |
| GET | `/api/payments/{id}` | poll for `status`: `pending` / `paid` / `expired` |
| GET | `/api/payments` | last 20 |
| POST | `/webhook/httpsms` | httpSMS calls this (JWT verified) |
| POST | `/webhook/sms` | SMSGate calls this (HMAC verified) |
| GET | `/api/phone-status` | phone online/offline from heartbeats |
| GET | `/api/sms-log` | dev only |
| POST | `/api/dev/simulate` | dev only: `{"message": "..."}` |

## Known limits (fine for testing, fix before production)

- httpSMS is a third party: SMS text (including payer numbers) passes through their servers,
  and verification stops if their service or your phone is down. Heartbeat events help you notice;
  keep a manual-review fallback.
- Money that arrives after a payment expired, or for the wrong amount, is logged as `unmatched`
  for manual review; it is not auto-credited.
- Anyone can send a text that looks like GCash. The webhook token protects the endpoint, and
  `SENDER_ALLOWLIST` guards against spoofed SMS.
- Not yet wired into `employer.html` / `applicant.html` or the main app's DB.
