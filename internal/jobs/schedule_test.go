package jobs

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

// A scheduled shift can't be for today (that is an instant search); tomorrow is fine.
func TestParseRejectsSameDaySchedule(t *testing.T) {
	h := &Handler{}
	manila := time.FixedZone("Asia/Manila", 8*3600)
	now := time.Now().In(manila)
	endOfToday := time.Date(now.Year(), now.Month(), now.Day(), 23, 30, 0, 0, manila)
	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 9, 0, 0, 0, manila)
	for when, ok := range map[time.Time]bool{endOfToday: false, tomorrow: true} {
		body, _ := json.Marshal(map[string]any{
			"category": "kitchen", "role": "cook", "employment_type": "full-time", "rate": 150, "rate_period": "hour",
			"city": "Cebu City", "map_location": "IT Park", "timing": "later", "scheduled_at": when.Format(time.RFC3339), "crew_needed": 1,
		})
		r := httptest.NewRequest("POST", "/api/jobs", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		if _, got := h.parse(w, r); got != ok {
			t.Errorf("scheduled %s: accepted=%v, want %v (%s)", when.Format(time.RFC3339), got, ok, w.Body.String())
		}
	}
}
