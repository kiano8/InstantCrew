package jobs_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"instantcrew/internal/auth"
	"instantcrew/internal/jobs"
	"instantcrew/internal/store"
	"instantcrew/internal/store/storetest"
)

type testEnv struct {
	mux       *http.ServeMux
	st        *store.Store
	authH     *auth.Handler
	jobsH     *jobs.Handler
	admin     *store.User
	employer  *store.User
	applicant *store.User
}

func setupTestEnv(t *testing.T) *testEnv {
	st, _ := storetest.Open(t)

	ctx := context.Background()
	if err := jobs.SyncCatalog(ctx, st); err != nil {
		t.Fatalf("SyncCatalog: %v", err)
	}

	authH := &auth.Handler{
		Store:        st,
		SessionTTL:   24 * time.Hour,
		IdleTimeout:  1 * time.Hour,
		MaxUsers:     100,
		CookieSecure: false,
	}

	jobsH := &jobs.Handler{
		Store:            st,
		MatchFeeCentavos: 10000,
		SearchWindow:     3 * time.Minute,
	}

	mux := http.NewServeMux()
	jobsH.Register(mux, authH)
	mux.HandleFunc("POST /api/auth/signup", authH.Signup)
	mux.HandleFunc("POST /api/auth/login", authH.Login)
	mux.HandleFunc("POST /api/auth/logout", authH.Logout)
	mux.Handle("GET /api/auth/me", authH.RequireAuth(http.HandlerFunc(authH.Me)))

	if err := st.EnsureAdminUser(ctx, "admin@instantcrew.com", "admin123", "Instant Crew Administrator"); err != nil {
		t.Fatalf("EnsureAdminUser: %v", err)
	}
	adminUser, err := st.UserByEmailRole(ctx, "admin@instantcrew.com", "admin")
	if err != nil {
		t.Fatalf("UserByEmailRole admin: %v", err)
	}

	emp := &store.User{Email: "employer@test.com", PasswordHash: "x", Role: "employer", Name: "Emp", Company: "Co"}
	if err := st.CreateUser(ctx, emp); err != nil {
		t.Fatalf("CreateUser employer: %v", err)
	}

	app := &store.User{Email: "applicant@test.com", PasswordHash: "x", Role: "applicant", Name: "App"}
	if err := st.CreateUser(ctx, app); err != nil {
		t.Fatalf("CreateUser applicant: %v", err)
	}

	active := true
	profile := store.ApplicantProfile{
		EmploymentType: "full-time",
		ExpectedRate:   150,
		RatePeriod:     "hour",
		City:           "Cebu City",
		Location:       "IT Park",
		Availability:   "now",
		Active:         &active,
		Roles:          []string{"waiter-server"},
	}
	if err := st.SaveApplicantProfile(ctx, app.ID, profile); err != nil {
		t.Fatalf("SaveApplicantProfile: %v", err)
	}

	return &testEnv{
		mux:       mux,
		st:        st,
		authH:     authH,
		jobsH:     jobsH,
		admin:     adminUser,
		employer:  emp,
		applicant: app,
	}
}

func (env *testEnv) createSessionCookie(t *testing.T, u *store.User) *http.Cookie {
	rawToken := "token_" + u.Email
	h := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(h[:])
	if err := env.st.CreateSession(context.Background(), tokenHash, u.ID, time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	return &http.Cookie{
		Name:     auth.CookieName,
		Value:    rawToken,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
	}
}

func TestUploadProof(t *testing.T) {
	env := setupTestEnv(t)
	cookie := env.createSessionCookie(t, env.employer)

	// Create multipart form with a dummy image
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("proof", "receipt.jpg")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	_, _ = part.Write([]byte("fake image content for test"))
	_ = w.Close()

	req := httptest.NewRequest("POST", "/api/upload-proof", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(cookie)

	rec := httptest.NewRecorder()
	env.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var res struct {
		Success bool   `json:"success"`
		URL     string `json:"url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if !res.Success || !strings.HasPrefix(res.URL, "/uploads/proof_") {
		t.Fatalf("unexpected upload response: %+v", res)
	}
	t.Cleanup(func() { _ = os.Remove(filepath.Join(".", strings.TrimPrefix(res.URL, "/"))) })
}

func TestAdminPaymentsEndpoints(t *testing.T) {
	env := setupTestEnv(t)
	adminCookie := env.createSessionCookie(t, env.admin)
	empCookie := env.createSessionCookie(t, env.employer)
	appCookie := env.createSessionCookie(t, env.applicant)

	// 1. Non-admin accessing /api/admin/payments should be 403 Forbidden
	req := httptest.NewRequest("GET", "/api/admin/payments", nil)
	req.AddCookie(empCookie)
	rec := httptest.NewRecorder()
	env.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 forbidden for employer, got %d", rec.Code)
	}

	// 2. Admin accessing /api/admin/payments -> 200 OK
	req = httptest.NewRequest("GET", "/api/admin/payments", nil)
	req.AddCookie(adminCookie)
	rec = httptest.NewRecorder()
	env.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for admin, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. Employer creates a job
	jobPayload := map[string]any{
		"category":        "kitchen",
		"role":            "waiter-server",
		"employment_type": "full-time",
		"rate":            150,
		"rate_period":     "hour",
		"city":            "Cebu City",
		"map_location":    "IT Park, Cebu",
		"timing":          "now",
		"crew_needed":     1,
	}
	body, _ := json.Marshal(jobPayload)
	req = httptest.NewRequest("POST", "/api/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(empCookie)
	rec = httptest.NewRecorder()
	env.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create job expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var jobRes struct {
		Job struct {
			ID int64 `json:"id"`
		} `json:"job"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &jobRes)
	jobID := jobRes.Job.ID

	// 4a. A proof_image that is not an UploadProof path is refused (it is rendered on the admin page).
	for _, bad := range []string{`/uploads/proof_1_ab.jpg" onerror="alert(1)`, "https://evil.example/x.jpg", "data:image/png;base64,AAAA"} {
		body, _ = json.Marshal(map[string]any{"method": "gcash", "proof_image": bad})
		req = httptest.NewRequest("POST", "/api/jobs/"+jsonNumber(jobID)+"/pay", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(empCookie)
		rec = httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("pay with proof_image %q expected 400, got %d: %s", bad, rec.Code, rec.Body.String())
		}
	}

	// 4b. Only GCash with a receipt is accepted; Maya and card were removed.
	for _, bad := range []map[string]any{{"method": "card"}, {"method": "maya", "proof_image": "/uploads/proof_1700000000_abcdef0123456789.jpg"}, {"method": "gcash", "details": "no receipt"}} {
		for _, path := range []string{"/api/jobs/" + jsonNumber(jobID) + "/pay", "/api/applicant/hiring-pass/pay"} {
			cookie := empCookie
			if strings.Contains(path, "applicant") {
				cookie = appCookie
			}
			body, _ = json.Marshal(bad)
			req = httptest.NewRequest("POST", path, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(cookie)
			rec = httptest.NewRecorder()
			env.mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("%s with %v expected 400, got %d: %s", path, bad, rec.Code, rec.Body.String())
			}
		}
	}

	// 4. Employer pays search fee
	payPayload := map[string]any{
		"method":      "gcash",
		"details":     "GCash payment test",
		"proof_image": "/uploads/proof_1700000000_abcdef0123456789.jpg",
	}
	body, _ = json.Marshal(payPayload)
	req = httptest.NewRequest("POST", "/api/jobs/"+strings.TrimSpace(jsonNumber(jobID))+"/pay", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(empCookie)
	rec = httptest.NewRecorder()
	env.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("pay job expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// 5. Employer tries to start search -> Should fail with 402 Payment Required because pending admin verification
	req = httptest.NewRequest("POST", "/api/jobs/"+jsonNumber(jobID)+"/search", nil)
	req.AddCookie(empCookie)
	rec = httptest.NewRecorder()
	env.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("search before verify expected 402, got %d: %s", rec.Code, rec.Body.String())
	}

	// 6. Admin checks payments list
	req = httptest.NewRequest("GET", "/api/admin/payments", nil)
	req.AddCookie(adminCookie)
	rec = httptest.NewRecorder()
	env.mux.ServeHTTP(rec, req)
	var listRes struct {
		Payments []store.AdminPaymentRecord `json:"payments"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &listRes)
	if len(listRes.Payments) == 0 {
		t.Fatalf("expected payments in admin list")
	}
	var empPayID int64
	for _, p := range listRes.Payments {
		if p.Type == "employer" && p.UserID == env.employer.ID {
			empPayID = p.RawID
			if p.Status != "pending" {
				t.Fatalf("expected pending status, got %s", p.Status)
			}
		}
	}
	if empPayID == 0 {
		t.Fatalf("employer payment not found")
	}

	// 7. Admin verifies payment
	verifyPayload := map[string]any{
		"type":     "employer",
		"id":       empPayID,
		"status":   "verified",
		"verified": true,
	}
	body, _ = json.Marshal(verifyPayload)
	req = httptest.NewRequest("POST", "/api/admin/payments/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie)
	rec = httptest.NewRecorder()
	env.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin verify expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// 8. Now employer starts search -> Should succeed with 200 OK!
	req = httptest.NewRequest("POST", "/api/jobs/"+jsonNumber(jobID)+"/search", nil)
	req.AddCookie(empCookie)
	rec = httptest.NewRecorder()
	env.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("search after verify expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// 9. Applicant pays hiring pass (₱10 for 5 hires)
	appPayPayload := map[string]any{
		"method":      "gcash",
		"proof_image": "/uploads/proof_1700000001_abcdef0123456789.png",
	}
	body, _ = json.Marshal(appPayPayload)
	req = httptest.NewRequest("POST", "/api/applicant/hiring-pass/pay", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(appCookie)
	rec = httptest.NewRecorder()
	env.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("applicant hiring pass pay expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var appPayRes struct {
		Pending bool `json:"pending"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &appPayRes)
	if !appPayRes.Pending {
		t.Fatalf("expected applicant payment to be pending")
	}

	// 10. Attempting to accept before admin verifies returns 402 ErrHiringPassPending
	req = httptest.NewRequest("POST", "/api/applicant/jobs/"+jsonNumber(jobID)+"/accept", nil)
	req.AddCookie(appCookie)
	rec = httptest.NewRecorder()
	env.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("applicant accept before verify expected 402, got %d: %s", rec.Code, rec.Body.String())
	}

	// 11. Admin verifies applicant payment
	req = httptest.NewRequest("GET", "/api/admin/payments", nil)
	req.AddCookie(adminCookie)
	rec = httptest.NewRecorder()
	env.mux.ServeHTTP(rec, req)
	_ = json.Unmarshal(rec.Body.Bytes(), &listRes)
	var appPaymentID int64
	for _, p := range listRes.Payments {
		if p.Type == "applicant" && p.UserID == env.applicant.ID {
			appPaymentID = p.RawID
		}
	}
	if appPaymentID == 0 {
		t.Fatalf("applicant payment not found in admin list")
	}

	appVerifyPayload := map[string]any{
		"type":     "applicant",
		"id":       appPaymentID,
		"status":   "verified",
		"verified": true,
	}
	body, _ = json.Marshal(appVerifyPayload)
	req = httptest.NewRequest("POST", "/api/admin/payments/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie)
	rec = httptest.NewRecorder()
	env.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin verify applicant payment expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// 11b. Applicant accepts after payment verified -> 200 OK
	req = httptest.NewRequest("POST", "/api/applicant/jobs/"+jsonNumber(jobID)+"/accept", nil)
	req.AddCookie(appCookie)
	rec = httptest.NewRecorder()
	env.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("applicant accept after verify expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// 12. Accepting spent nothing: 5 top-up = 5 until the employer confirms.
	do := func(method, path string, c *http.Cookie, payload any) *httptest.ResponseRecorder {
		var rdr io.Reader
		if payload != nil {
			b, _ := json.Marshal(payload)
			rdr = bytes.NewReader(b)
		}
		r := httptest.NewRequest(method, path, rdr)
		r.Header.Set("Content-Type", "application/json")
		r.AddCookie(c)
		w := httptest.NewRecorder()
		env.mux.ServeHTTP(w, r)
		return w
	}
	rec = do("GET", "/api/applicant/hiring-pass", appCookie, nil)
	var passRes struct {
		HiringPass struct {
			RemainingHires int `json:"remainingHires"`
			BatchesPaid    int `json:"batchesPaid"`
		} `json:"hiring_pass"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &passRes)
	if passRes.HiringPass.RemainingHires != 5 || passRes.HiringPass.BatchesPaid != 1 {
		t.Fatalf("expected 5 remaining hires (none spent before confirmation) and 1 paid batch, got %+v", passRes.HiringPass)
	}

	// 13. Transit is blocked until the employer confirms the booking, and must go in order.
	transit := "/api/applicant/assignments/" + jsonNumber(jobID) + "/transit"
	if rec = do("POST", transit, appCookie, map[string]string{"status": "en_route"}); rec.Code != http.StatusConflict {
		t.Fatalf("en_route before employer confirmation expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec = do("POST", "/api/jobs/"+jsonNumber(jobID)+"/confirm", empCookie, nil); rec.Code != http.StatusOK {
		t.Fatalf("employer confirm expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	rec = do("GET", "/api/applicant/hiring-pass", appCookie, nil)
	_ = json.Unmarshal(rec.Body.Bytes(), &passRes)
	if passRes.HiringPass.RemainingHires != 4 {
		t.Fatalf("expected the hire to be spent on employer confirmation (4 left), got %d", passRes.HiringPass.RemainingHires)
	}
	if rec = do("POST", transit, appCookie, map[string]string{"status": "arrived"}); rec.Code != http.StatusConflict {
		t.Fatalf("arrived before en_route expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec = do("POST", transit, appCookie, map[string]string{"status": "en_route"}); rec.Code != http.StatusOK {
		t.Fatalf("en_route after confirmation expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec = do("POST", transit, appCookie, map[string]string{"status": "arrived"}); rec.Code != http.StatusOK {
		t.Fatalf("arrived after en_route expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// 14. Payment history endpoints
	rec = do("GET", "/api/employer/payments", empCookie, nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "search_fee") {
		t.Fatalf("employer history expected 200 with a search_fee entry, got %d: %s", rec.Code, rec.Body.String())
	}
	rec = do("GET", "/api/applicant/payments", appCookie, nil)
	bodyStr := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(bodyStr, "hiring_pass") || !strings.Contains(bodyStr, "hire_used") {
		t.Fatalf("applicant history expected hiring_pass and hire_used entries, got %d: %s", rec.Code, bodyStr)
	}
	if rec = do("GET", "/api/employer/payments", appCookie, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("applicant must not read employer history, got %d", rec.Code)
	}
}

func jsonNumber(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func mustRead(r io.Reader) string {
	b, _ := io.ReadAll(r)
	return string(b)
}
