package store

// MediaExclusion is a user-created tombstone keyed by the canonical video_id. It
// permanently blocks automatic discovery of that video regardless of title, URL
// or filename changes.
type MediaExclusion struct {
	VideoID             string `json:"video_id"`
	Source              string `json:"source"`
	Reason              string `json:"reason"`
	TitleSnapshot       string `json:"title_snapshot"`
	OriginalURLSnapshot string `json:"original_url_snapshot"`
	ThumbnailSnapshot   string `json:"thumbnail_snapshot"`
	CreatedAt           string `json:"created_at"`
}

func (s *Store) AddMediaExclusion(e MediaExclusion) error {
	_, err := s.DB.Exec("INSERT INTO media_exclusions(video_id,source,reason,title_snapshot,original_url_snapshot,thumbnail_snapshot,created_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(video_id) DO UPDATE SET source=excluded.source,reason=excluded.reason,title_snapshot=excluded.title_snapshot,original_url_snapshot=excluded.original_url_snapshot,thumbnail_snapshot=excluded.thumbnail_snapshot",
		e.VideoID, e.Source, e.Reason, e.TitleSnapshot, e.OriginalURLSnapshot, e.ThumbnailSnapshot, Now())
	return err
}

func (s *Store) IsMediaExcluded(videoID string) bool {
	var count int
	if err := s.DB.QueryRow("SELECT count(*) FROM media_exclusions WHERE video_id=?", videoID).Scan(&count); err != nil {
		return false
	}
	return count > 0
}

func (s *Store) MediaExclusion(videoID string) (MediaExclusion, error) {
	var e MediaExclusion
	err := s.DB.QueryRow("SELECT video_id,source,reason,title_snapshot,original_url_snapshot,thumbnail_snapshot,created_at FROM media_exclusions WHERE video_id=?", videoID).
		Scan(&e.VideoID, &e.Source, &e.Reason, &e.TitleSnapshot, &e.OriginalURLSnapshot, &e.ThumbnailSnapshot, &e.CreatedAt)
	return e, err
}

func (s *Store) RemoveMediaExclusion(videoID string) error {
	_, err := s.DB.Exec("DELETE FROM media_exclusions WHERE video_id=?", videoID)
	return err
}

func (s *Store) MediaExclusions(limit, offset int, query, source string) ([]MediaExclusion, int64, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	where := "1=1"
	args := []any{}
	if query != "" {
		where += " AND (title_snapshot LIKE ? OR video_id LIKE ?)"
		args = append(args, "%"+query+"%", "%"+query+"%")
	}
	if source != "" {
		where += " AND source=?"
		args = append(args, source)
	}
	var total int64
	if err := s.DB.QueryRow("SELECT count(*) FROM media_exclusions WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	pageArgs := append(append([]any{}, args...), limit, offset)
	rows, err := s.DB.Query("SELECT video_id,source,reason,title_snapshot,original_url_snapshot,thumbnail_snapshot,created_at FROM media_exclusions WHERE "+where+" ORDER BY created_at DESC, video_id LIMIT ? OFFSET ?", pageArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []MediaExclusion{}
	for rows.Next() {
		var e MediaExclusion
		if err = rows.Scan(&e.VideoID, &e.Source, &e.Reason, &e.TitleSnapshot, &e.OriginalURLSnapshot, &e.ThumbnailSnapshot, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}

// MarkVideoRemoved clears the media paths of a video and sets a re-downloadable
// "removed" status without deleting the row (task/duplicate/audit history is
// preserved).
func (s *Store) MarkVideoRemoved(videoID string) error {
	_, err := s.DB.Exec("UPDATE videos SET status='removed',file_path='',plex_path='',aria_gid='',external_task_id='',downloader_id='',download_speed=0,connections=0,error='',downloaded_at='',updated_at=? WHERE id=?", Now(), videoID)
	if err != nil {
		return err
	}
	_, _ = s.DB.Exec("UPDATE media_fingerprints SET file_size=0,quick_hash='',sha256='',hash_status='',updated_at=? WHERE video_id=?", Now(), videoID)
	return nil
}
