package parser

import (
	"errors"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		msg     string
		amount  int64
		ref     string
		payer   string
		wantErr bool
	}{
		{
			name:   "standard",
			msg:    "You have received PHP 100.37 of GCash from JU**N D. 09171234567. Ref. No. 1234567890123.",
			amount: 10037, ref: "1234567890123", payer: "09171234567",
		},
		{
			name:   "with balance after, must use received amount",
			msg:    "You have received PHP 250.00 of GCash from JUAN D. 09171234567. Your new balance is PHP 1,999.99. Ref. No. 9876543210123.",
			amount: 25000, ref: "9876543210123", payer: "09171234567",
		},
		{
			name:   "thousands separator",
			msg:    "You have received PHP 1,250.50 of GCash from MA** S. Ref No 1112223334445",
			amount: 125050, ref: "1112223334445",
		},
		{
			name:   "peso sign and +63 number",
			msg:    "You have received ₱75.05 of GCash from AN** B. +639181234567. Ref. No. 5556667778889.",
			amount: 7505, ref: "5556667778889", payer: "09181234567",
		},
		{
			name:   "whole peso, no decimals",
			msg:    "You have received PHP 500 of GCash from X. Ref. No. 1234567890.",
			amount: 50000, ref: "1234567890",
		},
		{name: "sent money is ignored", msg: "You have sent PHP 100.00 of GCash to JUAN D. Ref. No. 1234567890123.", wantErr: true},
		{name: "otp is ignored", msg: "Your GCash OTP is 123456. Do not share it.", wantErr: true},
		{name: "random text", msg: "Hello, are you free tonight?", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.msg)
			if tc.wantErr {
				if !errors.Is(err, ErrNotPayment) {
					t.Fatalf("want ErrNotPayment, got %v / %+v", err, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.AmountCentavos != tc.amount {
				t.Errorf("amount = %d, want %d", got.AmountCentavos, tc.amount)
			}
			if got.Ref != tc.ref {
				t.Errorf("ref = %q, want %q", got.Ref, tc.ref)
			}
			if got.PayerNumber != tc.payer {
				t.Errorf("payer = %q, want %q", got.PayerNumber, tc.payer)
			}
		})
	}
}
