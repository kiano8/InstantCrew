package auth

import (
	"context"
	"net/http"

	"instantcrew/internal/httpx"
	"instantcrew/internal/store"
)

type ctxKey struct{}

func UserFrom(ctx context.Context) (*store.User, bool) {
	u, ok := ctx.Value(ctxKey{}).(*store.User)
	return u, ok
}

func (h *Handler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(CookieName)
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "not logged in")
			return
		}
		u, err := h.Store.UserBySession(r.Context(), hashToken(c.Value))
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "not logged in")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, u)))
	})
}

// RequireRole is for later: protect employer-only or applicant-only endpoints.
func (h *Handler) RequireRole(role string, next http.Handler) http.Handler {
	return h.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, _ := UserFrom(r.Context()); u.Role != role {
			httpx.Error(w, http.StatusForbidden, "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	}))
}
