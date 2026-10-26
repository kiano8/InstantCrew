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

// ssoFail sends the browser to a friendly error page instead of a bare text
// response, keeping the role and mode so the page can offer the right retry.
func ssoFail(w http.ResponseWriter, r *http.Request, reason, role, mode string) {
	q := url.Values{"reason": {reason}}
	if role != "" {
		q.Set("role", role)
	}
	if mode != "" {
		q.Set("mode", mode)
	}
	http.Redirect(w, r, "/sso-error.html?"+q.Encode(), http.StatusSeeOther)
}

func (h *Handler) GoogleStart(w http.ResponseWriter, r *http.Request) {
	role := r.URL.Query().Get("role")
	if role == "hirer" {
		role = "employer"
	}
	mode := r.URL.Query().Get("mode")
	if mode != "signup" {
		mode = "login"
	}
	if h.google == nil {
		ssoFail(w, r, "not_configured", role, mode)
		return
	}
	if role != "employer" && role != "applicant" {
		ssoFail(w, r, "invalid_role", "", mode)
		return
	}
	state, err := randomState()
	if err != nil {
		ssoFail(w, r, "server_error", role, mode)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: googleStateCookie, Value: state + "|" + role + "|" + mode, Path: "/api/auth/google/callback", MaxAge: 600, HttpOnly: true, Secure: h.CookieSecure, SameSite: http.SameSiteLaxMode})
	q := url.Values{"client_id": {h.google.ClientID}, "redirect_uri": {h.google.RedirectURL}, "response_type": {"code"}, "scope": {"openid email profile"}, "state": {state}, "prompt": {"select_account"}}
	http.Redirect(w, r, h.google.AuthURI+"?"+q.Encode(), http.StatusFound)
}

func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	if h.google == nil {
		ssoFail(w, r, "not_configured", "", "")
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
		ssoFail(w, r, "invalid_state", "", "")
		return
	}
	role, mode := parts[1], parts[2]
	if e := r.URL.Query().Get("error"); e != "" {
		ssoFail(w, r, "cancelled", role, mode)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	token, err := h.googleToken(ctx, r.URL.Query().Get("code"))
	if err != nil {
		ssoFail(w, r, "failed", role, mode)
		return
	}
	profile, err := h.googleProfile(ctx, token)
	if err != nil || !profile.EmailVerified || profile.Email == "" {
		ssoFail(w, r, "email_unverified", role, mode)
		return
	}
	email := normalizeEmail(profile.Email)
	u, err := h.Store.UserByEmailRole(ctx, email, role)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		ssoFail(w, r, "server_error", role, mode)
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		if mode != "signup" {
			ssoFail(w, r, "no_account", role, mode)
			return
		}
		u = &store.User{Email: email, Role: role, Name: strings.TrimSpace(profile.Name)}
		if u.Name == "" {
			u.Name = email
		}
		if err = h.Store.CreateUserCapped(ctx, u, h.MaxUsers); err != nil {
			if errors.Is(err, store.ErrBetaFull) {
				ssoFail(w, r, "beta_full", role, mode)
				return
			}
			if errors.Is(err, store.ErrEmailTaken) {
				ssoFail(w, r, "email_taken", role, mode)
				return
			}
			ssoFail(w, r, "server_error", role, mode)
			return
		}
	}
	if err := h.startSession(w, r, u); err != nil {
		ssoFail(w, r, "server_error", role, mode)
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
