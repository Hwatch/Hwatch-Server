package store

// SubscriptionLog records the outcome of a subscription check so operators can
// audit "what happened" for each subscription in the log center.
type SubscriptionLog struct {
	ID               int64  `json:"id"`
	SubscriptionSlug string `json:"subscription_slug"`
	Platform         string `json:"platform"`
	Level            string `json:"level"`
	Message          string `json:"message"`
	CreatedAt        string `json:"created_at"`
}

// subscriptionLogRetention caps the table so it never grows without bound.
const subscriptionLogRetention = 5000

func (s *Store) AddSubscriptionLog(slug, platform, level, message string) error {
	if _, err := s.DB.Exec("INSERT INTO subscription_logs(subscription_slug,platform,level,message,created_at) VALUES(?,?,?,?,?)", slug, platform, level, message, Now()); err != nil {
		return err
	}
	_, _ = s.DB.Exec("DELETE FROM subscription_logs WHERE id NOT IN (SELECT id FROM (SELECT id FROM subscription_logs ORDER BY id DESC LIMIT ?))", subscriptionLogRetention)
	return nil
}

func (s *Store) SubscriptionLogs(limit, offset int, slug, level string) ([]SubscriptionLog, error) {
	where, args := subscriptionLogWhere(slug, level)
	args = append(args, limit, offset)
	rows, err := s.DB.Query("SELECT id,subscription_slug,platform,level,message,created_at FROM subscription_logs"+where+" ORDER BY id DESC LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SubscriptionLog{}
	for rows.Next() {
		var x SubscriptionLog
		if err := rows.Scan(&x.ID, &x.SubscriptionSlug, &x.Platform, &x.Level, &x.Message, &x.CreatedAt); err != nil {
			return nil, err
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
