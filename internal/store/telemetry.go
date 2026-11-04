package store

import (
	"context"
	"database/sql"
	"time"
)

// manilaOffset aligns chart buckets and visitor days to Philippine calendar days.
const manilaOffset = 8 * 60 * 60

// ManilaDay returns the start of the Manila calendar day containing unix.
func ManilaDay(unix int64) int64 {
	return (unix+manilaOffset)/86400*86400 - manilaOffset
}

type TelemetryKey struct {
	Minute int64
	Route  string
	Status int
}

type TelemetryCount struct{ Requests, DurationMS, Slow int64 }

type TelemetryPoint struct {
	T        int64 `json:"t"`
	Requests int64 `json:"requests"`
	Client   int64 `json:"client_errors"`
	Server   int64 `json:"server_errors"`
	AvgMS    int64 `json:"avg_ms"`
	Slow     int64 `json:"slow"`
	Signups  int64 `json:"signups"`
	Jobs     int64 `json:"jobs"`
	Payments int64 `json:"payments"`
}

type TelemetryRoute struct {
	Route        string `json:"route"`
	Requests     int64  `json:"requests"`
	ClientErrors int64  `json:"client_errors"`
	ServerErrors int64  `json:"server_errors"`
	AvgMS        int64  `json:"avg_ms"`
	Slow         int64  `json:"slow"`
}

type TelemetryOutage struct {
	Start   int64 `json:"start"`
	End     int64 `json:"end"`
	Seconds int64 `json:"seconds"`
}

type TelemetryDay struct {
	Day      int64 `json:"t"`
	Visitors int64 `json:"visitors"`
}

// TelemetryUser is a real signup shown on the dashboard.
type TelemetryUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"` // contact number (+63…) or email, whatever they signed up with
	Role      string `json:"role"`
	Name      string `json:"name"`
	Company   string `json:"company"`
	CreatedAt int64  `json:"created_at"`
}

// realUsers leaves out admins and the demo accounts cmd/seed makes.
const realUsers = `role <> 'admin' AND email NOT LIKE '%@seed.instantcrew.test'`

type TelemetryReport struct {
	Series          []TelemetryPoint  `json:"series"`
	Routes          []TelemetryRoute  `json:"routes"`
	Outages         []TelemetryOutage `json:"outages"`
	UptimePercent   float64           `json:"uptime_percent"`
	MonitoringSince int64             `json:"monitoring_since"`
	Visitors        []TelemetryDay    `json:"visitors"`
	SignedIn        map[string]int64  `json:"signed_in"`
	Users           []TelemetryUser   `json:"users"` // every real signup, newest first
	DBOpenConns     int               `json:"db_open_conns"`
	DBInUseConns    int               `json:"db_in_use_conns"`
	DBPingMS        int64             `json:"db_ping_ms"`
}

// SaveTelemetry atomically adds the counts and records the heartbeat. A failed
// save therefore leaves a heartbeat gap, so losing the database counts as downtime.
func (s *Store) SaveTelemetry(ctx context.Context, minute int64, counts map[TelemetryKey]TelemetryCount, visitors map[int64]int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO telemetry_heartbeats (minute) VALUES ($1) ON CONFLICT DO NOTHING`, minute); err != nil {
		return err
	}
	for k, c := range counts {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO telemetry_requests (minute, route, status, requests, duration_ms, slow)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 ON CONFLICT (minute, route, status) DO UPDATE SET
			   requests = telemetry_requests.requests + excluded.requests,
			   duration_ms = telemetry_requests.duration_ms + excluded.duration_ms,
			   slow = telemetry_requests.slow + excluded.slow`,
			k.Minute, k.Route, k.Status, c.Requests, c.DurationMS, c.Slow); err != nil {
			return err
		}
	}
	for day, n := range visitors {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO telemetry_visitors (day, visitors) VALUES ($1, $2)
			 ON CONFLICT (day) DO UPDATE SET visitors = telemetry_visitors.visitors + excluded.visitors`,
			day, n); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) DeleteTelemetryBefore(ctx context.Context, cutoff int64) error {
	for _, q := range []string{
		`DELETE FROM telemetry_requests WHERE minute < $1`,
		`DELETE FROM telemetry_heartbeats WHERE minute < $1`,
		`DELETE FROM telemetry_visitors WHERE day < $1`,
	} {
		if _, err := s.db.ExecContext(ctx, q, cutoff); err != nil {
			return err
		}
	}
	return nil
}

// TelemetryReport covers `points` buckets of `step` seconds ending at the bucket
// containing now. Empty buckets are returned as zeros so charts keep a true time axis.
func (s *Store) TelemetryReport(ctx context.Context, now time.Time, step int64, points int) (*TelemetryReport, error) {
	bucket := func(t int64) int64 { return (t+manilaOffset)/step*step - manilaOffset }
	nowUnix := now.Unix()
	from := bucket(nowUnix) - int64(points-1)*step
	r := &TelemetryReport{
		Series:   make([]TelemetryPoint, points),
		Routes:   []TelemetryRoute{},
		Outages:  []TelemetryOutage{},
		Visitors: []TelemetryDay{},
		SignedIn: map[string]int64{},
		Users:    []TelemetryUser{},
	}
	for i := range r.Series {
		r.Series[i].T = from + int64(i)*step
	}
	at := func(t int64) *TelemetryPoint {
		i := (bucket(t) - from) / step
		if i < 0 || i >= int64(points) {
			return &TelemetryPoint{} // a row written just after now was captured
		}
		return &r.Series[i]
	}

	each := func(scan func(*sql.Rows) error, q string, args ...any) error {
		rows, err := s.db.QueryContext(ctx, q, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			if err := scan(rows); err != nil {
				return err
			}
		}
		return rows.Err()
	}

	err := each(func(rows *sql.Rows) error {
		var t, total, client, server, dur, slow int64
		if err := rows.Scan(&t, &total, &client, &server, &dur, &slow); err != nil {
			return err
		}
		p := at(t)
		p.Requests, p.Client, p.Server, p.Slow = total, client, server, slow
		if total > 0 {
			p.AvgMS = dur / total
		}
		return nil
	}, `SELECT (minute + $1::BIGINT) / $2::BIGINT * $2::BIGINT - $1::BIGINT AS b,
		       SUM(requests)::BIGINT,
		       COALESCE(SUM(requests) FILTER (WHERE status = 400), 0)::BIGINT,
		       COALESCE(SUM(requests) FILTER (WHERE status = 500), 0)::BIGINT,
		       SUM(duration_ms)::BIGINT, SUM(slow)::BIGINT
		FROM telemetry_requests WHERE minute >= $3 GROUP BY b`, manilaOffset, step, from)
	if err != nil {
		return nil, err
	}

	err = each(func(rows *sql.Rows) error {
		var kind string
		var t, n int64
		if err := rows.Scan(&kind, &t, &n); err != nil {
			return err
		}
		p := at(t)
		switch kind {
		case "signup":
			p.Signups = n
		case "job":
			p.Jobs = n
		case "payment":
			p.Payments = n
		}
		return nil
	}, `SELECT kind, (created_at + $1::BIGINT) / $2::BIGINT * $2::BIGINT - $1::BIGINT AS b, COUNT(*)::BIGINT
		FROM (
		  SELECT 'signup' AS kind, created_at FROM users WHERE `+realUsers+` AND created_at >= $3
		  UNION ALL SELECT 'job', created_at FROM jobs WHERE created_at >= $3
		  UNION ALL SELECT 'payment', created_at FROM payments WHERE created_at >= $3
		  UNION ALL SELECT 'payment', created_at FROM applicant_hiring_pass_payments WHERE created_at >= $3
		) activity GROUP BY kind, b`, manilaOffset, step, from)
	if err != nil {
		return nil, err
	}

	err = each(func(rows *sql.Rows) error {
		var x TelemetryRoute
		if err := rows.Scan(&x.Route, &x.Requests, &x.ClientErrors, &x.ServerErrors, &x.AvgMS, &x.Slow); err != nil {
			return err
		}
		r.Routes = append(r.Routes, x)
		return nil
	}, `SELECT route, SUM(requests)::BIGINT,
		       COALESCE(SUM(requests) FILTER (WHERE status = 400), 0)::BIGINT,
		       COALESCE(SUM(requests) FILTER (WHERE status = 500), 0)::BIGINT,
		       (SUM(duration_ms) / SUM(requests))::BIGINT, SUM(slow)::BIGINT
		FROM telemetry_requests WHERE minute >= $1
		GROUP BY route ORDER BY SUM(requests) DESC LIMIT 15`, from)
	if err != nil {
		return nil, err
	}

	var since sql.NullInt64
	if err := s.db.QueryRowContext(ctx, `SELECT MIN(minute) FROM telemetry_heartbeats`).Scan(&since); err != nil {
		return nil, err
	}
	r.MonitoringSince = since.Int64
	// Start from the last heartbeat before the window, so an outage that began
	// before the window is still seen. A gap needs a whole missing minute.
	var down int64
	err = each(func(rows *sql.Rows) error {
		var start, end int64
		if err := rows.Scan(&start, &end); err != nil {
			return err
		}
		start += 60
		if start < from {
			start = from
		}
		down += end - start
		r.Outages = append(r.Outages, TelemetryOutage{Start: start, End: end, Seconds: end - start})
		return nil
	}, `SELECT minute, next FROM (
		  SELECT minute, LEAD(minute) OVER (ORDER BY minute) AS next FROM telemetry_heartbeats
		  WHERE minute >= COALESCE((SELECT MAX(minute) FROM telemetry_heartbeats WHERE minute < $1), $1)
		) beats WHERE next - minute > 120 AND next > $1 ORDER BY minute DESC`, from)
	if err != nil {
		return nil, err
	}
	observedFrom := max(from, since.Int64)
	r.UptimePercent = 100
	if span := nowUnix - observedFrom; span > 0 {
		r.UptimePercent = max(0, 100*(1-float64(down)/float64(span)))
	}

	// Always the last 30 days: daily uniques cannot be summed into a range total.
	err = each(func(rows *sql.Rows) error {
		var d TelemetryDay
		if err := rows.Scan(&d.Day, &d.Visitors); err != nil {
			return err
		}
		r.Visitors = append(r.Visitors, d)
		return nil
	}, `SELECT day, visitors FROM telemetry_visitors WHERE day >= $1 ORDER BY day`, ManilaDay(nowUnix)-29*86400)
	if err != nil {
		return nil, err
	}

	err = each(func(rows *sql.Rows) error {
		var role string
		var n int64
		if err := rows.Scan(&role, &n); err != nil {
			return err
		}
		r.SignedIn[role] = n
		return nil
	}, `SELECT u.role, COUNT(DISTINCT s.user_id)::BIGINT
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.expires_at > $1 AND s.last_seen_at > $2 GROUP BY u.role`, nowUnix, s.idleCutoff(now))
	if err != nil {
		return nil, err
	}

	// shortcut: no paging, fine while the beta caps signups (BETA_MAX_USERS); page it past a few thousand.
	err = each(func(rows *sql.Rows) error {
		var u TelemetryUser
		if err := rows.Scan(&u.ID, &u.Login, &u.Role, &u.Name, &u.Company, &u.CreatedAt); err != nil {
			return err
		}
		r.Users = append(r.Users, u)
		return nil
	}, `SELECT id, email, role, name, company, created_at FROM users WHERE `+realUsers+` ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}

	stats := s.db.Stats()
	r.DBOpenConns, r.DBInUseConns = stats.OpenConnections, stats.InUse
	pingStart := time.Now()
	if err := s.db.PingContext(ctx); err != nil {
		return nil, err
	}
	r.DBPingMS = time.Since(pingStart).Milliseconds()
	return r, nil
}
