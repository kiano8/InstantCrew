package jobs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// paymentEvent is pushed to connected admin consoles whenever a payment
// verification request is submitted or its status changes.
type paymentEvent struct {
	Kind   string `json:"kind"` // "submitted" or "status"
	Type   string `json:"type"` // "employer" or "applicant"
	ID     int64  `json:"id,omitempty"`
	Status string `json:"status,omitempty"`
}

// paymentHub fans payment events out to admin SSE subscribers. The zero value
// is ready to use.
type paymentHub struct {
	mu   sync.Mutex
	subs map[chan paymentEvent]struct{}
}

func (hub *paymentHub) subscribe() chan paymentEvent {
	ch := make(chan paymentEvent, 16)
	hub.mu.Lock()
	if hub.subs == nil {
		hub.subs = make(map[chan paymentEvent]struct{})
	}
	hub.subs[ch] = struct{}{}
	hub.mu.Unlock()
	return ch
}

func (hub *paymentHub) unsubscribe(ch chan paymentEvent) {
	hub.mu.Lock()
	delete(hub.subs, ch)
	hub.mu.Unlock()
}

// publish never blocks: a subscriber whose buffer is full just misses the
// event, which is harmless because every event triggers a full re-sync.
func (hub *paymentHub) publish(ev paymentEvent) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	for ch := range hub.subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

// GET /api/admin/payments/stream (Server-Sent Events)
func (h *Handler) AdminPaymentStream(w http.ResponseWriter, r *http.Request) {
	rc := http.NewResponseController(w)
	// The server's WriteTimeout would otherwise cut the stream after 30s.
	_ = rc.SetWriteDeadline(time.Time{})

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering (nginx)
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "retry: 3000\n: connected\n\n")
	if err := rc.Flush(); err != nil {
		return
	}

	ch := h.payments.subscribe()
	defer h.payments.unsubscribe(ch)

	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case ev := <-ch:
			b, _ := json.Marshal(ev)
			fmt.Fprintf(w, "event: payment\ndata: %s\n\n", b)
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
		}
		if err := rc.Flush(); err != nil {
			return
		}
	}
}
