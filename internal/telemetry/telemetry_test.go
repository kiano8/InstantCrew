package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"instantcrew/internal/store"
	"instantcrew/internal/store/storetest"
)

func TestRecordsPatternsNotPaths(t *testing.T) {
	st, _ := storetest.Open(t)
	rec := New(st)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /things/{id}", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("GET /boom", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) })
	mux.HandleFunc("GET /stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("stream flush through wrapper: %v", err)
		}
	})
	h := rec.Middleware(mux)
	for _, p := range []string{"/things/secret-1", "/things/secret-2", "/boom", "/stream", "/nope"} {
		req := httptest.NewRequest("GET", p, nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.1")
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
	if err := rec.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	r, err := st.TelemetryReport(context.Background(), time.Now(), 60, 5)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]store.TelemetryRoute{}
	for _, x := range r.Routes {
		got[x.Route] = x
	}
	if len(got) != 3 || got["GET /things/{id}"].Requests != 2 || got["GET /boom"].ServerErrors != 1 || got["(unmatched)"].ClientErrors != 1 {
		t.Fatalf("routes = %+v", r.Routes)
	}
	if n := len(r.Visitors); n != 1 || r.Visitors[0].Visitors != 1 {
		t.Fatalf("visitors = %+v, want one visitor", r.Visitors)
	}
	if last := r.Series[len(r.Series)-1]; last.Requests+r.Series[len(r.Series)-2].Requests != 4 {
		t.Fatalf("series = %+v", r.Series)
	}
}

func TestFailedFlushKeepsCounts(t *testing.T) {
	st, _ := storetest.Open(t)
	rec := New(st)
	key := store.TelemetryKey{Minute: 60, Route: "GET /x", Status: 200}
	rec.counts[key] = store.TelemetryCount{Requests: 3}
	st.Close()
	if err := rec.Flush(context.Background()); err == nil {
		t.Fatal("flush on closed store succeeded")
	}
	if rec.counts[key].Requests != 3 {
		t.Fatalf("counts lost: %+v", rec.counts)
	}
}

func TestOutagesFromHeartbeatGaps(t *testing.T) {
	st, raw := storetest.Open(t)
	now := time.Now().Unix() / 60 * 60
	// Up for 60 minutes except 10 missing heartbeats.
	for m := now - 3600; m <= now; m += 60 {
		if m > now-1200 && m <= now-600 {
			continue
		}
		if _, err := raw.Exec(`INSERT INTO telemetry_heartbeats (minute) VALUES ($1)`, m); err != nil {
			t.Fatal(err)
		}
	}
	r, err := st.TelemetryReport(context.Background(), time.Unix(now, 0), 60, 60)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Outages) != 1 || r.Outages[0].Seconds != 600 || r.Outages[0].End != now-540 {
		t.Fatalf("outages = %+v", r.Outages)
	}
	if r.UptimePercent < 83 || r.UptimePercent > 84 {
		t.Fatalf("uptime = %v", r.UptimePercent)
	}
}

// The users table lists real signups only: no admins, no cmd/seed demo accounts.
func TestReportUsersSkipsSeedAndAdmin(t *testing.T) {
	st, _ := storetest.Open(t)
	ctx := context.Background()
	for _, u := range []store.User{
		{Email: "+639171234567", Role: "applicant", Name: "Crew"},
		{Email: "boss@gmail.com", Role: "employer", Name: "Boss", Company: "Co"},
		{Email: "cook@seed.instantcrew.test", Role: "applicant", Name: "Seed"},
		{Email: "admin@instantcrew.com", Role: "admin", Name: "Admin"},
	} {
		u.PasswordHash = "x"
		if err := st.CreateUser(ctx, &u); err != nil {
			t.Fatal(err)
		}
	}
	r, err := st.TelemetryReport(ctx, time.Now(), 60, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Users) != 2 || r.Users[0].Login != "boss@gmail.com" || r.Users[1].Login != "+639171234567" {
		t.Fatalf("users = %+v", r.Users)
	}
	var signups int64
	for _, p := range r.Series {
		signups += p.Signups
	}
	if signups != 2 {
		t.Fatalf("signup chart counted %d, want 2", signups)
	}
}
