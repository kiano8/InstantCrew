package verifier

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func makeJWT(key, alg string, claims map[string]any) string {
	enc := func(v any) string {
		b, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	h := enc(map[string]string{"alg": alg, "typ": "JWT"})
	p := enc(claims)
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(h + "." + p))
	return h + "." + p + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifyHTTPSMSToken(t *testing.T) {
	now := time.Now()
	good := makeJWT("secret", "HS256", map[string]any{"exp": now.Add(time.Hour).Unix()})
	noExp := makeJWT("secret", "HS256", map[string]any{"sub": "httpsms"})

	if !VerifyHTTPSMSToken("secret", good, now) {
		t.Fatal("valid token rejected")
	}
	if !VerifyHTTPSMSToken("secret", "Bearer "+good, now) {
		t.Fatal("valid token with Bearer prefix rejected")
	}
	if !VerifyHTTPSMSToken("secret", "bearer "+noExp, now) {
		t.Fatal("valid token without exp rejected")
	}
	if VerifyHTTPSMSToken("wrong", good, now) {
		t.Fatal("wrong key accepted")
	}
	if VerifyHTTPSMSToken("", good, now) {
		t.Fatal("empty key accepted")
	}
	expired := makeJWT("secret", "HS256", map[string]any{"exp": now.Add(-time.Hour).Unix()})
	if VerifyHTTPSMSToken("secret", expired, now) {
		t.Fatal("expired token accepted")
	}
	// alg=none with an empty signature must never pass.
	none := makeJWT("secret", "none", map[string]any{"exp": now.Add(time.Hour).Unix()})
	if VerifyHTTPSMSToken("secret", none, now) {
		t.Fatal("alg none accepted")
	}
	// Tampered payload: swap in a different payload, keep the old signature.
	parts := splitDots(good)
	forged := parts[0] + "." + base64.RawURLEncoding.EncodeToString([]byte(`{"exp":9999999999}`)) + "." + parts[2]
	if VerifyHTTPSMSToken("secret", forged, now) {
		t.Fatal("tampered payload accepted")
	}
	for _, junk := range []string{"", "abc", "a.b", "a.b.c", "Bearer "} {
		if VerifyHTTPSMSToken("secret", junk, now) {
			t.Fatalf("junk %q accepted", junk)
		}
	}
}

func splitDots(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '.' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	return append(out, cur)
}

func TestHTTPSMSEventParses(t *testing.T) {
	raw := `{"data":{"contact":"GCash","content":"You have received PHP 100.37 of GCash from X. Ref. No. 1234567890123.",
	"message_id":"0b0123bb","owner":"+639171234567","sim":"SIM1","timestamp":"2026-10-18T03:21:19.814Z","user_id":"u1"},
	"datacontenttype":"application/json","id":"f4aed1d3","source":"/v1/messages/receive","specversion":"1.0",
	"time":"2026-10-18T03:21:19Z","type":"message.phone.received"}`
	var ev HTTPSMSEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Type != "message.phone.received" || ev.Data.Contact != "GCash" || ev.Data.MessageID != "0b0123bb" ||
		ev.Data.Content == "" {
		t.Fatalf("unexpected parse: %+v", ev)
	}
}
