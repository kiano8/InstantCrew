package push

import (
	"testing"

	"instantcrew/internal/store"
)

func TestValidEndpoint(t *testing.T) {
	ok := []string{
		"https://fcm.googleapis.com/fcm/send/abc:def",
		"https://updates.push.services.mozilla.com/wpush/v2/gAAAA",
		"https://web.push.apple.com/QGuQyavXutnMH",
		"https://wns2-par02p.notify.windows.com/w/?token=x",
	}
	for _, e := range ok {
		if err := ValidEndpoint(e); err != nil {
			t.Errorf("ValidEndpoint(%q) = %v, want nil", e, err)
		}
	}
	bad := []string{
		"",
		"http://fcm.googleapis.com/fcm/send/abc", // not https
		"https://127.0.0.1:5432/",                // internal host
		"https://169.254.169.254/metadata",       // cloud metadata
		"https://fcm.googleapis.com.evil.example/x",    // suffix trick
		"https://evilpush.apple.com.example/x",         // suffix trick
		"https://notpush.apple.com/x",                  // lookalike without the dot
		"https://fcm.googleapis.com:8443/fcm/send/abc", // custom port
		"https://user:pw@fcm.googleapis.com/fcm/send/abc",
	}
	for _, e := range bad {
		if err := ValidEndpoint(e); err == nil {
			t.Errorf("ValidEndpoint(%q) = nil, want error", e)
		}
	}
}

func TestMessageLinksToJob(t *testing.T) {
	m := message(store.PendingPush{ID: 7, Title: "Shift accepted", Message: "Ana accepted.", Payload: `{"job_id":42}`})
	if m["url"] != "/employer.html?job=42" || m["tag"] != "employer-7" {
		t.Fatalf("message = %v", m)
	}
	m = message(store.PendingPush{ID: 8, Payload: `{"payment_type":"applicant","job_id":null}`})
	if m["url"] != "/employer.html" {
		t.Fatalf("url = %q, want /employer.html", m["url"])
	}
}
