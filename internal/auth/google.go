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
// response, keeping the provider, role and mode so the page can offer the right retry.
func ssoFail(w http.ResponseWriter, r *http.Request, provider, reason, role, mode string) {
	q := url.Values{"reason": {reason}, "provider": {provider}}
	if role != "" {
		q.Set("role", role)
	}
	if mode != "" {
		q.Set("mode", mode)
	}
	http.Redirect(w, r, "/sso-error.html?"+q.Encode(), http.StatusSeeOther)
}

func ssoStateCookie(provider string) string  { return "ic_" + provider + "_oauth_state" }
func ssoCallbackPath(provider string) string { return "/api/auth/" + provider + "/callback" }

// ssoStart validates ?role and ?mode, stores the OAuth state in a short-lived
// cookie scoped to the provider's callback, and returns the state to send.
func (h *Handler) ssoStart(w http.ResponseWriter, r *http.Request, provider string, configured bool) (string, bool) {
	role := r.URL.Query().Get("role")
	if role == "hirer" {
		role = "employer"
	}
	mode := r.URL.Query().Get("mode")
	if mode != "signup" {
		mode = "login"
	}
	if !configured {
		ssoFail(w, r, provider, "not_configured", role, mode)
		return "", false
	}
	if role != "employer" && role != "applicant" {
		ssoFail(w, r, provider, "invalid_role", "", mode)
		return "", false
	}
	state, err := randomState()
	if err != nil {
		ssoFail(w, r, provider, "server_error", role, mode)
		return "", false
	}
	http.SetCookie(w, &http.Cookie{Name: ssoStateCookie(provider), Value: state + "|" + role + "|" + mode, Path: ssoCallbackPath(provider), MaxAge: 600, HttpOnly: true, Secure: h.CookieSecure, SameSite: http.SameSiteLaxMode})
	return state, true
}

// ssoCallbackState checks the state cookie ssoStart set and returns the role
// and mode it carried. On false the browser has already been redirected.
func (h *Handler) ssoCallbackState(w http.ResponseWriter, r *http.Request, provider string) (role, mode string, ok bool) {
	cookie, err := r.Cookie(ssoStateCookie(provider))
	http.SetCookie(w, &http.Cookie{Name: ssoStateCookie(provider), Value: "", Path: ssoCallbackPath(provider), MaxAge: -1, HttpOnly: true, Secure: h.CookieSecure, SameSite: http.SameSiteLaxMode})
	parts := []string{}
	if err == nil {
		parts = strings.Split(cookie.Value, "|")
	}
	if len(parts) != 3 || r.URL.Query().Get("state") == "" || r.URL.Query().Get("state") != parts[0] {
		ssoFail(w, r, provider, "invalid_state", "", "")
		return "", "", false
	}
	role, mode = parts[1], parts[2]
	if r.URL.Query().Get("error") != "" {
		ssoFail(w, r, provider, "cancelled", role, mode)
		return "", "", false
	}
	return role, mode, true
}

// finishSSO signs in the account for a provider-confirmed email, creating it on
// signup (Gmail only, like password signup), and sends the browser to its portal.
func (h *Handler) finishSSO(ctx context.Context, w http.ResponseWriter, r *http.Request, provider, email, name, role, mode string) {
	email = normalizeEmail(email)
	u, err := h.Store.UserByEmailRole(ctx, email, role)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		ssoFail(w, r, provider, "server_error", role, mode)
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		if mode != "signup" {
			ssoFail(w, r, provider, "no_account", role, mode)
			return
		}
		if !isGmail(email) {
			ssoFail(w, r, provider, "gmail_only", role, mode)
			return
		}
		u = &store.User{Email: email, Role: role, Name: strings.TrimSpace(name)}
		if u.Name == "" {
			u.Name = email
		}
		if err = h.Store.CreateUserCapped(ctx, u, h.MaxUsers); err != nil {
			if errors.Is(err, store.ErrBetaFull) {
				ssoFail(w, r, provider, "beta_full", role, mode)
				return
			}
			if errors.Is(err, store.ErrEmailTaken) {
				ssoFail(w, r, provider, "email_taken", role, mode)
				return
			}
			ssoFail(w, r, provider, "server_error", role, mode)
			return
		}
		if h.OnUserCreated != nil {
			h.OnUserCreated(u.Role)
		}
	}
	if err := h.startSession(w, r, u); err != nil {
		ssoFail(w, r, provider, "server_error", role, mode)
		return
	}
	if u.Role == "employer" {
		http.Redirect(w, r, "/employer.html", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/applicant.html", http.StatusSeeOther)
	}
}

func (h *Handler) GoogleStart(w http.ResponseWriter, r *http.Request) {
	state, ok := h.ssoStart(w, r, "google", h.google != nil)
	if !ok {
		return
	}
	q := url.Values{"client_id": {h.google.ClientID}, "redirect_uri": {h.google.RedirectURL}, "response_type": {"code"}, "scope": {"openid email profile"}, "state": {state}, "prompt": {"select_account"}}
	http.Redirect(w, r, h.google.AuthURI+"?"+q.Encode(), http.StatusFound)
}

func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	if h.google == nil {
		ssoFail(w, r, "google", "not_configured", "", "")
		return
	}
	role, mode, ok := h.ssoCallbackState(w, r, "google")
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	token, err := h.googleToken(ctx, r.URL.Query().Get("code"))
	if err != nil {
		ssoFail(w, r, "google", "failed", role, mode)
		return
	}
	profile, err := h.googleProfile(ctx, token)
	if err != nil || !profile.EmailVerified || profile.Email == "" {
		ssoFail(w, r, "google", "email_unverified", role, mode)
		return
	}
	h.finishSSO(ctx, w, r, "google", profile.Email, profile.Name, role, mode)
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
