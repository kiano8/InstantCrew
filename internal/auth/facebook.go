package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// facebookGraph is the Graph API version the app is built against. Meta serves
// calls to a retired version from the oldest one still available, so bumping
// this is routine maintenance, not an outage.
const facebookGraph = "v23.0"

type facebookOAuth struct {
	AppID, AppSecret, RedirectURL string
}

// ConfigureFacebook enables "Continue with Facebook" with the Meta app's credentials.
func (h *Handler) ConfigureFacebook(appID, appSecret, redirectURL string) error {
	if appID == "" || appSecret == "" {
		return errors.New("Facebook login requires FACEBOOK_APP_ID and FACEBOOK_APP_SECRET")
	}
	if redirectURL == "" {
		return errors.New("set FACEBOOK_REDIRECT_URL, e.g. https://your-domain/api/auth/facebook/callback")
	}
	h.facebook = &facebookOAuth{appID, appSecret, redirectURL}
	return nil
}

func (h *Handler) FacebookStart(w http.ResponseWriter, r *http.Request) {
	state, ok := h.ssoStart(w, r, "facebook", h.facebook != nil)
	if !ok {
		return
	}
	q := url.Values{"client_id": {h.facebook.AppID}, "redirect_uri": {h.facebook.RedirectURL}, "response_type": {"code"}, "scope": {"email,public_profile"}, "state": {state}}
	http.Redirect(w, r, "https://www.facebook.com/"+facebookGraph+"/dialog/oauth?"+q.Encode(), http.StatusFound)
}

func (h *Handler) FacebookCallback(w http.ResponseWriter, r *http.Request) {
	if h.facebook == nil {
		ssoFail(w, r, "facebook", "not_configured", "", "")
		return
	}
	role, mode, ok := h.ssoCallbackState(w, r, "facebook")
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	token, err := h.facebookToken(ctx, r.URL.Query().Get("code"))
	if err != nil {
		ssoFail(w, r, "facebook", "failed", role, mode)
		return
	}
	profile, err := h.facebookProfile(ctx, token)
	if err != nil {
		ssoFail(w, r, "facebook", "failed", role, mode)
		return
	}
	// Facebook only returns an email the person has confirmed with them, and
	// leaves it out when there is none or they declined the email permission.
	if profile.Email == "" {
		ssoFail(w, r, "facebook", "no_email", role, mode)
		return
	}
	h.finishSSO(ctx, w, r, "facebook", profile.Email, profile.Name, role, mode)
}

func (g *facebookOAuth) get(ctx context.Context, endpoint string, q url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Facebook returned %s", resp.Status)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
}

func (h *Handler) facebookToken(ctx context.Context, code string) (string, error) {
	if code == "" {
		return "", errors.New("missing authorization code")
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	err := h.facebook.get(ctx, "https://graph.facebook.com/"+facebookGraph+"/oauth/access_token", url.Values{
		"client_id": {h.facebook.AppID}, "client_secret": {h.facebook.AppSecret},
		"redirect_uri": {h.facebook.RedirectURL}, "code": {code},
	}, &out)
	if err == nil && out.AccessToken == "" {
		err = errors.New("Facebook returned no access token")
	}
	return out.AccessToken, err
}

type facebookUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (h *Handler) facebookProfile(ctx context.Context, accessToken string) (facebookUser, error) {
	var out facebookUser
	// appsecret_proof proves the call comes from our server, so a leaked user
	// token alone can't be used against the app.
	mac := hmac.New(sha256.New, []byte(h.facebook.AppSecret))
	mac.Write([]byte(accessToken))
	err := h.facebook.get(ctx, "https://graph.facebook.com/"+facebookGraph+"/me", url.Values{
		"fields": {"name,email"}, "access_token": {accessToken}, "appsecret_proof": {hex.EncodeToString(mac.Sum(nil))},
	}, &out)
	return out, err
}
