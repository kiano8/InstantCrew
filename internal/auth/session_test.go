package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"instantcrew/internal/store"
	"instantcrew/internal/store/storetest"
)

// With employer and applicant signed in on one browser, the shared cookie holds
// the applicant token. Activity from the employer tab must still keep the
// employer session alive, and a full logout must end both sessions.
func TestActivityAndLogout_CoverEveryRoleCookie(t *testing.T) {
	st, raw := storetest.Open(t)
	st.SetSessionIdleTimeout(15 * time.Minute)
	ctx := context.Background()
	h := &Handler{Store: st, SessionTTL: time.Hour, IdleTimeout: 15 * time.Minute}

	emp := &store.User{Email: "p@gmail.com", PasswordHash: "x", Role: "employer", Name: "P", Company: "Co"}
	app := &store.User{Email: "p@gmail.com", PasswordHash: "x", Role: "applicant", Name: "P"}
	for _, u := range []*store.User{emp, app} {
		if err := st.CreateUser(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	for token, u := range map[string]*store.User{"emp-token": emp, "app-token": app} {
		if err := st.CreateSession(ctx, hashToken(token), u.ID, time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	// Both sessions were last used 10 minutes ago.
	if _, err := raw.Exec(`UPDATE sessions SET last_seen_at=$1`, time.Now().Add(-10*time.Minute).Unix()); err != nil {
		t.Fatal(err)
	}
	withCookies := func(req *http.Request) *http.Request {
		req.AddCookie(&http.Cookie{Name: CookieName, Value: "app-token"})
		req.AddCookie(&http.Cookie{Name: CookieName + "_applicant", Value: "app-token"})
		req.AddCookie(&http.Cookie{Name: CookieName + "_employer", Value: "emp-token"})
		return req
	}

	h.Activity(httptest.NewRecorder(), withCookies(httptest.NewRequest("POST", "/api/auth/activity", nil)))

	var lastSeen int64
	if err := raw.QueryRow(`SELECT last_seen_at FROM sessions WHERE token_hash=$1`, hashToken("emp-token")).Scan(&lastSeen); err != nil {
		t.Fatal(err)
	}
	if time.Since(time.Unix(lastSeen, 0)) > time.Minute {
		t.Fatalf("employer session was not touched by activity (last seen %v ago)", time.Since(time.Unix(lastSeen, 0)))
	}

	h.Logout(httptest.NewRecorder(), withCookies(httptest.NewRequest("POST", "/api/auth/logout", nil)))
	var left int
	if err := raw.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 0 {
		t.Fatalf("logout left %d sessions alive", left)
	}
}
