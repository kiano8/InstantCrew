package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/mail"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"instantcrew/internal/httpx"
	"instantcrew/internal/store"
)

const (
	codeTTL            = 15 * time.Minute
	codeResendInterval = time.Minute
	// Outgoing code emails per hour, across everyone. Gmail allows about 500 a
	// day, so a script requesting codes can't use up the whole quota.
	maxCodeMailsPerHour = 100
)

// Mailer sends the verification and reset codes (see internal/mailer).
type Mailer interface {
	Configured() bool
	Send(ctx context.Context, to, subject, body string) error
}

var (
	errMailUnavailable = errors.New("email sending is not configured")
	errMailBusy        = errors.New("too many verification emails right now; please try again in a few minutes")
	errMailFailed      = errors.New("we couldn't send the email; please try again")
)

type mailLimiter struct {
	mu   sync.Mutex
	sent []time.Time
}

func (l *mailLimiter) allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := time.Now().Add(-time.Hour)
	kept := l.sent[:0]
	for _, t := range l.sent {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	l.sent = kept
	if len(l.sent) >= maxCodeMailsPerHour {
		return false
	}
	l.sent = append(l.sent, time.Now())
	return true
}

// validEmail rejects display-name forms like "Name <a@b>" that ParseAddress accepts.
func validEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}

// isGmail reports whether a normalized email is a personal Gmail address.
func isGmail(email string) bool {
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return false
	}
	domain := email[at+1:]
	return domain == "gmail.com" || domain == "googlemail.com"
}

func newCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func codeHash(purpose, email, code string) string {
	return hashToken(purpose + "|" + email + "|" + strings.TrimSpace(code))
}

func (h *Handler) mailReady() bool { return (h.Mail != nil && h.Mail.Configured()) || h.DevLogCodes }

// sendCode stores a fresh code for (email, purpose) and emails it.
func (h *Handler) sendCode(ctx context.Context, email, purpose, subject, bodyFormat string) error {
	if !h.mailReady() {
		return errMailUnavailable
	}
	code, err := newCode()
	if err != nil {
		return err
	}
	if err := h.Store.SaveEmailCode(ctx, email, purpose, codeHash(purpose, email, code), codeTTL, codeResendInterval); err != nil {
		return err
	}
	if !h.codeMail.allow() {
		_ = h.Store.DeleteEmailCode(ctx, email, purpose)
		return errMailBusy
	}
	if h.Mail == nil || !h.Mail.Configured() {
		log.Printf("dev: %s code for %s is %s (SMTP not configured)", purpose, email, code)
		return nil
	}
	if err := h.Mail.Send(ctx, email, subject, fmt.Sprintf(bodyFormat, code)); err != nil {
		// Dropping the code lets the person retry straight away.
		_ = h.Store.DeleteEmailCode(ctx, email, purpose)
		log.Printf("mail to %s: %v", email, err)
		return errMailFailed
	}
	return nil
}

func writeCodeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrCodeTooSoon), errors.Is(err, errMailBusy):
		httpx.Error(w, http.StatusTooManyRequests, err.Error())
	case errors.Is(err, errMailUnavailable):
		httpx.Error(w, http.StatusServiceUnavailable, "email verification is temporarily unavailable")
	case errors.Is(err, errMailFailed):
		httpx.Error(w, http.StatusBadGateway, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, "something went wrong")
	}
}

const signupCodeBody = `Your Instant Crew verification code is:

    %s

Enter it on the sign-up form to finish creating your account. It expires in 15 minutes.

If you didn't try to sign up for Instant Crew, you can ignore this email.
`

const resetCodeBody = `Your Instant Crew password reset code is:

    %s

Enter it in the "Reset your password" window together with your new password. It expires in 15 minutes.

If you didn't ask to reset your password, you can ignore this email; your password stays the same.
`

// POST /api/auth/signup-code body: {"email","role"} emails the code that Signup requires.
func (h *Handler) SignupCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if !httpx.Decode(w, r, &req) {
		return
	}
	email := normalizeEmail(req.Email)
	if req.Role != "employer" && req.Role != "applicant" {
		httpx.Error(w, http.StatusBadRequest, "invalid role")
		return
	}
	if !validEmail(email) || !isGmail(email) {
		httpx.Error(w, http.StatusBadRequest, "please sign up with a Gmail address (name@gmail.com)")
		return
	}
	switch _, err := h.Store.UserByEmailRole(r.Context(), email, req.Role); {
	case err == nil:
		httpx.Error(w, http.StatusConflict, "an account with this email already exists")
		return
	case !errors.Is(err, store.ErrNotFound):
		httpx.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	if err := h.sendCode(r.Context(), email, "signup:"+req.Role, "Your Instant Crew verification code", signupCodeBody); err != nil {
		writeCodeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusAccepted, map[string]bool{"sent": true})
}

// POST /api/auth/forgot-password body: {"email","role"}. The answer is the same
// whether or not the account exists, so it can't be used to discover emails.
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if !httpx.Decode(w, r, &req) {
		return
	}
	email := normalizeEmail(req.Email)
	if req.Role != "employer" && req.Role != "applicant" {
		httpx.Error(w, http.StatusBadRequest, "invalid role")
		return
	}
	if !validEmail(email) {
		httpx.Error(w, http.StatusBadRequest, "please enter a valid email address")
		return
	}
	if !h.mailReady() {
		writeCodeError(w, errMailUnavailable)
		return
	}
	switch _, err := h.Store.UserByEmailRole(r.Context(), email, req.Role); {
	case err == nil:
		if err := h.sendCode(r.Context(), email, "reset:"+req.Role, "Reset your Instant Crew password", resetCodeBody); err != nil &&
			!errors.Is(err, store.ErrCodeTooSoon) {
			log.Printf("password reset for %s: %v", email, err)
		}
	case !errors.Is(err, store.ErrNotFound):
		httpx.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	httpx.JSON(w, http.StatusAccepted, map[string]bool{"sent": true})
}

// POST /api/auth/reset-password body: {"email","role","code","password"}.
// Every existing session for the account is signed out.
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Role     string `json:"role"`
		Code     string `json:"code"`
		Password string `json:"password"`
	}
	if !httpx.Decode(w, r, &req) {
		return
	}
	email := normalizeEmail(req.Email)
	if req.Role != "employer" && req.Role != "applicant" {
		httpx.Error(w, http.StatusBadRequest, "invalid role")
		return
	}
	if len(req.Password) < 8 || len(req.Password) > 72 {
		httpx.Error(w, http.StatusBadRequest, "password must be 8 to 72 characters")
		return
	}
	u, err := h.Store.UserByEmailRole(r.Context(), email, req.Role)
	if errors.Is(err, store.ErrNotFound) {
		httpx.Error(w, http.StatusBadRequest, store.ErrCodeInvalid.Error())
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	if err := h.Store.UseEmailCode(r.Context(), email, "reset:"+req.Role, codeHash("reset:"+req.Role, email, req.Code)); err != nil {
		if errors.Is(err, store.ErrCodeInvalid) {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	if err := h.Store.ResetPassword(r.Context(), u.ID, string(hash)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
