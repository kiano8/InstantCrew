// Package verifier turns incoming SMS webhooks into settled payments.
package verifier

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"strings"
	"time"

	"instantcrew/gcash-verifier/internal/parser"
	"instantcrew/gcash-verifier/internal/store"
)

// VerifySignature checks the SMSGate webhook signature:
// hex(HMAC-SHA256(key, rawBody + X-Timestamp)) must equal X-Signature.
func VerifySignature(key string, body []byte, timestamp, signatureHex string) bool {
	if key == "" || signatureHex == "" {
		return false
	}
	got, err := hex.DecodeString(strings.TrimSpace(signatureHex))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(body)
	mac.Write([]byte(timestamp))
	return hmac.Equal(mac.Sum(nil), got)
}

type Service struct {
	Store          *store.Store
	AllowedSenders []string // case-insensitive; empty means accept any sender (testing only)
	// OnPaid, if set, is called in its own goroutine after a payment is settled.
	OnPaid func(*store.Payment)
}

type Incoming struct {
	MessageID string
	Sender    string
	Message   string
	SigOK     bool
	Raw       string
	// ReceivedAt is when the phone received the SMS. Zero means unknown (check skipped).
	ReceivedAt time.Time
}

type Outcome struct {
	Result    string `json:"result"` // paid | unmatched | duplicate | duplicate_ref | ignored_sender | ignored_not_payment
	Note      string `json:"note,omitempty"`
	PaymentID string `json:"paymentId,omitempty"`
	Amount    string `json:"amount,omitempty"`
	Ref       string `json:"ref,omitempty"`
}

func (s *Service) senderAllowed(sender string) bool {
	if len(s.AllowedSenders) == 0 {
		return true
	}
	sender = strings.TrimSpace(sender)
	for _, a := range s.AllowedSenders {
		if strings.EqualFold(a, sender) {
			return true
		}
	}
	return false
}

// Handle processes one incoming SMS. A non-nil error means an internal
// failure (the webhook should return 5xx so SMSGate retries); business
// outcomes such as "unmatched" are returned in Outcome with a nil error.
func (s *Service) Handle(ctx context.Context, in Incoming) (*Outcome, error) {
	logID, err := s.Store.InsertSMS(ctx, &store.SMSLog{
		MessageID: in.MessageID, Sender: in.Sender, Body: in.Message, Raw: in.Raw, SigOK: in.SigOK,
	})
	if errors.Is(err, store.ErrDuplicate) {
		return &Outcome{Result: "duplicate", Note: "message already processed"}, nil
	}
	if err != nil {
		return nil, err
	}

	finish := func(o *Outcome, amount int64) (*Outcome, error) {
		if err := s.Store.UpdateSMSResult(ctx, logID, o.Result, o.Note, o.PaymentID, amount, o.Ref); err != nil {
			return nil, err
		}
		log.Printf("sms #%d -> %s %s", logID, o.Result, o.Note)
		return o, nil
	}

	if !s.senderAllowed(in.Sender) {
		return finish(&Outcome{Result: "ignored_sender", Note: "sender " + quote(in.Sender) + " not in allowlist"}, 0)
	}

	res, err := parser.Parse(in.Message)
	if errors.Is(err, parser.ErrNotPayment) {
		return finish(&Outcome{Result: "ignored_not_payment", Note: "not a GCash received-money message"}, 0)
	}
	if err != nil {
		return nil, err
	}

	amountStr := store.FormatPesos(res.AmountCentavos)

	if res.Ref != "" {
		used, err := s.Store.RefAlreadyUsed(ctx, res.Ref)
		if err != nil {
			return nil, err
		}
		if used {
			return finish(&Outcome{Result: "duplicate_ref", Note: "reference number already used", Amount: amountStr, Ref: res.Ref}, res.AmountCentavos)
		}
	}

	p, err := s.Store.MarkPaid(ctx, res.AmountCentavos, res.Ref, res.PayerNumber, in.ReceivedAt)
	switch {
	case errors.Is(err, store.ErrNoMatch):
		return finish(&Outcome{
			Result: "unmatched", Amount: amountStr, Ref: res.Ref,
			Note: "no pending, unexpired payment for exactly PHP " + amountStr + " (wrong amount, expired, or the SMS predates the payment) - needs manual review",
		}, res.AmountCentavos)
	case errors.Is(err, store.ErrDuplicate):
		return finish(&Outcome{Result: "duplicate_ref", Note: "reference number already used", Amount: amountStr, Ref: res.Ref}, res.AmountCentavos)
	case err != nil:
		return nil, err
	}
	if s.OnPaid != nil {
		go s.OnPaid(p)
	}
	return finish(&Outcome{Result: "paid", PaymentID: p.ID, Amount: amountStr, Ref: res.Ref}, res.AmountCentavos)
}

func quote(s string) string {
	if s == "" {
		return "(empty)"
	}
	return `"` + s + `"`
}
