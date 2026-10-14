package store

import "encoding/json"

// SubscriptionLogEvent is a single step in a subscription check, so the log
// center can show a full timeline for one task.
type SubscriptionLogEvent struct {
	At      string `json:"at"`
	Level   string `json:"level"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

// SubscriptionLog records the outcome of a subscription check.
type SubscriptionLog struct {
	ID               int64                  `json:"id"`
	SubscriptionSlug string                 `json:"subscription_slug"`
	Platform         string                 `json:"platform"`
	Level            string                 `json:"level"`
	Message          string                 `json:"message"`
	DurationMS       int64                  `json:"duration_ms"`
	FoundCount       int                    `json:"found"`
	CreatedCount     int                    `json:"created"`
	QueuedCount      int                    `json:"queued"`
	Events           []SubscriptionLogEvent `json:"events"`
	CreatedAt        string                 `json:"created_at"`
}

// subscriptionLogRetention caps the table so it never grows without bound.
const subscriptionLogRetention = 5000

func (s *Store) AddSubscriptionLog(slug, platform, level, message string, found, created, queued int, durationMS int64, events []SubscriptionLogEvent) error {
	if events == nil {
		events = []SubscriptionLogEvent{}
	}
	raw, _ := json.Marshal(events)
	if _, err := s.DB.Exec(
		"INSERT INTO subscription_logs(subscription_slug,platform,level,message,duration_ms,found_count,created_count,queued_count,events_json,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)",
		slug, platform, level, message, durationMS, found, created, queued, string(raw), Now(),
	); err != nil {
		return err
	}
	_, _ = s.DB.Exec("DELETE FROM subscription_logs WHERE id NOT IN (SELECT id FROM (SELECT id FROM subscription_logs ORDER BY id DESC LIMIT ?))", subscriptionLogRetention)
	return nil
}

func (s *Store) SubscriptionLogs(limit, offset int, slug, level string) ([]SubscriptionLog, error) {
	where, args := subscriptionLogWhere(slug, level)
	args = append(args, limit, offset)
	rows, err := s.DB.Query("SELECT id,subscription_slug,platform,level,message,duration_ms,found_count,created_count,queued_count,events_json,created_at FROM subscription_logs"+where+" ORDER BY id DESC LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SubscriptionLog{}
	for rows.Next() {
		var x SubscriptionLog
		var eventsJSON string
		if err := rows.Scan(&x.ID, &x.SubscriptionSlug, &x.Platform, &x.Level, &x.Message, &x.DurationMS, &x.FoundCount, &x.CreatedCount, &x.QueuedCount, &eventsJSON, &x.CreatedAt); err != nil {
			return nil, err
		}
		x.Events = []SubscriptionLogEvent{}
		if eventsJSON != "" {
			_ = json.Unmarshal([]byte(eventsJSON), &x.Events)
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *Store) CountSubscriptionLogs(slug, level string) (int64, error) {
	where, args := subscriptionLogWhere(slug, level)
	var n int64
	err := s.DB.QueryRow("SELECT count(*) FROM subscription_logs"+where, args...).Scan(&n)
	return n, err
}

func subscriptionLogWhere(slug, level string) (string, []any) {
	where := " WHERE 1=1"
	args := []any{}
	if slug != "" {
		where += " AND subscription_slug=?"
		args = append(args, slug)
	}
	if level != "" {
		where += " AND level=?"
		args = append(args, level)
	}
	return where, args
}
