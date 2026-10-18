package verifier

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

// HTTPSMSEvent is the CloudEvent httpSMS POSTs to your callback URL.
// Only message.phone.received carries an SMS; heartbeat events leave Data mostly empty.
type HTTPSMSEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"` // message.phone.received, phone.heartbeat.offline, ...
	Data struct {
		Contact   string `json:"contact"` // who sent the SMS (for GCash: the sender ID)
		Content   string `json:"content"` // SMS text
		MessageID string `json:"message_id"`
		Owner     string `json:"owner"` // your phone's number
		SIM       string `json:"sim"`
		Timestamp string `json:"timestamp"`
	} `json:"data"`
}

// VerifyHTTPSMSToken validates the HS256 JWT httpSMS sends as
// "Authorization: Bearer <jwt>", signed with the signing key you set on the
// webhook. It rejects other algorithms (including "none"), and enforces
// exp/nbf when those claims are present.
func VerifyHTTPSMSToken(key, authHeader string, now time.Time) bool {
	if key == "" {
		return false
	}
	token := strings.TrimSpace(authHeader)
	if len(token) > 7 && strings.EqualFold(token[:7], "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}
	dec := func(s string) ([]byte, error) {
		return base64.RawURLEncoding.DecodeString(strings.TrimRight(s, "="))
	}

	hb, err := dec(parts[0])
	if err != nil {
		return false
	}
	var hdr struct {
		Alg string `json:"alg"`
	}
	if json.Unmarshal(hb, &hdr) != nil || hdr.Alg != "HS256" {
		return false
	}

	sig, err := dec(parts[2])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(mac.Sum(nil), sig) {
		return false
	}

	// Signature is good; honour time claims if present (30s clock-skew leeway).
	pb, err := dec(parts[1])
	if err != nil {
		return false
	}
	var claims struct {
		Exp float64 `json:"exp"`
		Nbf float64 `json:"nbf"`
	}
	if json.Unmarshal(pb, &claims) != nil {
		return false
	}
	const leeway = 30
	if claims.Exp > 0 && now.Unix() > int64(claims.Exp)+leeway {
		return false
	}
	if claims.Nbf > 0 && now.Unix()+leeway < int64(claims.Nbf) {
		return false
	}
	return true
}
