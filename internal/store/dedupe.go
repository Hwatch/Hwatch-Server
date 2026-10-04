package store

import "database/sql"

type FingerprintMatch struct {
	VideoID, CoverHash string
	DurationSeconds    int64
}
type DuplicateCandidate struct {
	ID               int64   `json:"id"`
	VideoID          string  `json:"video_id"`
	VideoTitle       string  `json:"video_title"`
	VideoThumbnail   string  `json:"video_thumbnail"`
	MatchVideoID     string  `json:"match_video_id"`
	MatchTitle       string  `json:"match_title"`
	MatchThumbnail   string  `json:"match_thumbnail"`
	MatchType        string  `json:"match_type"`
	Classification   string  `json:"classification"`
	Backend          string  `json:"backend"`
	AlgorithmVersion string  `json:"algorithm_version"`
	MetricsJSON      string  `json:"metrics_json"`
	Confidence       float64 `json:"confidence"`
	DurationDelta    int64   `json:"duration_delta"`
	Status           string  `json:"status"`
	CreatedAt        string  `json:"created_at"`
	ReviewedAt       string  `json:"reviewed_at"`
	EstimatedBytes   int64   `json:"estimated_bytes"`
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
func (s *Store) ClearFileHashes(videoID string) error {
	_, err := s.DB.Exec("UPDATE media_fingerprints SET file_size=0,quick_hash='',sha256='',hash_status='',updated_at=? WHERE video_id=?", Now(), videoID)
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
	rows, err := s.DB.Query("SELECT d.id,d.video_id,v.title,v.thumbnail_url,d.match_video_id,m.title,m.thumbnail_url,d.match_type,d.classification,d.backend,d.algorithm_version,d.metrics_json,d.confidence,d.duration_delta,d.status,d.created_at,d.reviewed_at,CASE WHEN v.total_bytes>0 THEN v.total_bytes ELSE COALESCE(f.file_size,0) END FROM duplicate_candidates d JOIN videos v ON v.id=d.video_id JOIN videos m ON m.id=d.match_video_id LEFT JOIN media_fingerprints f ON f.video_id=d.video_id ORDER BY CASE d.status WHEN 'pending' THEN 0 ELSE 1 END,d.created_at DESC LIMIT 500")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DuplicateCandidate{}
	for rows.Next() {
		var x DuplicateCandidate
		if err = rows.Scan(&x.ID, &x.VideoID, &x.VideoTitle, &x.VideoThumbnail, &x.MatchVideoID, &x.MatchTitle, &x.MatchThumbnail, &x.MatchType, &x.Classification, &x.Backend, &x.AlgorithmVersion, &x.MetricsJSON, &x.Confidence, &x.DurationDelta, &x.Status, &x.CreatedAt, &x.ReviewedAt, &x.EstimatedBytes); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (s *Store) ReviewDuplicates(ids []int64, action string) (int, int64, error) {
	count := 0
	var bytes int64
	for _, id := range ids {
		var estimate int64
		_ = s.DB.QueryRow("SELECT CASE WHEN v.total_bytes>0 THEN v.total_bytes ELSE COALESCE(f.file_size,0) END FROM duplicate_candidates d JOIN videos v ON v.id=d.video_id LEFT JOIN media_fingerprints f ON f.video_id=v.id WHERE d.id=? AND d.status='pending'", id).Scan(&estimate)
		if _, err := s.ReviewDuplicate(id, action); err != nil {
			return count, bytes, err
		}
		count++
		bytes += estimate
	}
	return count, bytes, nil
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

// MediaDuplicateTargets returns completed media that already have a content
// hash (so exact duplicates are resolved first) and do not yet have a
// media-duplicate compute job for the given algorithm version.
func (s *Store) MediaDuplicateTargets(algorithmVersion string, limit int) ([]Video, error) {
	if limit < 1 {
		limit = 1
	}
	return scanVideos(s.DB.Query("SELECT "+prefixedVideoColumns("v")+" FROM videos v JOIN media_fingerprints f ON f.video_id=v.id WHERE v.status='completed' AND COALESCE(f.sha256,'')<>'' AND v.duration_seconds>0 AND (v.plex_path<>'' OR v.file_path<>'') AND NOT EXISTS (SELECT 1 FROM compute_jobs j WHERE j.video_id=v.id AND j.job_type='dedupe' AND j.subtype='media_duplicate' AND j.algorithm_version=?) ORDER BY v.downloaded_at DESC LIMIT ?", algorithmVersion, limit))
}

// MediaDuplicateHash returns the stored full SHA-256 for a video, if any.
func (s *Store) MediaDuplicateHash(videoID string) (string, error) {
	var sha string
	err := s.DB.QueryRow("SELECT COALESCE(sha256,'') FROM media_fingerprints WHERE video_id=?", videoID).Scan(&sha)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return sha, err
}

// MediaDuplicateCandidates returns completed media whose duration is within the
// tolerance of the target, excluding the target itself, exact SHA-256
// duplicates, media without a file path, and pairs already confirmed or
// dismissed. This is the server-side candidate generator: the node never scans
// the library on its own.
func (s *Store) MediaDuplicateCandidates(targetID, targetSHA string, duration, tolerance int64, limit int) ([]Video, error) {
	if limit < 1 {
		limit = 1
	}
	return scanVideos(s.DB.Query("SELECT "+prefixedVideoColumns("v")+" FROM videos v JOIN media_fingerprints f ON f.video_id=v.id WHERE v.status='completed' AND v.id<>? AND v.duration_seconds BETWEEN ? AND ? AND (v.plex_path<>'' OR v.file_path<>'') AND COALESCE(f.sha256,'')<>? AND NOT EXISTS (SELECT 1 FROM duplicate_candidates d WHERE d.match_type='media_duplicate' AND d.status IN ('confirmed','dismissed') AND ((d.video_id=? AND d.match_video_id=v.id) OR (d.video_id=v.id AND d.match_video_id=?))) ORDER BY ABS(v.duration_seconds-?) ASC, v.id ASC LIMIT ?", targetID, duration-tolerance, duration+tolerance, targetSHA, targetID, targetID, duration, limit))
}

// AddMediaDuplicate records a content-duplicate candidate using a canonical
// (sorted) pair ordering so A→B and B→A never produce two rows. It returns true
// when a new candidate row was created.
func (s *Store) AddMediaDuplicate(videoID, matchID, classification, backend, algorithmVersion, metricsJSON string, confidence float64, delta int64) (bool, error) {
	if videoID == matchID {
		return false, nil
	}
	a, b := videoID, matchID
	if a > b {
		a, b = b, a
	}
	var exists int
	if err := s.DB.QueryRow("SELECT count(*) FROM duplicate_candidates WHERE video_id=? AND match_video_id=? AND match_type='media_duplicate'", a, b).Scan(&exists); err != nil {
		return false, err
	}
	_, err := s.DB.Exec("INSERT INTO duplicate_candidates(video_id,match_video_id,match_type,classification,backend,algorithm_version,metrics_json,confidence,duration_delta,status,created_at) VALUES(?,?,?,?,?,?,?,?,?,'pending',?) ON CONFLICT(video_id,match_video_id,match_type) DO UPDATE SET classification=excluded.classification,backend=excluded.backend,algorithm_version=excluded.algorithm_version,metrics_json=excluded.metrics_json,confidence=excluded.confidence,duration_delta=excluded.duration_delta", a, b, "media_duplicate", classification, backend, algorithmVersion, metricsJSON, confidence, delta, Now())
	if err != nil {
		return false, err
	}
	return exists == 0, nil
}
