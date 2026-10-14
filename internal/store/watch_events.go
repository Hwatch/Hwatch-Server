package store

import "time"

// WatchEvent is a single playback event used by the watch timeline.
type WatchEvent struct {
	ID        int64  `json:"id"`
	VideoID   string `json:"video_id"`
	WatchedAt string `json:"watched_at"`
}

const watchEventRetention = 5000

// RecordWatch appends a watch event, de-duplicating repeated events for the
// same video within a 10-minute window (a single playback issues many HTTP
// range requests, which must not spam the timeline).
func (s *Store) RecordWatch(videoID string) error {
	if videoID == "" {
		return nil
	}
	cutoff := time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339)
	var recent int
	_ = s.DB.QueryRow("SELECT count(*) FROM watch_events WHERE video_id=? AND watched_at>=?", videoID, cutoff).Scan(&recent)
	if recent > 0 {
		return nil
	}
	if _, err := s.DB.Exec("INSERT INTO watch_events(video_id,watched_at) VALUES(?,?)", videoID, Now()); err != nil {
		return err
	}
	_, _ = s.DB.Exec("DELETE FROM watch_events WHERE id NOT IN (SELECT id FROM (SELECT id FROM watch_events ORDER BY id DESC LIMIT ?))", watchEventRetention)
	return nil
}

func (s *Store) WatchEvents(limit, offset int) ([]WatchEvent, error) {
	rows, err := s.DB.Query("SELECT id,video_id,watched_at FROM watch_events ORDER BY id DESC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []WatchEvent{}
	for rows.Next() {
		var x WatchEvent
		if err := rows.Scan(&x.ID, &x.VideoID, &x.WatchedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *Store) CountWatchEvents() (int64, error) {
	var n int64
	err := s.DB.QueryRow("SELECT count(*) FROM watch_events").Scan(&n)
	return n, err
}
