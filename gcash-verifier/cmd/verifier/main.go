// Command verifier is a standalone GCash payment verification service.
//
// Flow: payer sends GCash -> SMS lands on the dedicated phone -> SMSGate app
// POSTs a signed webhook here -> we parse amount + ref, match the exact amount
// to a pending payment, and mark it paid. The website polls GET /api/payments/{id}.
//
// Run from the project root:
//
//	SIGNING_KEY=<from SMSGate app> GCASH_NUMBER=09XXXXXXXXX go run ./gcash-verifier/cmd/verifier
package main

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"instantcrew/gcash-verifier/internal/store"
	"instantcrew/gcash-verifier/internal/verifier"
)

//go:embed web/index.html
var indexHTML []byte

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

type app struct {
	store       *store.Store
	svc         *verifier.Service
	signingKey  string
	gcashNumber string
	ttl         time.Duration
	prod        bool
	corsOrigin  string
}

func main() {
	port := getenv("PORT", "4000")
	dbPath := getenv("VERIFIER_DB_PATH", "data/gcash-verifier.db")
	env := getenv("ENV", "dev")
	ttlMin, err := strconv.Atoi(getenv("PAYMENT_TTL_MINUTES", "30"))
	if err != nil || ttlMin <= 0 {
		ttlMin = 30
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatal(err)
	}
	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	var senders []string
	for _, s := range strings.Split(os.Getenv("SENDER_ALLOWLIST"), ",") {
		if s = strings.TrimSpace(s); s != "" {
			senders = append(senders, s)
		}
	}

	a := &app{
		store:       st,
		svc:         &verifier.Service{Store: st, AllowedSenders: senders},
		signingKey:  os.Getenv("SIGNING_KEY"),
		gcashNumber: os.Getenv("GCASH_NUMBER"),
		ttl:         time.Duration(ttlMin) * time.Minute,
		prod:        env == "prod",
		corsOrigin:  getenv("CORS_ORIGIN", "http://localhost:3000"),
	}

	if a.signingKey == "" {
		if a.prod {
			log.Fatal("SIGNING_KEY is required when ENV=prod")
		}
		log.Println("WARNING: SIGNING_KEY not set - webhook signatures are NOT being checked (dev only!)")
	}
	if len(senders) == 0 {
		log.Println("WARNING: SENDER_ALLOWLIST not set - accepting SMS from any sender. After you see the real GCash sender in the SMS log, set SENDER_ALLOWLIST.")
	}
	if a.gcashNumber == "" {
		log.Println("note: GCASH_NUMBER not set; the test page won't show which number to pay")
	}

	a.svc.OnPaid = a.notifyPaid // call the website back when a payment is verified

	mux := http.NewServeMux()
	a.registerCallbacks(mux)
	mux.HandleFunc("GET /{$}", a.index)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("POST /webhook/sms", a.webhook) // SMSGate
	a.registerHTTPSMS(mux)                         // httpSMS: /webhook/httpsms, /api/phone-status
	mux.HandleFunc("POST /api/payments", a.createPayment)
	mux.HandleFunc("GET /api/payments", a.listPayments)
	mux.HandleFunc("GET /api/payments/{id}", a.getPayment)
	mux.HandleFunc("GET /api/sms-log", a.smsLog)
	mux.HandleFunc("POST /api/dev/simulate", a.simulate)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           a.cors(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("gcash-verifier listening on http://localhost:%s (db: %s, env: %s)", port, dbPath, env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// ---- helpers ----

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// cors lets the main site (port 3000) call /api/* from the browser.
func (a *app) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") && a.corsOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", a.corsOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Vary", "Origin")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// ---- handlers ----

func (a *app) index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(indexHTML)
}

func (a *app) createPayment(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Amount float64 `json:"amount"` // pesos
		Label  string  `json:"label"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&in); err != nil {
		writeErr(w, 400, "invalid JSON")
		return
	}
	if math.IsNaN(in.Amount) || in.Amount < 1 || in.Amount > 100000 {
		writeErr(w, 400, "amount must be between 1 and 100000 pesos")
		return
	}
	if len(in.Label) > 100 {
		in.Label = in.Label[:100]
	}
	p, err := a.store.CreatePayment(r.Context(), in.Label, int64(math.Round(in.Amount*100)), a.ttl)
	if err != nil {
		log.Printf("create payment: %v", err)
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, map[string]any{"payment": p, "gcashNumber": a.gcashNumber})
}

func (a *app) getPayment(w http.ResponseWriter, r *http.Request) {
	p, err := a.store.GetPayment(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, 404, "payment not found")
		return
	}
	if err != nil {
		writeErr(w, 500, "internal error")
		return
	}
	writeJSON(w, 200, p)
}

func (a *app) listPayments(w http.ResponseWriter, r *http.Request) {
	ps, err := a.store.ListPayments(r.Context(), 20)
	if err != nil {
		writeErr(w, 500, "internal error")
		return
	}
	writeJSON(w, 200, ps)
}

func (a *app) smsLog(w http.ResponseWriter, r *http.Request) {
	if a.prod {
		writeErr(w, 404, "not found")
		return
	}
	ls, err := a.store.ListSMS(r.Context(), 30)
	if err != nil {
		writeErr(w, 500, "internal error")
		return
	}
	writeJSON(w, 200, ls)
}

// webhook receives SMSGate "sms:received" events.
func (a *app) webhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if err != nil {
		writeErr(w, 413, "body too large")
		return
	}
	log.Printf("webhook from %s: %s", r.RemoteAddr, body) // raw payload: handy while testing

	sigOK := false
	if a.signingKey != "" {
		sigOK = verifier.VerifySignature(a.signingKey, body, r.Header.Get("X-Timestamp"), r.Header.Get("X-Signature"))
		if !sigOK {
			log.Printf("webhook REJECTED: bad or missing signature (check SIGNING_KEY matches the app)")
			writeErr(w, 401, "bad signature")
			return
		}
	}

	var wb struct {
		Event   string `json:"event"`
		Payload struct {
			MessageID  string `json:"messageId"`
			Message    string `json:"message"`
			Sender     string `json:"sender"`
			Phone      string `json:"phoneNumber"` // some versions name the sender this way
			ReceivedAt string `json:"receivedAt"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(body, &wb); err != nil {
		writeErr(w, 400, "invalid JSON")
		return
	}
	if wb.Event != "sms:received" {
		writeJSON(w, 200, map[string]string{"result": "ignored_event"})
		return
	}
	sender := wb.Payload.Sender
	if sender == "" {
		sender = wb.Payload.Phone
	}
	out, err := a.svc.Handle(r.Context(), verifier.Incoming{
		MessageID: wb.Payload.MessageID, Sender: sender, Message: wb.Payload.Message,
		SigOK: sigOK, Raw: string(body), ReceivedAt: parseTime(wb.Payload.ReceivedAt),
	})
	if err != nil {
		log.Printf("webhook handle error: %v", err)
		writeErr(w, 500, "internal error") // 5xx makes SMSGate retry
		return
	}
	writeJSON(w, 200, out)
}

// simulate feeds a fake SMS through the exact same pipeline as the webhook,
// so you can test the website flow without sending real money. Disabled when ENV=prod.
func (a *app) simulate(w http.ResponseWriter, r *http.Request) {
	if a.prod {
		writeErr(w, 404, "not found")
		return
	}
	var in struct {
		Message string `json:"message"`
		Sender  string `json:"sender"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&in); err != nil || in.Message == "" {
		writeErr(w, 400, "message required")
		return
	}
	if in.Sender == "" {
		in.Sender = "GCash"
	}
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	out, err := a.svc.Handle(r.Context(), verifier.Incoming{
		MessageID: "sim-" + hex.EncodeToString(b), Sender: in.Sender, Message: in.Message, SigOK: false, Raw: "(simulated)",
	})
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, out)
}
