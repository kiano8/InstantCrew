#!/usr/bin/env bash
# Demo helper: create a payment, show exactly what to send via GCash, and wait until it is verified.
#
#   gcash-verifier/scripts/demo-payment.sh [amount_php] [label] [--simulate]
#
#   amount_php   base price in pesos (default 1). The payer must send base + random centavos.
#   label        your own id, echoed back in the callback (default demo-<timestamp>)
#   --simulate   no real money: feed a fake GCash SMS through the same pipeline (dev mode only)
#
# Env: VERIFIER_BASE (default http://localhost:4000)
set -euo pipefail

BASE="${VERIFIER_BASE:-http://localhost:4000}"
AMOUNT="${1:-1}"
LABEL="${2:-demo-$(date +%s)}"
MODE="${3:-}"

py() { python3 -c "$1"; }

resp=$(curl -fsS -X POST "$BASE/api/payments" -H 'Content-Type: application/json' \
  -d "{\"amount\": $AMOUNT, \"label\": \"$LABEL\"}") || {
  echo "Could not reach the verifier at $BASE. Is it running? (go run ./gcash-verifier/cmd/verifier)" >&2
  exit 1
}

IFS=$'\t' read -r ID PAY NUM EXPIRES < <(printf '%s' "$resp" | py '
import sys, json
d = json.load(sys.stdin)
p = d["payment"]
print("\t".join([p["id"], p["payAmount"], d.get("gcashNumber") or "(GCASH_NUMBER not set)", p["expiresAt"]]))
')

echo
echo "  Payment id : $ID   (label: $LABEL)"
echo "  Send EXACTLY : PHP $PAY"
echo "  To GCash number : $NUM"
echo "  Valid until : $EXPIRES"
echo

if [ "$MODE" = "--simulate" ]; then
  REF=$(py 'import random; print("".join(str(random.randint(0,9)) for _ in range(13)))')
  echo "Simulating the GCash SMS (ref $REF)..."
  curl -fsS -X POST "$BASE/api/dev/simulate" -H 'Content-Type: application/json' \
    -d "{\"message\": \"You have received PHP $PAY of GCash from TEST U. 09171234567. Ref. No. $REF.\"}"
  echo
fi

echo "Waiting for verification (Ctrl+C to stop)..."
while true; do
  IFS=$'\t' read -r STATUS REFNO PAYER < <(curl -fsS "$BASE/api/payments/$ID" | py '
import sys, json
d = json.load(sys.stdin)
print("\t".join([d["status"], d.get("gcashRef") or "-", d.get("payerNumber") or "-"]))
')
  case "$STATUS" in
    paid)
      echo
      echo "PAID. GCash ref: $REFNO  payer: $PAYER"
      echo "Now check the verifier terminal for: callback: delivered payment.paid for $ID"
      exit 0 ;;
    expired)
      echo
      echo "Expired before any matching SMS arrived." >&2
      exit 2 ;;
  esac
  printf '.'
  sleep 2
done
