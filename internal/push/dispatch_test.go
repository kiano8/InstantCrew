package push

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"

	webpush "github.com/SherClockHolmes/webpush-go"

	"instantcrew/internal/store"
)

// TestDispatch needs a throwaway Postgres database, e.g.
// TEST_DATABASE_URL=postgres://localhost/instantcrew_test?sslmode=disable go test ./internal/push/
func TestDispatch(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	st, err := store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()

	employer := &store.User{Email: randomString(t) + "@test.local", PasswordHash: "x", Role: "employer", Name: "E"}
	applicant := &store.User{Email: randomString(t) + "@test.local", PasswordHash: "x", Role: "applicant", Name: "A"}
	for _, u := range []*store.User{employer, applicant} {
		if err := st.CreateUser(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	// Clear anything other tests left unclaimed.
	if _, err := st.ClaimPushNotifications(ctx, 10000); err != nil {
		t.Fatal(err)
	}

	// The push service is faked by routing every request to a local server.
	var mu sync.Mutex
	hits := map[string]int{}
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits[r.URL.Path]++
		mu.Unlock()
		if r.URL.Path == "/fcm/send/gone" {
			w.WriteHeader(http.StatusGone)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer fake.Close()

	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}
	s := New(st, Config{PublicKey: pub, PrivateKey: priv, Subject: "admin@test.local"})
	target, _ := url.Parse(fake.URL)
	s.client.Transport = rewriteTransport{target}

	live := store.PushSubscription{Endpoint: "https://fcm.googleapis.com/fcm/send/live-" + randomString(t)}
	gone := store.PushSubscription{Endpoint: "https://fcm.googleapis.com/fcm/send/gone"}
	for _, sub := range []*store.PushSubscription{&live, &gone} {
		sub.P256dh, sub.Auth = browserKeys(t)
		if err := st.SavePushSubscription(ctx, employer.ID, *sub); err != nil {
			t.Fatal(err)
		}
	}
	applicantSub := store.PushSubscription{Endpoint: "https://fcm.googleapis.com/fcm/send/applicant-" + randomString(t)}
	applicantSub.P256dh, applicantSub.Auth = browserKeys(t)
	if err := st.SavePushSubscription(ctx, applicant.ID, applicantSub); err != nil {
		t.Fatal(err)
	}

	if err := st.AddNotification(ctx, employer.ID, "Shift accepted", "Ana accepted.", "accept", `{"job_id":1}`); err != nil {
		t.Fatal(err)
	}
	if err := st.AddNotification(ctx, applicant.ID, "New shift", "Nearby.", "match", `{"job_id":2}`); err != nil {
		t.Fatal(err)
	}
	if err := st.AddNotification(ctx, applicant.ID, "Contract ended", "Ended.", "end", "{}"); err != nil {
		t.Fatal(err)
	}

	for range 2 { // the second pass must find nothing left to send
		if err := s.dispatch(ctx); err != nil {
			t.Fatal(err)
		}
	}

	livePath := "/fcm/send/" + live.Endpoint[len("https://fcm.googleapis.com/fcm/send/"):]
	applicantPath := "/fcm/send/" + applicantSub.Endpoint[len("https://fcm.googleapis.com/fcm/send/"):]
	if hits[livePath] != 1 || hits["/fcm/send/gone"] != 1 || hits[applicantPath] != 2 || len(hits) != 3 {
		t.Fatalf("push requests = %v, want one each to the employer's two devices and two to the applicant's device", hits)
	}
	subs, err := st.PushSubscriptions(ctx, employer.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 1 || subs[0].Endpoint != live.Endpoint {
		t.Fatalf("subscriptions after 410 = %v, want only the live one", subs)
	}
}

type rewriteTransport struct{ target *url.URL }

func (rt rewriteTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.URL.Scheme, r.URL.Host = rt.target.Scheme, rt.target.Host
	return http.DefaultTransport.RoundTrip(r)
}

// browserKeys returns the p256dh/auth pair a browser would generate.
func browserKeys(t *testing.T) (p256dh, auth string) {
	key, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	secret := make([]byte, 16)
	rand.Read(secret)
	enc := base64.RawURLEncoding
	return enc.EncodeToString(key.PublicKey().Bytes()), enc.EncodeToString(secret)
}

func randomString(t *testing.T) string {
	b := make([]byte, 6)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
