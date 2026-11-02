package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"instantcrew/internal/store/storetest"
)

// Facebook Login for Business takes its permissions from config_id, not scope.
func TestFacebookStart_SendsConfigID(t *testing.T) {
	h := &Handler{}
	if err := h.ConfigureFacebook("app", "secret", "https://x.test/cb", "cfg123"); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.FacebookStart(rec, httptest.NewRequest("GET", "/api/auth/facebook?role=applicant&mode=signup", nil))
	if rec.Code != http.StatusFound {
		t.Fatalf("status %d", rec.Code)
	}
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	q := loc.Query()
	if !strings.HasPrefix(loc.Host, "www.facebook.com") || q.Get("config_id") != "cfg123" || q.Has("scope") || q.Get("response_type") != "code" || q.Get("state") == "" {
		t.Fatalf("bad redirect: %s", loc)
	}
}

// Regular Facebook Login has no configuration, so it must ask for email itself.
func TestFacebookStart_RegularLoginAsksForEmail(t *testing.T) {
	h := &Handler{}
	if err := h.ConfigureFacebook("app", "secret", "https://x.test/cb", ""); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.FacebookStart(rec, httptest.NewRequest("GET", "/api/auth/facebook?role=applicant&mode=login", nil))
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if q := loc.Query(); q.Get("scope") != "email,public_profile" || q.Has("config_id") {
		t.Fatalf("bad redirect: %s", loc)
	}
}

// Facebook signups accept any confirmed email; Google keeps the Gmail rule.
func TestFinishSSO_GmailRuleByProvider(t *testing.T) {
	st, _ := storetest.Open(t)
	h := &Handler{Store: st, SessionTTL: time.Hour}
	finish := func(provider, email string) string {
		rec := httptest.NewRecorder()
		h.finishSSO(context.Background(), rec, httptest.NewRequest("GET", "/cb", nil), provider, email, "Crew", "applicant", "signup")
		return rec.Header().Get("Location")
	}
	if loc := finish("facebook", "crew@yahoo.com"); loc != "/applicant.html" {
		t.Fatalf("facebook signup redirected to %q", loc)
	}
	if loc := finish("google", "other@yahoo.com"); !strings.Contains(loc, "gmail_only") {
		t.Fatalf("google signup redirected to %q", loc)
	}
}
