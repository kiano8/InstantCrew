// Package push delivers employer and applicant notifications to their devices through Web Push,
// so they arrive while the app or browser is closed.
package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"

	"instantcrew/internal/auth"
	"instantcrew/internal/httpx"
	"instantcrew/internal/store"
)

// Notifications older than this when the dispatcher sees them are not pushed;
// the in-app list still shows them.
const maxAge = time.Hour

type Config struct {
	PublicKey  string // VAPID keys from `go run ./cmd/vapid`
	PrivateKey string
	Subject    string // contact for push services: an email or https URL
}

type Service struct {
	Store  *store.Store
	cfg    Config
	client *http.Client
}

func New(st *store.Store, cfg Config) *Service {
	return &Service{
		Store: st,
		cfg:   cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
			// Endpoints are checked against allowedHosts; a redirect must not leave them.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

func (s *Service) Enabled() bool { return s.cfg.PublicKey != "" && s.cfg.PrivateKey != "" }

func (s *Service) Register(mux *http.ServeMux, a *auth.Handler) {
	pushAuth := func(f http.HandlerFunc) http.Handler {
		return a.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, _ := auth.UserFrom(r.Context())
			if u == nil || (u.Role != "employer" && u.Role != "applicant" && u.Role != "admin") {
				httpx.Error(w, http.StatusForbidden, "forbidden")
				return
			}
			f(w, r)
		}))
	}
	mux.Handle("GET /api/push/key", pushAuth(s.Key))
	mux.Handle("POST /api/push/subscribe", pushAuth(s.Subscribe))
	mux.Handle("POST /api/push/unsubscribe", pushAuth(s.Unsubscribe))
}

// GET /api/push/key returns the VAPID public key the browser subscribes with.
func (s *Service) Key(w http.ResponseWriter, r *http.Request) {
	if !s.Enabled() {
		httpx.Error(w, http.StatusNotFound, "push notifications are not configured")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"public_key": s.cfg.PublicKey})
}

// subscriptionJSON is PushSubscription.toJSON() from the browser.
type subscriptionJSON struct {
	Endpoint       string   `json:"endpoint"`
	ExpirationTime *float64 `json:"expirationTime"`
	Keys           struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

// POST /api/push/subscribe
func (s *Service) Subscribe(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in subscriptionJSON
	if !httpx.Decode(w, r, &in) {
		return
	}
	if !s.Enabled() {
		httpx.Error(w, http.StatusNotFound, "push notifications are not configured")
		return
	}
	if err := ValidEndpoint(in.Endpoint); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Keys.P256dh == "" || in.Keys.Auth == "" || len(in.Keys.P256dh) > 200 || len(in.Keys.Auth) > 100 {
		httpx.Error(w, http.StatusBadRequest, "invalid subscription keys")
		return
	}
	sub := store.PushSubscription{Endpoint: in.Endpoint, P256dh: in.Keys.P256dh, Auth: in.Keys.Auth}
	if err := s.Store.SavePushSubscription(r.Context(), u.ID, sub); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not save subscription")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/push/unsubscribe body: {"endpoint":"..."}; called before logout.
func (s *Service) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in struct {
		Endpoint string `json:"endpoint"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if err := s.Store.DeletePushSubscription(r.Context(), u.ID, in.Endpoint); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not remove subscription")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// allowedHosts are the browser vendors' push services. The server POSTs to the
// endpoint a browser hands it, so anything else could aim it at internal hosts.
var allowedHosts = []string{
	"fcm.googleapis.com",                // Chrome, Edge on Android, Samsung Internet, Opera
	"updates.push.services.mozilla.com", // Firefox
	".push.apple.com",                   // Safari, iOS home-screen apps
	".notify.windows.com",               // Edge on Windows
}

func ValidEndpoint(endpoint string) error {
	if len(endpoint) > 2048 {
		return errors.New("endpoint too long")
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return errors.New("endpoint must be an https URL")
	}
	host := strings.ToLower(u.Hostname())
	for _, h := range allowedHosts {
		if host == h || (strings.HasPrefix(h, ".") && strings.HasSuffix(host, h)) {
			return nil
		}
	}
	return errors.New("unsupported push service")
}

// Run sends newly created notifications until ctx is cancelled.
func (s *Service) Run(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			if err := s.dispatch(ctx); err != nil && ctx.Err() == nil {
				log.Printf("push: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) dispatch(ctx context.Context) error {
	pending, err := s.Store.ClaimPushNotifications(ctx, 100)
	if err != nil {
		return err
	}
	for _, n := range pending {
		if (n.Role != "employer" && n.Role != "applicant") || time.Since(time.Unix(n.CreatedAt, 0)) > maxAge {
			continue
		}
		subs, err := s.Store.PushSubscriptions(ctx, n.UserID)
		if err != nil {
			return err
		}
		if len(subs) == 0 {
			continue
		}
		msg, err := json.Marshal(message(n))
		if err != nil {
			return err
		}
		for _, sub := range subs {
			s.send(ctx, sub, msg)
		}
	}
	return nil
}

// message is what sw.js's push handler shows. The tag matches the one the open
// page uses, so the two never stack up as duplicate notifications.
func message(n store.PendingPush) map[string]string {
	role := "employer"
	link := "/employer.html"
	if n.Role == "applicant" {
		role = "applicant"
		link = "/applicant.html"
	}
	var payload struct {
		JobID *int64 `json:"job_id"`
	}
	if json.Unmarshal([]byte(n.Payload), &payload) == nil && payload.JobID != nil {
		link += fmt.Sprintf("?job=%d", *payload.JobID)
	}
	return map[string]string{
		"title": n.Title,
		"body":  n.Message,
		"tag":   fmt.Sprintf("%s-%d", role, n.ID),
		"url":   link,
	}
}

func (s *Service) send(ctx context.Context, sub store.PushSubscription, msg []byte) {
	if ValidEndpoint(sub.Endpoint) != nil {
		_ = s.Store.DeletePushEndpoint(ctx, sub.Endpoint)
		return
	}
	resp, err := webpush.SendNotificationWithContext(ctx, msg, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys:     webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth},
	}, &webpush.Options{
		HTTPClient:      s.client,
		Subscriber:      s.cfg.Subject,
		VAPIDPublicKey:  s.cfg.PublicKey,
		VAPIDPrivateKey: s.cfg.PrivateKey,
		TTL:             int(maxAge / time.Second),
		Urgency:         webpush.UrgencyHigh,
	})
	if err != nil {
		log.Printf("push: send: %v", err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		// The user revoked permission or the browser dropped the subscription.
		_ = s.Store.DeletePushEndpoint(ctx, sub.Endpoint)
	case resp.StatusCode >= 400:
		log.Printf("push: %s responded %d", hostOf(sub.Endpoint), resp.StatusCode)
	}
}

func hostOf(endpoint string) string {
	if u, err := url.Parse(endpoint); err == nil {
		return u.Host
	}
	return "push service"
}
