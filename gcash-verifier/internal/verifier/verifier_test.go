package verifier

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"

	"instantcrew/gcash-verifier/internal/store"
)

func newSvc(t *testing.T, senders ...string) (*Service, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return &Service{Store: st, AllowedSenders: senders}, st
}

func TestVerifySignature(t *testing.T) {
	body := []byte(`{"event":"sms:received"}`)
	ts := "1700000000"
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write(body)
	mac.Write([]byte(ts))
	sig := hex.EncodeToString(mac.Sum(nil))

	if !VerifySignature("secret", body, ts, sig) {
		t.Fatal("valid signature rejected")
	}
	if VerifySignature("wrong", body, ts, sig) {
		t.Fatal("wrong key accepted")
	}
	if VerifySignature("secret", append(body, ' '), ts, sig) {
		t.Fatal("tampered body accepted")
	}
	if VerifySignature("secret", body, "1700000001", sig) {
		t.Fatal("tampered timestamp accepted")
	}
	if VerifySignature("secret", body, ts, "zz") || VerifySignature("", body, ts, sig) {
		t.Fatal("garbage accepted")
	}
}

func TestPaymentFlow(t *testing.T) {
	svc, st := newSvc(t, "GCash")
	ctx := context.Background()

	p, err := st.CreatePayment(ctx, "test", 10000, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if p.PayCentavos <= 10000 || p.PayCentavos > 10099 {
		t.Fatalf("pay amount out of range: %d", p.PayCentavos)
	}
	sms := "You have received PHP " + p.PayAmount + " of GCash from JU**N D. 09171234567. Ref. No. 1234567890123."

	// Wrong sender is ignored and does not pay.
	o, err := svc.Handle(ctx, Incoming{MessageID: "m0", Sender: "Scammer", Message: sms})
	if err != nil || o.Result != "ignored_sender" {
		t.Fatalf("want ignored_sender, got %+v err=%v", o, err)
	}

	// Correct sender settles it.
	o, err = svc.Handle(ctx, Incoming{MessageID: "m1", Sender: "gcash", Message: sms})
	if err != nil || o.Result != "paid" || o.PaymentID != p.ID {
		t.Fatalf("want paid, got %+v err=%v", o, err)
	}
	got, _ := st.GetPayment(ctx, p.ID)
	if got.Status != "paid" || got.GcashRef != "1234567890123" || got.PayerNumber != "09171234567" {
		t.Fatalf("payment not settled correctly: %+v", got)
	}

	// Retry of the same webhook (same messageId) is a no-op.
	o, _ = svc.Handle(ctx, Incoming{MessageID: "m1", Sender: "GCash", Message: sms})
	if o.Result != "duplicate" {
		t.Fatalf("want duplicate, got %+v", o)
	}

	// Same SMS replayed under a new messageId: reference already used.
	o, _ = svc.Handle(ctx, Incoming{MessageID: "m2", Sender: "GCash", Message: sms})
	if o.Result != "duplicate_ref" {
		t.Fatalf("want duplicate_ref, got %+v", o)
	}

	// Money for an amount nobody is waiting on is flagged, not lost.
	o, _ = svc.Handle(ctx, Incoming{MessageID: "m3", Sender: "GCash",
		Message: "You have received PHP 999.99 of GCash from X. Ref. No. 5550001112223."})
	if o.Result != "unmatched" {
		t.Fatalf("want unmatched, got %+v", o)
	}

	// Non-payment texts are ignored.
	o, _ = svc.Handle(ctx, Incoming{MessageID: "m4", Sender: "GCash", Message: "Your GCash OTP is 123456"})
	if o.Result != "ignored_not_payment" {
		t.Fatalf("want ignored_not_payment, got %+v", o)
	}
}

func TestPendingAmountsAreUnique(t *testing.T) {
	_, st := newSvc(t)
	ctx := context.Background()
	seen := map[int64]bool{}
	for i := 0; i < 50; i++ {
		p, err := st.CreatePayment(ctx, "x", 10000, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		if seen[p.PayCentavos] {
			t.Fatalf("duplicate pending amount %d", p.PayCentavos)
		}
		seen[p.PayCentavos] = true
	}
}

func TestExpiredPaymentCannotBePaid(t *testing.T) {
	svc, st := newSvc(t)
	ctx := context.Background()
	p, err := st.CreatePayment(ctx, "x", 10000, -time.Minute) // already expired
	if err != nil {
		t.Fatal(err)
	}
	o, _ := svc.Handle(ctx, Incoming{MessageID: "e1", Sender: "GCash",
		Message: "You have received PHP " + p.PayAmount + " of GCash from X. Ref. No. 7778889990001."})
	if o.Result != "unmatched" {
		t.Fatalf("expired payment must not settle, got %+v", o)
	}
}
