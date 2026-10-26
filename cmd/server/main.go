package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"instantcrew/internal/auth"
	"instantcrew/internal/jobs"
	"instantcrew/internal/store"
)

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
		log.Printf("ignoring invalid %s=%q, using %d", k, v, def)
	}
	return def
}

func googleCredentialsPath() string {
	if configured := os.Getenv("GOOGLE_OAUTH_CREDENTIALS"); configured != "" {
		return configured
	}
	name := "client_secret_95317878064-1pd11edtj1i76buqrpbhj752gat1jmuf.apps.googleusercontent.com.json"
	var roots []string
	if cwd, err := os.Getwd(); err == nil {
		roots = append(roots, cwd)
	}
	if exe, err := os.Executable(); err == nil {
		roots = append(roots, filepath.Dir(exe))
	}
	seen := make(map[string]bool)
	for _, root := range roots {
		for i := 0; i < 6 && root != "." && root != string(filepath.Separator); i++ {
			if !seen[root] {
				seen[root] = true
				candidate := filepath.Join(root, name)
				if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
					return candidate
				}
			}
			parent := filepath.Dir(root)
			if parent == root {
				break
			}
			root = parent
		}
	}
	return name
}

// Only these files are public. The Go source and the database are never served.
var pages = map[string]bool{
	"index.html": true, "login.html": true,
	"employer.html": true, "applicant.html": true,
	"admin.html": true, "accounting.html": true,
	"privacy.html": true, "terms.html": true,
}

func main() {
	port := getenv("PORT", "3000")
	env := getenv("ENV", "dev")
	dbPath := getenv("DB_PATH", "data/instantcrew.db")

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll("uploads", 0o755); err != nil {
		log.Fatal(err)
	}
	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	// Beta settings (override with env vars).
	idleTimeout := time.Duration(getenvInt("SESSION_IDLE_MINUTES", 15)) * time.Minute
	st.SetSessionIdleTimeout(idleTimeout)

	if err := jobs.SyncCatalog(context.Background(), st); err != nil {
		log.Fatal(err)
	}

	adminEmail := getenv("ADMIN_EMAIL", "admin@instantcrew.com")
	adminPass := os.Getenv("ADMIN_PASSWORD")
	if adminPass == "" && env == "dev" {
		adminPass = "admin123"
		log.Printf("dev admin login: %s / admin123 (set ADMIN_PASSWORD in production)", adminEmail)
	}
	if adminPass != "" {
		if err := st.EnsureAdminUser(context.Background(), adminEmail, adminPass, "Instant Crew Administrator"); err != nil {
			log.Printf("warning: seed admin account: %v", err)
		}
	} else {
		log.Printf("ADMIN_PASSWORD is unset; admin account will not be auto-created")
	}
	if accPass := os.Getenv("ACCOUNTING_PASSWORD"); accPass != "" {
		accEmail := getenv("ACCOUNTING_EMAIL", "accounting@instantcrew.com")
		if err := st.EnsureAdminUser(context.Background(), accEmail, accPass, "Accounting Manager"); err != nil {
			log.Printf("warning: seed accounting account: %v", err)
		}
	}

	authH := &auth.Handler{
		Store:        st,
		SessionTTL:   7 * 24 * time.Hour,
		IdleTimeout:  idleTimeout,
		MaxUsers:     getenvInt("BETA_MAX_USERS", 300),
		CookieSecure: getenv("COOKIE_SECURE", "false") == "true", // set true behind HTTPS
	}
	googleID, googleSecret := os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET")
	var googleErr error
	if googleID != "" || googleSecret != "" {
		googleErr = authH.ConfigureGoogle(googleID, googleSecret, "", "", getenv("GOOGLE_REDIRECT_URL", ""), nil)
	} else {
		googleErr = authH.ConfigureGoogleFromFile(googleCredentialsPath(), getenv("GOOGLE_REDIRECT_URL", ""))
	}
	if googleErr != nil {
		log.Printf("Google SSO disabled: %v", googleErr)
	}
	// Development mode must not create implicit accounts. Authentication always
	// uses credentials explicitly written through the signup endpoint.

	jobsH := &jobs.Handler{
		Store:            st,
		MatchFeeCentavos: 10000,
		SearchWindow:     3 * time.Minute,
	}

	mux := http.NewServeMux()

	jobsH.Register(mux, authH)
	// API
	mux.HandleFunc("POST /api/auth/signup", authH.Signup)
	mux.HandleFunc("POST /api/auth/login", authH.Login)
	mux.HandleFunc("POST /api/auth/logout", authH.Logout)
	mux.HandleFunc("GET /api/auth/google", authH.GoogleStart)
	mux.HandleFunc("GET /api/auth/google/callback", authH.GoogleCallback)
	mux.Handle("GET /api/auth/me", authH.RequireAuth(http.HandlerFunc(authH.Me)))
	mux.Handle("POST /api/auth/activity", authH.RequireAuth(http.HandlerFunc(authH.Activity)))

	// Frontend
	static := http.FileServer(http.Dir("."))
	for _, p := range []string{"/css/", "/js/", "/images/", "/uploads/"} {
		mux.Handle("GET "+p, static)
	}
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
	mux.HandleFunc("GET /admin", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "admin.html")
	})
	mux.HandleFunc("GET /accounting", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "admin.html")
	})
	mux.HandleFunc("GET /privacy", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "privacy.html")
	})
	mux.HandleFunc("GET /terms", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "terms.html")
	})
	mux.HandleFunc("GET /{page}", func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Clean(r.PathValue("page"))
		if filepath.Dir(p) != "." || strings.HasPrefix(p, ".") {
			http.NotFound(w, r)
			return
		}
		if p == "accounting.html" {
			http.ServeFile(w, r, "admin.html")
			return
		}
		if !pages[p] {
			if strings.HasSuffix(strings.ToLower(p), ".html") {
				if info, err := os.Stat(p); err == nil && !info.IsDir() {
					http.ServeFile(w, r, p)
					return
				}
			}
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

	go func() { // purge expired and idle sessions every minute
		t := time.NewTicker(time.Minute)
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
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				_, _ = st.ExpireSearches(context.Background())
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
