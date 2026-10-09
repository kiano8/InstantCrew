package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"instantcrew/internal/httpx"
	"instantcrew/internal/store"
)

const CookieName = "ic_session"

type Handler struct {
	Store        *store.Store
	SessionTTL   time.Duration
	CookieSecure bool
}

// Compared against when the email doesn't exist, so response time doesn't reveal valid emails.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcrypt.DefaultCost)

type userDTO struct {
	ID      int64  `json:"id"`
	Email   string `json:"email"`
	Role    string `json:"role"`
	Name    string `json:"name"`
	Company string `json:"company"`
}

func toDTO(u *store.User) userDTO {
	return userDTO{u.ID, u.Email, u.Role, u.Name, u.Company}
}

func validRole(r string) bool { return r == "employer" || r == "applicant" }

func normalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (h *Handler) startSession(w http.ResponseWriter, r *http.Request, u *store.User) error {
	token, err := newToken()
	if err != nil {
		return err
	}
	expires := time.Now().Add(h.SessionTTL)
	// Only a hash is stored, so a leaked database can't be used to hijack sessions.
	if err := h.Store.CreateSession(r.Context(), hashToken(token), u.ID, expires); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   h.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

// POST /api/auth/signup
func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role     string `json:"role"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`    // applicant full name, or employer contact person
		Company  string `json:"company"` // employers only
	}
	if !httpx.Decode(w, r, &req) {
		return
	}

	email := normalizeEmail(req.Email)
	name := strings.TrimSpace(req.Name)
	company := strings.TrimSpace(req.Company)

	if !validRole(req.Role) {
		httpx.Error(w, http.StatusBadRequest, "invalid role")
		return
	}
	if _, err := mail.ParseAddress(email); err != nil {
		httpx.Error(w, http.StatusBadRequest, "please enter a valid email address")
		return
	}
	// bcrypt only uses the first 72 bytes, so reject anything longer.
	if len(req.Password) < 8 || len(req.Password) > 72 {
		httpx.Error(w, http.StatusBadRequest, "password must be 8 to 72 characters")
		return
	}
	if name == "" {
		httpx.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Role == "employer" && company == "" {
		httpx.Error(w, http.StatusBadRequest, "company name is required")
		return
	}
	if req.Role == "applicant" {
		company = ""
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	u := &store.User{Email: email, PasswordHash: string(hash), Role: req.Role, Name: name, Company: company}
	if err := h.Store.CreateUser(r.Context(), u); err != nil {
		if errors.Is(err, store.ErrEmailTaken) {
			httpx.Error(w, http.StatusConflict, "an account with this email already exists")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	if err := h.startSession(w, r, u); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"user": toDTO(u)})
}

// POST /api/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role     string `json:"role"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !httpx.Decode(w, r, &req) {
		return
	}

	hash := dummyHash
	var u *store.User
	if validRole(req.Role) {
		found, err := h.Store.UserByEmailRole(r.Context(), normalizeEmail(req.Email), req.Role)
		switch {
		case err == nil:
			u = found
			hash = []byte(found.PasswordHash)
		case !errors.Is(err, store.ErrNotFound):
			httpx.Error(w, http.StatusInternalServerError, "something went wrong")
			return
		}
	}

	pwErr := bcrypt.CompareHashAndPassword(hash, []byte(req.Password))
	if u == nil || pwErr != nil {
		// One message for both cases so attackers can't tell which emails exist.
		httpx.Error(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err := h.startSession(w, r, u); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"user": toDTO(u)})
}

// POST /api/auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(CookieName); err == nil {
		_ = h.Store.DeleteSession(r.Context(), hashToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{
		Name: CookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: h.CookieSecure, SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/auth/me  (wrap with RequireAuth)
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	u, _ := UserFrom(r.Context())
	httpx.JSON(w, http.StatusOK, map[string]any{"user": toDTO(u)})
}
