package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"instantcrew/gcash-verifier/internal/store"
)

// parseTime parses RFC3339 timestamps from SMS forwarders. Zero time if unknown.
func parseTime(s string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func signCallback(secret string, body []byte, ts string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	mac.Write([]byte(ts)) // hex(HMAC-SHA256(secret, body + X-Timestamp))
	return hex.EncodeToString(mac.Sum(nil))
}

// notifyPaid POSTs a signed "payment.paid" event to PAYMENT_CALLBACK_URL.
//
//	POST <PAYMENT_CALLBACK_URL>
//	X-Timestamp: <unix seconds>
//	X-Signature: hex(HMAC-SHA256(CALLBACK_SECRET, body + X-Timestamp))   (only if CALLBACK_SECRET is set)
//	{"event":"payment.paid","payment":{"id":"pay_..","label":"job-123","payAmount":"100.37","gcashRef":"..",...}}
//
// Put your own identifier (job id, booking id) in the payment's `label` when you
// create it, and read it back from the callback. Delivery is retried with backoff;
// make your receiver idempotent (key on payment.id).
func (a *app) notifyPaid(p *store.Payment) {
	url := os.Getenv("PAYMENT_CALLBACK_URL")
	if url == "" {
		return
	}
	secret := os.Getenv("CALLBACK_SECRET")
	body, err := json.Marshal(map[string]any{"event": "payment.paid", "payment": p})
	if err != nil {
		log.Printf("callback: marshal: %v", err)
		return
	}
	client := &http.Client{Timeout: 10 * time.Second}

	const attempts = 5
	for i := 1; i <= attempts; i++ {
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			log.Printf("callback: bad PAYMENT_CALLBACK_URL: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Timestamp", ts)
		if secret != "" {
			req.Header.Set("X-Signature", signCallback(secret, body, ts))
		}

		resp, err := client.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				log.Printf("callback: delivered payment.paid for %s (attempt %d)", p.ID, i)
				return
			}
			log.Printf("callback: %s answered %d for %s (attempt %d/%d)", url, resp.StatusCode, p.ID, i, attempts)
		} else {
			log.Printf("callback: %v (attempt %d/%d)", err, i, attempts)
		}
		if i < attempts {
			time.Sleep(time.Duration(1<<i) * time.Second) // 2s, 4s, 8s, 16s
		}
	}
	log.Printf("callback: GAVE UP delivering payment.paid for %s - the payment IS paid in the verifier; reconcile via GET /api/payments/%s", p.ID, p.ID)
}

func (a *app) registerCallbacks(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/dev/callback-sink", a.callbackSink)
}

// callbackSink is a dev-only receiver so you can test the callback without wiring
// your main site yet:
//
//	PAYMENT_CALLBACK_URL=http://localhost:4000/api/dev/callback-sink
func (a *app) callbackSink(w http.ResponseWriter, r *http.Request) {
	if a.prod {
		writeErr(w, 404, "not found")
		return
	}
	body, _ := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if secret := os.Getenv("CALLBACK_SECRET"); secret != "" {
		want := signCallback(secret, body, r.Header.Get("X-Timestamp"))
		if !hmac.Equal([]byte(want), []byte(r.Header.Get("X-Signature"))) {
			log.Printf("callback-sink: BAD SIGNATURE")
			writeErr(w, 401, "bad signature")
			return
		}
	}
	log.Printf("callback-sink: received %s", body)
	writeJSON(w, 200, map[string]string{"status": "received"})
}
