package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"instantcrew/internal/store"
	"instantcrew/internal/store/storetest"
)

type fakeMail struct{ sent bool }

func (m *fakeMail) Configured() bool { return true }
func (m *fakeMail) Send(context.Context, string, string, string) error {
	m.sent = true
	return nil
}

// A verified account signs in with its password alone, and no email goes out.
func TestLogin_PasswordOnly(t *testing.T) {
	st, _ := storetest.Open(t)
	mail := &fakeMail{}
	h := &Handler{Store: st, SessionTTL: time.Hour, Mail: mail}
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	if err := st.CreateUser(context.Background(), &store.User{Email: "crew@gmail.com", PasswordHash: string(hash), Role: "applicant", Name: "Crew"}); err != nil {
		t.Fatal(err)
	}
	body := `{"role":"applicant","email":"crew@gmail.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Login(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body)
	}
	hasSession := false
	for _, c := range rec.Result().Cookies() {
		hasSession = hasSession || (c.Name == CookieName && c.Value != "")
	}
	if !hasSession {
		t.Fatal("want a session cookie")
	}
	if mail.sent {
		t.Fatal("login must not send email")
	}
}

// A Gmail with an employer account can't also sign up as an applicant.
func TestSignup_EmailHoldsOneRole(t *testing.T) {
	st, _ := storetest.Open(t)
	ctx := context.Background()
	mail := &fakeMail{}
	h := &Handler{Store: st, SessionTTL: time.Hour, Mail: mail}
	if err := st.CreateUser(ctx, &store.User{Email: "boss@gmail.com", PasswordHash: "x", Role: "employer", Name: "Boss", Company: "Co"}); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/api/auth/signup-code", strings.NewReader(`{"email":"boss@gmail.com","role":"applicant"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.SignupCode(rec, req)
	if rec.Code != http.StatusConflict || mail.sent {
		t.Fatalf("signup code: want 409 and no email, got %d sent=%v", rec.Code, mail.sent)
	}
	err := st.CreateUserCapped(ctx, &store.User{Email: "boss@gmail.com", PasswordHash: "x", Role: "applicant", Name: "Boss"}, 0)
	if !errors.Is(err, store.ErrEmailTaken) {
		t.Fatalf("create: want ErrEmailTaken, got %v", err)
	}
	if err := st.CreateUserCapped(ctx, &store.User{Email: "crew@gmail.com", PasswordHash: "x", Role: "applicant", Name: "Crew"}, 0); err != nil {
		t.Fatalf("a new email must still sign up: %v", err)
	}
}

// Ten wrong passwords lock the email; a success clears the count.
func TestLoginLimiter(t *testing.T) {
	var l loginLimiter
	for i := 0; i < maxLoginFailures-1; i++ {
		l.fail("a@b.c")
	}
	if l.blocked("a@b.c") {
		t.Fatal("blocked before the limit")
	}
	l.fail("a@b.c")
	if !l.blocked("a@b.c") || l.blocked("other@b.c") {
		t.Fatal("want a@b.c blocked and other@b.c not")
	}
	l.fails["a@b.c"] = []time.Time{time.Now().Add(-loginLockout)}
	if l.blocked("a@b.c") {
		t.Fatal("old failures must expire")
	}
	l.fail("a@b.c")
	l.reset("a@b.c")
	if len(l.fails) != 0 {
		t.Fatal("reset must clear the email")
	}
}
