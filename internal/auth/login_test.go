package auth

import (
	"context"
	"errors"
	"fmt"
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

// Ten wrong passwords from one address lock that address, not the owner elsewhere.
func TestLoginLimiter(t *testing.T) {
	var l loginLimiter
	for i := 0; i < maxLoginFailures-1; i++ {
		l.fail("a@b.c", "1.1.1.1")
	}
	if l.blocked("a@b.c", "1.1.1.1") {
		t.Fatal("blocked before the limit")
	}
	l.fail("a@b.c", "1.1.1.1")
	if !l.blocked("a@b.c", "1.1.1.1") || l.blocked("other@b.c", "1.1.1.1") {
		t.Fatal("want a@b.c blocked and other@b.c not")
	}
	if l.blocked("a@b.c", "2.2.2.2") {
		t.Fatal("an attacker's failures must not lock the owner's address")
	}
	l.fails["a@b.c|1.1.1.1"] = []time.Time{time.Now().Add(-loginLockout)}
	if l.blocked("a@b.c", "1.1.1.1") {
		t.Fatal("old failures must expire")
	}
	l.reset("a@b.c", "1.1.1.1")
	if len(l.fails) != 0 {
		t.Fatal("reset must clear the email")
	}
	for i := 0; i < maxLoginFailuresPerEmail; i++ {
		l.fail("a@b.c", fmt.Sprintf("10.0.0.%d", i))
	}
	if !l.blocked("a@b.c", "9.9.9.9") {
		t.Fatal("rotating addresses must still hit the per-email cap")
	}
}

// A password reset ends the lockout, as the lockout message promises.
func TestResetPasswordClearsLockout(t *testing.T) {
	st, _ := storetest.Open(t)
	ctx := context.Background()
	h := &Handler{Store: st, SessionTTL: time.Hour}
	if err := st.CreateUser(ctx, &store.User{Email: "a@gmail.com", PasswordHash: "x", Role: "employer", Name: "A"}); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveEmailCode(ctx, "a@gmail.com", "reset:employer", codeHash("reset:employer", "a@gmail.com", "123456"), time.Hour, 0); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/api/auth/reset-password", strings.NewReader(`{"email":"a@gmail.com","role":"employer","code":"123456","password":"new-password"}`))
	req.Header.Set("Content-Type", "application/json")
	ip := clientIP(req)
	for i := 0; i < maxLoginFailures; i++ {
		h.loginFails.fail("a@gmail.com", ip)
	}
	rec := httptest.NewRecorder()
	h.ResetPassword(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("reset: %d %s", rec.Code, rec.Body)
	}
	if h.loginFails.blocked("a@gmail.com", ip) {
		t.Fatal("reset must clear the lockout")
	}
}

func TestNormalizePHMobile(t *testing.T) {
	for _, in := range []string{"09171234567", "0917 123 4567", "+63 917 123-4567", "639171234567", "9171234567"} {
		if got, ok := normalizePHMobile(in); !ok || got != "+639171234567" {
			t.Fatalf("%q: got %q %v", in, got, ok)
		}
	}
	for _, in := range []string{"", "0917123456", "08171234567", "091712345678", "a@gmail.com", "+1 415 555 2671"} {
		if got, ok := normalizePHMobile(in); ok {
			t.Fatalf("%q must be refused, got %q", in, got)
		}
	}
}

// Crew sign up with a contact number and no code, then log in however they type it.
func TestApplicantSignupWithContactNumber(t *testing.T) {
	st, _ := storetest.Open(t)
	h := &Handler{Store: st, SessionTTL: time.Hour}
	post := func(f http.HandlerFunc, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		f(rec, req)
		return rec
	}
	signup := `{"role":"applicant","name":"Crew","email":"0917 123 4567","password":"secret-pass"}`
	if rec := post(h.Signup, signup); rec.Code != http.StatusCreated {
		t.Fatalf("signup: %d %s", rec.Code, rec.Body)
	}
	if rec := post(h.Signup, `{"role":"applicant","name":"Twin","email":"+639171234567","password":"secret-pass"}`); rec.Code != http.StatusConflict {
		t.Fatalf("same number in another format: want 409, got %d", rec.Code)
	}
	if rec := post(h.Login, `{"role":"applicant","email":"09171234567","password":"secret-pass"}`); rec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	if rec := post(h.Signup, `{"role":"applicant","name":"Crew","email":"crew@gmail.com","password":"secret-pass"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("email signup for crew: want 400, got %d", rec.Code)
	}
	if rec := post(h.Signup, `{"role":"employer","name":"Boss","company":"Co","email":"boss@gmail.com","password":"secret-pass"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("employer without a code: want 400, got %d", rec.Code)
	}
	for i := 0; i < maxLoginFailures; i++ {
		h.crewSignups.fail("signup", "192.0.2.1")
	}
	if rec := post(h.Signup, `{"role":"applicant","name":"Bot","email":"09181234567","password":"secret-pass"}`); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("signup flood: want 429, got %d", rec.Code)
	}
}
