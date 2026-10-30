package auth

import (
	"context"
	"net/http"
	"strings"

	"instantcrew/internal/httpx"
	"instantcrew/internal/store"
)

type ctxKey struct{}

func UserFrom(ctx context.Context) (*store.User, bool) {
	u, ok := ctx.Value(ctxKey{}).(*store.User)
	return u, ok
}

func (h *Handler) resolveUser(r *http.Request, preferredRole string) *store.User {
	// 1. Use preferredRole's cookie, or infer the role from the query, header, or
	// referer. A role cookie only ever holds a session of that role (or admin),
	// and RequireRole checks the role afterwards.
	targetRole := preferredRole
	if targetRole == "" {
		if qRole := r.URL.Query().Get("role"); qRole != "" {
			targetRole = qRole
		} else if hRole := r.Header.Get("X-Auth-Role"); hRole != "" {
			targetRole = hRole
		} else if ref := r.Referer(); ref != "" {
			switch {
			case strings.Contains(ref, "admin.html") || strings.Contains(ref, "accounting.html"):
				targetRole = "admin"
			case strings.Contains(ref, "employer.html"):
				targetRole = "employer"
			case strings.Contains(ref, "applicant.html"):
				targetRole = "applicant"
			}
		}
	}

	if targetRole != "" {
		if c, err := r.Cookie(CookieName + "_" + targetRole); err == nil && c.Value != "" {
			if u, err := h.Store.UserBySession(r.Context(), hashToken(c.Value)); err == nil && u != nil {
				return u
			}
		}
	}

	// 2. Fallback to standard shared session cookie
	if c, err := r.Cookie(CookieName); err == nil && c.Value != "" {
		if u, err := h.Store.UserBySession(r.Context(), hashToken(c.Value)); err == nil && u != nil {
			return u
		}
	}

	// 3. Fallback: check any role-scoped cookies if standard cookie was missing
	for _, role := range []string{"employer", "admin", "applicant"} {
		if c, err := r.Cookie(CookieName + "_" + role); err == nil && c.Value != "" {
			if u, err := h.Store.UserBySession(r.Context(), hashToken(c.Value)); err == nil && u != nil {
				return u
			}
		}
	}

	return nil
}

func (h *Handler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := h.resolveUser(r, "")
		if u == nil {
			httpx.Error(w, http.StatusUnauthorized, "not logged in")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, u)))
	})
}

// RequireRole protects employer-only, applicant-only, or admin-only endpoints.
// Note: system administrators ('admin') are granted access across employer endpoints to allow auditing,
// verification, and smooth administration.
func (h *Handler) RequireRole(role string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := h.resolveUser(r, role)
		if u == nil {
			httpx.Error(w, http.StatusUnauthorized, "not logged in")
			return
		}
		if u.Role != role && !(role == "employer" && u.Role == "admin") {
			httpx.Error(w, http.StatusForbidden, "forbidden")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, u)))
	})
}
