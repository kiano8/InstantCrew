// Package telemetry records aggregate request metrics and heartbeats in the
// server's own database and serves them to the admin dashboard.
package telemetry

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"log"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"instantcrew/internal/auth"
	"instantcrew/internal/httpx"
	"instantcrew/internal/store"
)

const (
	reportRoute   = "GET /api/admin/telemetry"
	slowThreshold = time.Second
	retention     = 35 * 24 * time.Hour
	maxSeen       = 100_000 // caps memory if someone floods us with made-up clients
)

type Recorder struct {
	st      *store.Store
	started time.Time
	salt    [32]byte

	mu       sync.Mutex
	counts   map[store.TelemetryKey]store.TelemetryCount
	visitors map[int64]int64
	seen     map[[32]byte]struct{}
	seenDay  int64
}

func New(st *store.Store) *Recorder {
	t := &Recorder{
		st:       st,
		started:  time.Now(),
		counts:   map[store.TelemetryKey]store.TelemetryCount{},
		visitors: map[int64]int64{},
		seen:     map[[32]byte]struct{}{},
	}
	_, _ = rand.Read(t.salt[:])
	return t
}

type statusWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (w *statusWriter) WriteHeader(code int) {
	if !w.wrote && code >= 200 {
		w.status, w.wrote = code, true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	w.wrote = true
	return w.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach Flush and SetWriteDeadline, which
// the admin payment stream needs.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// Middleware must wrap the ServeMux: the mux sets r.Pattern on this same request.
func (t *Recorder) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			v := recover()
			if v != nil {
				sw.status = http.StatusInternalServerError
			}
			// The dashboard's own polling is not traffic, and an event stream's
			// duration is how long a tab stayed open, not a response time.
			if r.Pattern != reportRoute && !strings.HasPrefix(w.Header().Get("Content-Type"), "text/event-stream") {
				t.record(r, start, sw.status)
			}
			if v != nil {
				panic(v)
			}
		}()
		next.ServeHTTP(sw, r)
	})
}

func (t *Recorder) record(r *http.Request, start time.Time, status int) {
	route := r.Pattern
	if route == "" {
		route = "(unmatched)" // raw paths are unbounded and may hold tokens
	}
	elapsed := time.Since(start)
	key := store.TelemetryKey{Minute: start.Unix() / 60 * 60, Route: route, Status: status / 100 * 100}
	hash := t.visitorHash(r)
	day := store.ManilaDay(start.Unix())

	t.mu.Lock()
	defer t.mu.Unlock()
	c := t.counts[key]
	c.Requests++
	c.DurationMS += elapsed.Milliseconds()
	if elapsed >= slowThreshold {
		c.Slow++
	}
	t.counts[key] = c

	if day != t.seenDay {
		t.seenDay = day
		clear(t.seen)
	}
	if _, ok := t.seen[hash]; !ok && len(t.seen) < maxSeen {
		t.seen[hash] = struct{}{}
		// shortcut: the seen set is memory-only, so a restart recounts that day's
		// returning visitors. Persist salted hashes if deploys become frequent.
		t.visitors[day]++
	}
}

// visitorHash identifies a browser for one process lifetime. The salt is never
// stored, so no IP can be recovered from it.
func (t *Recorder) visitorHash(r *http.Request) [32]byte {
	// The app listens on localhost only and Caddy overwrites X-Forwarded-For.
	ip, _, _ := strings.Cut(r.Header.Get("X-Forwarded-For"), ",")
	if ip == "" {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	h := sha256.New()
	h.Write(t.salt[:])
	h.Write([]byte(strings.TrimSpace(ip) + "\x00" + r.UserAgent()))
	return [32]byte(h.Sum(nil))
}

// Flush saves pending counts and a heartbeat. On failure the counts are kept for
// the next attempt, so a database outage loses no traffic data.
func (t *Recorder) Flush(ctx context.Context) error {
	t.mu.Lock()
	counts, visitors := t.counts, t.visitors
	t.counts = map[store.TelemetryKey]store.TelemetryCount{}
	t.visitors = map[int64]int64{}
	t.mu.Unlock()

	err := t.st.SaveTelemetry(ctx, time.Now().Unix()/60*60, counts, visitors)
	if err != nil {
		t.mu.Lock()
		for k, c := range counts {
			cur := t.counts[k]
			cur.Requests += c.Requests
			cur.DurationMS += c.DurationMS
			cur.Slow += c.Slow
			t.counts[k] = cur
		}
		for d, n := range visitors {
			t.visitors[d] += n
		}
		t.mu.Unlock()
	}
	return err
}

// Run flushes every minute until ctx ends. Call Flush once more after the HTTP
// server has shut down so the final requests are kept.
func (t *Recorder) Run(ctx context.Context) {
	flush := func() {
		fctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := t.Flush(fctx); err != nil && ctx.Err() == nil {
			log.Printf("telemetry flush: %v", err)
		}
	}
	flush()
	beat := time.NewTicker(time.Minute)
	defer beat.Stop()
	purge := time.NewTicker(time.Hour)
	defer purge.Stop()
	for {
		select {
		case <-beat.C:
			flush()
		case <-purge.C:
			if err := t.st.DeleteTelemetryBefore(ctx, time.Now().Add(-retention).Unix()); err != nil && ctx.Err() == nil {
				log.Printf("telemetry purge: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

var ranges = map[string]struct {
	step   int64
	points int
}{
	"1h":  {60, 60},
	"24h": {900, 96},
	"7d":  {3600, 168},
	"30d": {86400, 30},
}

func (t *Recorder) Register(mux *http.ServeMux, a *auth.Handler) {
	mux.Handle(reportRoute, a.RequireRole("admin", http.HandlerFunc(t.report)))
}

func (t *Recorder) report(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("range")
	rng, ok := ranges[name]
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "range must be 1h, 24h, 7d or 30d")
		return
	}
	// Flushing first shows the current minute; a failure here means the database
	// is unreachable, which the dashboard reports as an outage.
	if err := t.Flush(r.Context()); err != nil {
		log.Printf("telemetry report: %v", err)
		httpx.Error(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	report, err := t.st.TelemetryReport(r.Context(), time.Now(), rng.step, rng.points)
	if err != nil {
		log.Printf("telemetry report: %v", err)
		httpx.Error(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"report":     report,
		"now":        time.Now().Unix(),
		"step":       rng.step,
		"started_at": t.started.Unix(),
		"goroutines": runtime.NumGoroutine(),
		"memory_mb":  mem.Alloc / (1 << 20),
	})
}
