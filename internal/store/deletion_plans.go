package store

// DeletionPlan and DeletionPlanItem persist the execution state of an imported
// TransRunner duplicate-deletion plan so a crash mid-quarantine can be
// reconciled instead of guessed at.
type DeletionPlan struct {
	PlanID           string `json:"plan_id"`
	Kind             string `json:"kind"`
	PlanVersion      int    `json:"plan_version"`
	SubscriptionSlug string `json:"subscription_slug"`
	ScanID           string `json:"scan_id"`
	Status           string `json:"status"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type DeletionPlanItem struct {
	PlanID         string `json:"plan_id"`
	VideoID        string `json:"video_id"`
	GroupID        string `json:"group_id"`
	KeptVideoID    string `json:"kept_video_id"`
	ExpectedSize   int64  `json:"expected_size"`
	ExpectedSHA256 string `json:"expected_sha256"`
	OriginalPath   string `json:"original_path"`
	QuarantinePath string `json:"quarantine_path"`
	Status         string `json:"status"`
	Reason         string `json:"reason"`
	UpdatedAt      string `json:"updated_at"`
}

func (s *Store) UpsertDeletionPlan(p DeletionPlan) error {
	_, err := s.DB.Exec("INSERT INTO deletion_plans(plan_id,kind,plan_version,subscription_slug,scan_id,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(plan_id) DO UPDATE SET kind=excluded.kind,plan_version=excluded.plan_version,subscription_slug=excluded.subscription_slug,scan_id=excluded.scan_id,status=excluded.status,updated_at=excluded.updated_at",
		p.PlanID, p.Kind, p.PlanVersion, p.SubscriptionSlug, p.ScanID, p.Status, Now(), Now())
	return err
}

func (s *Store) SaveDeletionPlanItem(it DeletionPlanItem) error {
	_, err := s.DB.Exec("INSERT INTO deletion_plan_items(plan_id,video_id,group_id,kept_video_id,expected_size,expected_sha256,original_path,quarantine_path,status,reason,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(plan_id,video_id) DO UPDATE SET group_id=excluded.group_id,kept_video_id=excluded.kept_video_id,expected_size=excluded.expected_size,expected_sha256=excluded.expected_sha256,original_path=excluded.original_path,quarantine_path=excluded.quarantine_path,status=excluded.status,reason=excluded.reason,updated_at=excluded.updated_at",
		it.PlanID, it.VideoID, it.GroupID, it.KeptVideoID, it.ExpectedSize, it.ExpectedSHA256, it.OriginalPath, it.QuarantinePath, it.Status, it.Reason, Now())
	return err
}

func (s *Store) DeletionPlan(planID string) (DeletionPlan, error) {
	var p DeletionPlan
	err := s.DB.QueryRow("SELECT plan_id,kind,plan_version,subscription_slug,scan_id,status,created_at,updated_at FROM deletion_plans WHERE plan_id=?", planID).
		Scan(&p.PlanID, &p.Kind, &p.PlanVersion, &p.SubscriptionSlug, &p.ScanID, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (s *Store) DeletionPlans(limit, offset int) ([]DeletionPlan, int64, error) {
	var total int64
	if err := s.DB.QueryRow("SELECT count(*) FROM deletion_plans").Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.DB.Query("SELECT plan_id,kind,plan_version,subscription_slug,scan_id,status,created_at,updated_at FROM deletion_plans ORDER BY created_at DESC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []DeletionPlan{}
	for rows.Next() {
		var p DeletionPlan
		if err = rows.Scan(&p.PlanID, &p.Kind, &p.PlanVersion, &p.SubscriptionSlug, &p.ScanID, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

func (s *Store) DeletionPlanItems(planID string) ([]DeletionPlanItem, error) {
	rows, err := s.DB.Query("SELECT plan_id,video_id,group_id,kept_video_id,expected_size,expected_sha256,original_path,quarantine_path,status,reason,updated_at FROM deletion_plan_items WHERE plan_id=? ORDER BY video_id", planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DeletionPlanItem{}
	for rows.Next() {
		var it DeletionPlanItem
		if err = rows.Scan(&it.PlanID, &it.VideoID, &it.GroupID, &it.KeptVideoID, &it.ExpectedSize, &it.ExpectedSHA256, &it.OriginalPath, &it.QuarantinePath, &it.Status, &it.Reason, &it.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// DeletionPlanItemsByStatus returns all plan items with the given status across
// every plan, used for idempotent "already done" evidence.
func (s *Store) DeletionPlanItemsByStatus(status string) ([]DeletionPlanItem, error) {
	rows, err := s.DB.Query("SELECT plan_id,video_id,group_id,kept_video_id,expected_size,expected_sha256,original_path,quarantine_path,status,reason,updated_at FROM deletion_plan_items WHERE status=? ORDER BY updated_at DESC", status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DeletionPlanItem{}
	for rows.Next() {
		var it DeletionPlanItem
		if err = rows.Scan(&it.PlanID, &it.VideoID, &it.GroupID, &it.KeptVideoID, &it.ExpectedSize, &it.ExpectedSHA256, &it.OriginalPath, &it.QuarantinePath, &it.Status, &it.Reason, &it.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Store) UpdateDeletionPlanItemStatus(planID, videoID, status, quarantinePath, reason string) error {
	_, err := s.DB.Exec("UPDATE deletion_plan_items SET status=?,quarantine_path=CASE WHEN ?<>'' THEN ? ELSE quarantine_path END,reason=?,updated_at=? WHERE plan_id=? AND video_id=?",
		status, quarantinePath, quarantinePath, reason, Now(), planID, videoID)
	return err
}

func (s *Store) SetDeletionPlanStatus(planID, status string) error {
	_, err := s.DB.Exec("UPDATE deletion_plans SET status=?,updated_at=? WHERE plan_id=?", status, Now(), planID)
	return err
}

// CountVideosWithPath counts other video rows that reference the same physical
// path (via file_path or plex_path), which blocks destructive cleanup.
func (s *Store) CountVideosWithPath(path, exceptID string) (int, error) {
	var n int
	err := s.DB.QueryRow("SELECT count(*) FROM videos WHERE id<>? AND (file_path=? OR plex_path=?)", exceptID, path, path).Scan(&n)
	return n, err
}
