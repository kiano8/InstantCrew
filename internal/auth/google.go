package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"instantcrew/internal/store"
)

const googleStateCookie = "ic_google_oauth_state"

type googleCredentials struct {
	Web struct {
		ClientID     string   `json:"client_id"`
		ClientSecret string   `json:"client_secret"`
		AuthURI      string   `json:"auth_uri"`
		TokenURI     string   `json:"token_uri"`
		RedirectURIs []string `json:"redirect_uris"`
	} `json:"web"`
}

type googleOAuth struct {
	ClientID, ClientSecret, AuthURI, TokenURI, RedirectURL string
}

func (h *Handler) ConfigureGoogleFromFile(path, redirectURL string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read credentials file: %w", err)
	}
	var c googleCredentials
	if err := json.Unmarshal(b, &c); err != nil {
		return fmt.Errorf("parse credentials file: %w", err)
	}
	return h.ConfigureGoogle(c.Web.ClientID, c.Web.ClientSecret, c.Web.AuthURI, c.Web.TokenURI, redirectURL, c.Web.RedirectURIs)
}

// ConfigureGoogle enables OAuth using credentials supplied by the deployment environment.
// This is useful when the credential JSON is intentionally excluded from source control.
func (h *Handler) ConfigureGoogle(clientID, clientSecret, authURI, tokenURI, redirectURL string, redirectURIs []string) error {
	if clientID == "" || clientSecret == "" {
		return errors.New("Google OAuth requires GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET")
	}
	if authURI == "" {
		authURI = "https://accounts.google.com/o/oauth2/v2/auth"
	}
	if tokenURI == "" {
		tokenURI = "https://oauth2.googleapis.com/token"
	}
	if redirectURL == "" && len(redirectURIs) > 0 {
		redirectURL = redirectURIs[0]
	}
	if redirectURL == "" {
		return errors.New("set GOOGLE_REDIRECT_URL or add a redirect URI to the credentials file")
	}
	h.google = &googleOAuth{clientID, clientSecret, authURI, tokenURI, redirectURL}
	return nil
}

func randomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (h *Handler) GoogleStart(w http.ResponseWriter, r *http.Request) {
	if h.google == nil {
		http.Error(w, "Google sign-in is not configured", http.StatusServiceUnavailable)
		return
	}
	role := r.URL.Query().Get("role")
	if role == "hirer" {
		role = "employer"
	}
	if role != "employer" && role != "applicant" {
		http.Error(w, "invalid account type", http.StatusBadRequest)
		return
	}
	mode := r.URL.Query().Get("mode")
	if mode != "signup" {
		mode = "login"
	}
	state, err := randomState()
	if err != nil {
		http.Error(w, "could not start Google sign-in", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: googleStateCookie, Value: state + "|" + role + "|" + mode, Path: "/api/auth/google/callback", MaxAge: 600, HttpOnly: true, Secure: h.CookieSecure, SameSite: http.SameSiteLaxMode})
	q := url.Values{"client_id": {h.google.ClientID}, "redirect_uri": {h.google.RedirectURL}, "response_type": {"code"}, "scope": {"openid email profile"}, "state": {state}, "prompt": {"select_account"}}
	http.Redirect(w, r, h.google.AuthURI+"?"+q.Encode(), http.StatusFound)
}

func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	if h.google == nil {
		http.Error(w, "Google sign-in is not configured", http.StatusServiceUnavailable)
		return
	}
	cookie, err := r.Cookie(googleStateCookie)
	clear := &http.Cookie{Name: googleStateCookie, Value: "", Path: "/api/auth/google/callback", MaxAge: -1, HttpOnly: true, Secure: h.CookieSecure, SameSite: http.SameSiteLaxMode}
	http.SetCookie(w, clear)
	parts := []string{}
	if err == nil {
		parts = strings.Split(cookie.Value, "|")
	}
	if len(parts) != 3 || r.URL.Query().Get("state") == "" || r.URL.Query().Get("state") != parts[0] {
		http.Error(w, "invalid sign-in state", http.StatusBadRequest)
		return
	}
	if e := r.URL.Query().Get("error"); e != "" {
		http.Error(w, "Google sign-in was cancelled", http.StatusUnauthorized)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	token, err := h.googleToken(ctx, r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "Google sign-in failed", http.StatusUnauthorized)
		return
	}
	profile, err := h.googleProfile(ctx, token)
	if err != nil || !profile.EmailVerified || profile.Email == "" {
		http.Error(w, "Google account email could not be verified", http.StatusUnauthorized)
		return
	}
	email := normalizeEmail(profile.Email)
	u, err := h.Store.UserByEmailRole(ctx, email, parts[1])
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		http.Error(w, "could not load account", http.StatusInternalServerError)
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		if parts[2] != "signup" {
			http.Error(w, "No account found for this Google email. Choose Create Account to register.", http.StatusForbidden)
			return
		}
		u = &store.User{Email: email, Role: parts[1], Name: strings.TrimSpace(profile.Name)}
		if u.Name == "" {
			u.Name = email
		}
		if err = h.Store.CreateUserCapped(ctx, u, h.MaxUsers); err != nil {
			if errors.Is(err, store.ErrBetaFull) {
				http.Error(w, "the beta is full right now. Please check back soon.", http.StatusForbidden)
				return
			}
			if errors.Is(err, store.ErrEmailTaken) {
				http.Error(w, "an account with this email already exists", http.StatusConflict)
				return
			}
			http.Error(w, "could not create account", http.StatusInternalServerError)
			return
		}
	}
	if err := h.startSession(w, r, u); err != nil {
		http.Error(w, "could not start session", http.StatusInternalServerError)
		return
	}
	if u.Role == "employer" {
		http.Redirect(w, r, "/employer.html", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/applicant.html", http.StatusSeeOther)
	}
}

type googleTokenResponse struct {
	AccessToken string `json:"access_token"`
}

func (g *googleOAuth) postForm(ctx context.Context, endpoint string, values url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Google returned %s", resp.Status)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
}
func (h *Handler) googleToken(ctx context.Context, code string) (string, error) {
	if code == "" {
		return "", errors.New("missing authorization code")
	}
	var out googleTokenResponse
	err := h.google.postForm(ctx, h.google.TokenURI, url.Values{"code": {code}, "client_id": {h.google.ClientID}, "client_secret": {h.google.ClientSecret}, "redirect_uri": {h.google.RedirectURL}, "grant_type": {"authorization_code"}}, &out)
	return out.AccessToken, err
}

type googleUserInfo struct {
	Email         string `json:"email"`
	Name          string `json:"name"`
	EmailVerified bool   `json:"email_verified"`
}

func (h *Handler) googleProfile(ctx context.Context, accessToken string) (googleUserInfo, error) {
	var out googleUserInfo
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://openidconnect.googleapis.com/v1/userinfo", nil)
	if err != nil {
		return out, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return out, fmt.Errorf("Google returned %s", resp.Status)
	}
	err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out)
	return out, err
}
