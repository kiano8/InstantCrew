package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"instantcrew/internal/store"
	"instantcrew/internal/store/storetest"
)

type fakeMail struct{ last string }

func (m *fakeMail) Configured() bool { return true }
func (m *fakeMail) Send(_ context.Context, _, _, body string) error {
	m.last = body
	return nil
}

// A Gmail password login only starts a session after the emailed code;
// other addresses sign in with the password alone.
func TestLogin_GmailNeedsEmailedCode(t *testing.T) {
	st, _ := storetest.Open(t)
	ctx := context.Background()
	mail := &fakeMail{}
	h := &Handler{Store: st, SessionTTL: time.Hour, Mail: mail}
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	for _, email := range []string{"crew@gmail.com", "crew@company.ph"} {
		if err := st.CreateUser(ctx, &store.User{Email: email, PasswordHash: string(hash), Role: "applicant", Name: "Crew"}); err != nil {
			t.Fatal(err)
		}
	}
	login := func(email, code string) (*httptest.ResponseRecorder, map[string]any) {
		body, _ := json.Marshal(map[string]string{"role": "applicant", "email": email, "password": "password123", "code": code})
		req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.Login(rec, req)
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec, out
	}
	hasSession := func(rec *httptest.ResponseRecorder) bool {
		for _, c := range rec.Result().Cookies() {
			if c.Name == CookieName && c.Value != "" {
				return true
			}
		}
		return false
	}

	rec, out := login("crew@gmail.com", "")
	if rec.Code != http.StatusAccepted || out["mfa_required"] != true || hasSession(rec) {
		t.Fatalf("first step: want 202 mfa_required and no session, got %d %v", rec.Code, out)
	}
	code := regexp.MustCompile(`\d{6}`).FindString(mail.last)
	if code == "" {
		t.Fatalf("no code in email: %q", mail.last)
	}
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	if rec, _ := login("crew@gmail.com", wrong); rec.Code != http.StatusBadRequest || hasSession(rec) {
		t.Fatalf("wrong code: want 400 and no session, got %d", rec.Code)
	}
	if rec, _ := login("crew@gmail.com", code); rec.Code != http.StatusOK || !hasSession(rec) {
		t.Fatalf("right code: want 200 with session, got %d", rec.Code)
	}
	if rec, _ := login("crew@gmail.com", code); rec.Code != http.StatusBadRequest {
		t.Fatalf("a used code must not work twice, got %d", rec.Code)
	}

	if rec, _ := login("crew@company.ph", ""); rec.Code != http.StatusOK || !hasSession(rec) {
		t.Fatalf("non-Gmail login: want 200 with session, got %d", rec.Code)
	}
}
