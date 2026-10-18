package verifier

import (
	"context"
	"testing"
	"time"

	"instantcrew/gcash-verifier/internal/store"
)

func TestStaleSMSCannotSettleNewPayment(t *testing.T) {
	svc, st := newSvc(t)
	ctx := context.Background()

	p, err := st.CreatePayment(ctx, "job-1", 10000, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	sms := "You have received PHP " + p.PayAmount + " of GCash from X. Ref. No. 3334445556667."

	// An SMS from an hour ago (e.g. replayed after an agent restart) must not pay a payment created just now.
	o, err := svc.Handle(ctx, Incoming{MessageID: "old1", Sender: "GCash", Message: sms, ReceivedAt: time.Now().Add(-time.Hour)})
	if err != nil || o.Result != "unmatched" {
		t.Fatalf("stale SMS must not settle: %+v err=%v", o, err)
	}

	// A fresh SMS settles it, and the OnPaid hook fires with the payment.
	paid := make(chan *store.Payment, 1)
	svc.OnPaid = func(p *store.Payment) { paid <- p }

	o, err = svc.Handle(ctx, Incoming{MessageID: "new1", Sender: "GCash", Message: sms, ReceivedAt: time.Now()})
	if err != nil || o.Result != "paid" {
		t.Fatalf("fresh SMS should settle: %+v err=%v", o, err)
	}
	select {
	case got := <-paid:
		if got.ID != p.ID || got.Label != "job-1" || got.GcashRef != "3334445556667" {
			t.Fatalf("unexpected payment in callback: %+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("OnPaid was not called")
	}
}

func TestPhoneClockSkewIsTolerated(t *testing.T) {
	svc, st := newSvc(t)
	ctx := context.Background()
	p, _ := st.CreatePayment(ctx, "x", 10000, 30*time.Minute)
	// Phone clock 2 minutes behind: SMS timestamp slightly before the payment was created.
	o, _ := svc.Handle(ctx, Incoming{MessageID: "skew1", Sender: "GCash",
		Message:    "You have received PHP " + p.PayAmount + " of GCash from X. Ref. No. 4445556667778.",
		ReceivedAt: time.Now().Add(-2 * time.Minute)})
	if o.Result != "paid" {
		t.Fatalf("2 min clock skew should be tolerated, got %+v", o)
	}
}
