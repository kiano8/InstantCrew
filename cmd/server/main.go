package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"instantcrew/internal/auth"
	"instantcrew/internal/store"
)

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// Only these files are public. The Go source and the database are never served.
var pages = map[string]bool{
	"index.html": true, "login.html": true,
	"employer.html": true, "applicant.html": true,
}

func main() {
	port := getenv("PORT", "3000")
	env := getenv("ENV", "dev")
	dbPath := getenv("DB_PATH", "data/instantcrew.db")

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatal(err)
	}
	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	authH := &auth.Handler{
		Store:        st,
		SessionTTL:   7 * 24 * time.Hour,
		CookieSecure: getenv("COOKIE_SECURE", "false") == "true", // set true behind HTTPS
	}
	if env == "dev" {
		if err := authH.SeedDemo(context.Background()); err != nil {
			log.Fatal(err)
		}
	}

	mux := http.NewServeMux()

	// API
	mux.HandleFunc("POST /api/auth/signup", authH.Signup)
	mux.HandleFunc("POST /api/auth/login", authH.Login)
	mux.HandleFunc("POST /api/auth/logout", authH.Logout)
	mux.Handle("GET /api/auth/me", authH.RequireAuth(http.HandlerFunc(authH.Me)))

	// Frontend
	static := http.FileServer(http.Dir("."))
	for _, p := range []string{"/css/", "/js/", "/images/"} {
		mux.Handle("GET "+p, static)
	}
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
	mux.HandleFunc("GET /{page}", func(w http.ResponseWriter, r *http.Request) {
		p := r.PathValue("page")
		if !pages[p] {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, p)
	})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() { // purge expired sessions hourly
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				_ = st.DeleteExpiredSessions(context.Background())
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		log.Printf("listening on http://localhost:%s (env=%s)", port, env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
