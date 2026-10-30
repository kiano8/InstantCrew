package jobs

import (
	"fmt"
	"net/http"

	qrcode "github.com/skip2/go-qrcode"

	"instantcrew/internal/auth"
)

// The QR Ph (InstaPay) payload of the GCash account that receives search fees,
// split around tag 54 (amount). Taken from images/payment_qr_100.png.
const (
	gcashQRHead = "00020101021227830012com.p2pqrpay0111GXCHPHM2XXX02089996440303152170200000006560417DWQM4TK3JDNX06S0K52046016" + "5303608"
	gcashQRTail = "5802PH5913CA*I JU**A T.6009GUADALUPE61041234"
)

// qrPhPayload is the GCash payload for paying exactly centavos, ending in the
// tag 63 checksum (CRC-16/CCITT-FALSE over everything before it).
func qrPhPayload(centavos int64) string {
	amount := fmt.Sprintf("%d.%02d", centavos/100, centavos%100)
	p := fmt.Sprintf("%s54%02d%s%s6304", gcashQRHead, len(amount), amount, gcashQRTail)
	crc := uint16(0xFFFF)
	for i := 0; i < len(p); i++ {
		crc ^= uint16(p[i]) << 8
		for b := 0; b < 8; b++ {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return fmt.Sprintf("%s%04X", p, crc)
}

// GET /api/employer/fee-qr is a GCash QR pre-filled with the employer's next
// search fee: the base fee plus any owed cancellation penalties.
func (h *Handler) FeeQR(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	penalty, err := h.Store.EmployerSearchPenalty(r.Context(), u.ID)
	if err != nil {
		h.fail(w, err)
		return
	}
	png, err := qrcode.Encode(qrPhPayload(h.MatchFeeCentavos+penalty), qrcode.Medium, 512)
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png)
}
