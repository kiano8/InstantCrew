package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"instantcrew/gcash-verifier/internal/verifier"
)

// phone tracks the dedicated phone's health from httpSMS heartbeat events.
// httpSMS emits phone.heartbeat.offline after ~1 hour without a ping and
// phone.heartbeat.online when it comes back.
var phone struct {
	sync.Mutex
	known     bool
	online    bool
	changedAt time.Time
	lastSMS   time.Time
}

func (a *app) registerHTTPSMS(mux *http.ServeMux) {
	mux.HandleFunc("POST /webhook/httpsms", a.httpsmsWebhook)
	mux.HandleFunc("GET /api/phone-status", a.phoneStatus)
}

func (a *app) phoneStatus(w http.ResponseWriter, r *http.Request) {
	phone.Lock()
	defer phone.Unlock()
	out := map[string]any{"known": phone.known}
	if phone.known {
		out["online"] = phone.online
		out["since"] = phone.changedAt
	}
	if !phone.lastSMS.IsZero() {
		out["lastSmsAt"] = phone.lastSMS
	}
	writeJSON(w, 200, out)
}

// httpsmsWebhook receives events from httpSMS (https://httpsms.com).
// Auth: "Authorization: Bearer <HS256 JWT>" signed with the webhook's signing key.
// httpSMS times out after 5 seconds, so we keep this fast and always answer 200
// for business outcomes (only genuine internal errors return 5xx).
func (a *app) httpsmsWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if err != nil {
		writeErr(w, 413, "body too large")
		return
	}
	log.Printf("httpsms webhook from %s type=%s: %s", r.RemoteAddr, r.Header.Get("X-Event-Type"), body)

	// HTTPSMS_SIGNING_KEY wins; falls back to SIGNING_KEY so one variable is enough.
	key := getenv("HTTPSMS_SIGNING_KEY", a.signingKey)
	sigOK := false
	if key != "" {
		sigOK = verifier.VerifyHTTPSMSToken(key, r.Header.Get("Authorization"), time.Now())
		if !sigOK {
			log.Printf("httpsms webhook REJECTED: bad/missing token (does the signing key match the one set on the webhook?)")
			writeErr(w, 401, "bad token")
			return
		}
	}

	var ev verifier.HTTPSMSEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		writeErr(w, 400, "invalid JSON")
		return
	}
	evType := ev.Type
	if evType == "" {
		evType = r.Header.Get("X-Event-Type")
	}

	switch evType {
	case "phone.heartbeat.offline", "phone.heartbeat.online":
		online := evType == "phone.heartbeat.online"
		phone.Lock()
		phone.known, phone.online, phone.changedAt = true, online, time.Now()
		phone.Unlock()
		if online {
			log.Printf("PHONE ONLINE: payments will be verified")
		} else {
			log.Printf("PHONE OFFLINE: no heartbeat for ~1h. Incoming payments will NOT be verified until it's back")
		}
		writeJSON(w, 200, map[string]string{"result": "ok"})

	case "message.phone.received":
		msgID := ev.Data.MessageID
		if msgID == "" {
			msgID = ev.ID
		}
		out, err := a.svc.Handle(r.Context(), verifier.Incoming{
			MessageID: msgID, Sender: ev.Data.Contact, Message: ev.Data.Content,
			SigOK: sigOK, Raw: string(body), ReceivedAt: parseTime(ev.Data.Timestamp),
		})
		if err != nil {
			log.Printf("httpsms handle error: %v", err)
			writeErr(w, 500, "internal error")
			return
		}
		phone.Lock()
		phone.lastSMS = time.Now()
		phone.Unlock()
		writeJSON(w, 200, out)

	default:
		writeJSON(w, 200, map[string]string{"result": "ignored_event", "type": evType})
	}
}
