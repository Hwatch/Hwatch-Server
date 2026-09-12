package store

import "database/sql"

type FingerprintMatch struct {
	VideoID, CoverHash string
	DurationSeconds    int64
}
type DuplicateCandidate struct {
	ID             int64   `json:"id"`
	VideoID        string  `json:"video_id"`
	VideoTitle     string  `json:"video_title"`
	VideoThumbnail string  `json:"video_thumbnail"`
	MatchVideoID   string  `json:"match_video_id"`
	MatchTitle     string  `json:"match_title"`
	MatchThumbnail string  `json:"match_thumbnail"`
	MatchType      string  `json:"match_type"`
	Confidence     float64 `json:"confidence"`
	DurationDelta  int64   `json:"duration_delta"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"created_at"`
	ReviewedAt     string  `json:"reviewed_at"`
}

func (s *Store) UpsertFingerprint(videoID, normalizedURL, coverHash string, duration int64) error {
	_, err := s.DB.Exec("INSERT INTO media_fingerprints(video_id,page_url_normalized,cover_hash,duration_seconds,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(video_id) DO UPDATE SET page_url_normalized=excluded.page_url_normalized,cover_hash=CASE WHEN excluded.cover_hash='' THEN media_fingerprints.cover_hash ELSE excluded.cover_hash END,duration_seconds=excluded.duration_seconds,updated_at=excluded.updated_at", videoID, normalizedURL, coverHash, duration, Now())
	return err
}
func (s *Store) IdentityDuplicate(videoID, normalizedURL string) (string, error) {
	var id string
	err := s.DB.QueryRow("SELECT video_id FROM media_fingerprints WHERE video_id<>? AND page_url_normalized=? AND page_url_normalized<>'' LIMIT 1", videoID, normalizedURL).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return id, err
}
func (s *Store) CoverCandidates(videoID string, duration int64) ([]FingerprintMatch, error) {
	rows, err := s.DB.Query("SELECT f.video_id,f.cover_hash,f.duration_seconds FROM media_fingerprints f JOIN videos v ON v.id=f.video_id WHERE f.video_id<>? AND v.status='completed' AND f.cover_hash<>'' AND f.duration_seconds BETWEEN ? AND ?", videoID, duration-5, duration+5)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FingerprintMatch{}
	for rows.Next() {
		var x FingerprintMatch
		if err = rows.Scan(&x.VideoID, &x.CoverHash, &x.DurationSeconds); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (s *Store) AddDuplicate(videoID, matchID, matchType string, confidence float64, delta int64, block bool) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec("INSERT INTO duplicate_candidates(video_id,match_video_id,match_type,confidence,duration_delta,status,created_at) VALUES(?,?,?,?,?,'pending',?) ON CONFLICT(video_id,match_video_id,match_type) DO UPDATE SET confidence=excluded.confidence,duration_delta=excluded.duration_delta", videoID, matchID, matchType, confidence, delta, Now())
	if err != nil {
		return err
	}
	if block {
		_, err = tx.Exec("UPDATE videos SET status='excluded',error=?,updated_at=? WHERE id=? AND status IN ('discovered','queued','failed','excluded')", "疑似重复，等待人工审查", Now(), videoID)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *Store) DuplicateDismissed(videoID, matchID, matchType string) bool {
	var n int
	_ = s.DB.QueryRow("SELECT count(*) FROM duplicate_candidates WHERE video_id=? AND match_video_id=? AND match_type=? AND status='dismissed'", videoID, matchID, matchType).Scan(&n)
	return n > 0
}
func (s *Store) SaveFileHashes(videoID string, size int64, quickHash, sha string) error {
	_, err := s.DB.Exec("INSERT INTO media_fingerprints(video_id,file_size,quick_hash,sha256,hash_status,updated_at) VALUES(?,?,?,?,'completed',?) ON CONFLICT(video_id) DO UPDATE SET file_size=excluded.file_size,quick_hash=excluded.quick_hash,sha256=excluded.sha256,hash_status='completed',updated_at=excluded.updated_at", videoID, size, quickHash, sha, Now())
	return err
}
func (s *Store) MissingFileHash() (Video, error) {
	return scanVideo(s.DB.QueryRow("SELECT " + videoColumns + " FROM videos v LEFT JOIN media_fingerprints f ON f.video_id=v.id WHERE v.status='completed' AND COALESCE(f.sha256,'')='' AND (v.plex_path<>'' OR v.file_path<>'') ORDER BY v.downloaded_at DESC LIMIT 1"))
}
func (s *Store) ExactHashDuplicate(videoID, sha string) (string, error) {
	var id string
	err := s.DB.QueryRow("SELECT video_id FROM media_fingerprints WHERE video_id<>? AND sha256=? AND sha256<>'' LIMIT 1", videoID, sha).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return id, err
}
func (s *Store) DuplicateCandidates() ([]DuplicateCandidate, error) {
	rows, err := s.DB.Query("SELECT d.id,d.video_id,v.title,v.thumbnail_url,d.match_video_id,m.title,m.thumbnail_url,d.match_type,d.confidence,d.duration_delta,d.status,d.created_at,d.reviewed_at FROM duplicate_candidates d JOIN videos v ON v.id=d.video_id JOIN videos m ON m.id=d.match_video_id ORDER BY CASE d.status WHEN 'pending' THEN 0 ELSE 1 END,d.created_at DESC LIMIT 500")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DuplicateCandidate{}
	for rows.Next() {
		var x DuplicateCandidate
		if err = rows.Scan(&x.ID, &x.VideoID, &x.VideoTitle, &x.VideoThumbnail, &x.MatchVideoID, &x.MatchTitle, &x.MatchThumbnail, &x.MatchType, &x.Confidence, &x.DurationDelta, &x.Status, &x.CreatedAt, &x.ReviewedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (s *Store) ReviewDuplicate(id int64, action string) (string, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var videoID string
	if err = tx.QueryRow("SELECT video_id FROM duplicate_candidates WHERE id=?", id).Scan(&videoID); err != nil {
		return "", err
	}
	status := "confirmed"
	if action == "dismiss" {
		status = "dismissed"
	}
	if _, err = tx.Exec("UPDATE duplicate_candidates SET status=?,reviewed_at=? WHERE id=?", status, Now(), id); err != nil {
		return "", err
	}
	if action == "dismiss" {
		_, err = tx.Exec("UPDATE videos SET status='queued',error='',updated_at=? WHERE id=? AND status='excluded'", Now(), videoID)
	}
	if err != nil {
		return "", err
	}
	return videoID, tx.Commit()
}
