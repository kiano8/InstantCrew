// Package parser extracts payment details from GCash "you have received" SMS text.
//
// GCash wording changes over time, so the patterns are deliberately loose.
// After you capture real messages from your phone (see sms_log), add them to
// parser_test.go and tighten or extend the regexes as needed.
package parser

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var ErrNotPayment = errors.New("not a GCash received-money message")

type Result struct {
	AmountCentavos int64
	Ref            string // GCash reference number, "" if absent
	PayerNumber    string // normalized 09XXXXXXXXX, "" if masked/absent
}

var (
	// "You have received PHP 100.37 of GCash from ..."
	reAmount = regexp.MustCompile(`(?i)\b(?:received|nareceive)\s+(?:php|₱|p)\s*([0-9][0-9,]*(?:\.[0-9]{1,2})?)`)
	// "Ref. No. 1234567890123" / "Ref No 123..." / "Reference Number: 123..."
	reRef = regexp.MustCompile(`(?i)\bref(?:erence)?\.?\s*(?:no\.?|number|#)?\s*:?\s*([0-9]{6,})`)
	// 09171234567 / +639171234567 / 639171234567, not part of a longer number
	rePhone = regexp.MustCompile(`(?:^|[^0-9])((?:\+?63|0)9[0-9]{9})(?:[^0-9]|$)`)
)

func Parse(msg string) (*Result, error) {
	m := reAmount.FindStringSubmatch(msg)
	if m == nil {
		return nil, ErrNotPayment
	}
	centavos, ok := parseCentavos(m[1])
	if !ok || centavos <= 0 {
		return nil, ErrNotPayment
	}
	r := &Result{AmountCentavos: centavos}
	if rm := reRef.FindStringSubmatch(msg); rm != nil {
		r.Ref = rm[1]
	}
	if pm := rePhone.FindStringSubmatch(msg); pm != nil {
		r.PayerNumber = normalizePhone(pm[1])
	}
	return r, nil
}

func parseCentavos(s string) (int64, bool) {
	s = strings.ReplaceAll(s, ",", "")
	whole, frac, _ := strings.Cut(s, ".")
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, false
	}
	for len(frac) < 2 {
		frac += "0"
	}
	f, err := strconv.ParseInt(frac[:2], 10, 64)
	if err != nil {
		return 0, false
	}
	return w*100 + f, true
}

func normalizePhone(p string) string {
	p = strings.TrimPrefix(p, "+")
	if strings.HasPrefix(p, "63") {
		return "0" + p[2:]
	}
	return p
}
