package jobs

import (
	"net/http"

	"instantcrew/internal/auth"
	"instantcrew/internal/httpx"
)

// GET /api/employer/payments: the employer's own search-fee history.
func (h *Handler) EmployerPaymentHistory(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	list, err := h.Store.EmployerPaymentHistory(r.Context(), u.ID)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"payments": list})
}

// GET /api/applicant/payments: free hires, top-ups and hires used.
func (h *Handler) ApplicantPaymentHistory(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	list, err := h.Store.ApplicantPaymentHistory(r.Context(), u.ID)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"payments": list})
}
