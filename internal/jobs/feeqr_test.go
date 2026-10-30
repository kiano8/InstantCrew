package jobs

import "testing"

// The generated payloads must match the static GCash QRs, checksum included.
func TestQRPhPayload(t *testing.T) {
	for centavos, want := range map[int64]string{
		10000: "00020101021227830012com.p2pqrpay0111GXCHPHM2XXX02089996440303152170200000006560417DWQM4TK3JDNX06S0K5204601653036085406100.005802PH5913CA*I JU**A T.6009GUADALUPE610412346304B4E3",
		1000:  "00020101021227830012com.p2pqrpay0111GXCHPHM2XXX02089996440303152170200000006560417DWQM4TK3JDNX06S0K520460165303608540510.005802PH5913CA*I JU**A T.6009GUADALUPE6104123463047C1B",
	} {
		if got := qrPhPayload(centavos); got != want {
			t.Errorf("qrPhPayload(%d) =\n%s\nwant\n%s", centavos, got, want)
		}
	}
}
