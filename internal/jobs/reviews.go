package jobs

import (
	"net/http"

	"instantcrew/internal/auth"
	"instantcrew/internal/httpx"
)

// GET /api/reviews/pending lists finished contracts the user hasn't rated or skipped.
func (h *Handler) PendingReviews(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	list, err := h.Store.PendingReviews(r.Context(), u.ID)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"pending": list})
}

// POST /api/reviews body: {"job_id", "stars"}; stars 1-5 rates the other side of
// a finished contract, 0 skips it for good.
func (h *Handler) SubmitReview(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var req struct {
		JobID int64 `json:"job_id"`
		Stars int   `json:"stars"`
	}
	if !httpx.Decode(w, r, &req) {
		return
	}
	if req.JobID < 1 || req.Stars < 0 || req.Stars > 5 {
		httpx.Error(w, http.StatusBadRequest, "stars must be 1 to 5, or 0 to skip")
		return
	}
	if err := h.Store.SubmitReview(r.Context(), u.ID, req.JobID, req.Stars); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/employer/rating is the average of the stars crew gave this employer.
func (h *Handler) EmployerRating(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	m, err := h.Store.EmployerRatings(r.Context(), []int64{u.ID})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, m[u.ID])
}
