package store

import "database/sql"

// IntegrationToken is a scoped bearer credential for an external service such as
// TransRunner. Only the SHA-256 hash of the token is stored; the plaintext is
// returned exactly once at creation time.
type IntegrationToken struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	ScopesJSON string `json:"scopes_json"`
	CreatedAt  string `json:"created_at"`
	LastUsedAt string `json:"last_used_at"`
	RevokedAt  string `json:"revoked_at"`
}

func (s *Store) CreateIntegrationToken(name, tokenHash, scopesJSON string) (int64, error) {
	if scopesJSON == "" {
		scopesJSON = "[]"
	}
	res, err := s.DB.Exec("INSERT INTO integration_tokens(name,token_hash,scopes_json,created_at) VALUES(?,?,?,?)", name, tokenHash, scopesJSON, Now())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ActiveIntegrationToken returns the newest non-revoked token for a name.
func (s *Store) ActiveIntegrationToken(name string) (IntegrationToken, error) {
	var t IntegrationToken
	err := s.DB.QueryRow("SELECT id,name,scopes_json,created_at,last_used_at,revoked_at FROM integration_tokens WHERE name=? AND revoked_at='' ORDER BY id DESC LIMIT 1", name).
		Scan(&t.ID, &t.Name, &t.ScopesJSON, &t.CreatedAt, &t.LastUsedAt, &t.RevokedAt)
	return t, err
}

// IntegrationTokenByHash returns an active (non-revoked) token matching the hash.
func (s *Store) IntegrationTokenByHash(tokenHash string) (IntegrationToken, error) {
	var t IntegrationToken
	err := s.DB.QueryRow("SELECT id,name,scopes_json,created_at,last_used_at,revoked_at FROM integration_tokens WHERE token_hash=? AND revoked_at=''", tokenHash).
		Scan(&t.ID, &t.Name, &t.ScopesJSON, &t.CreatedAt, &t.LastUsedAt, &t.RevokedAt)
	return t, err
}

func (s *Store) TouchIntegrationToken(id int64) error {
	_, err := s.DB.Exec("UPDATE integration_tokens SET last_used_at=? WHERE id=?", Now(), id)
	return err
}

func (s *Store) RevokeIntegrationToken(id int64) error {
	_, err := s.DB.Exec("UPDATE integration_tokens SET revoked_at=? WHERE id=? AND revoked_at=''", Now(), id)
	return err
}

// RevokeIntegrationTokens revokes every active token for a name and returns how
// many were revoked.
func (s *Store) RevokeIntegrationTokens(name string) (int64, error) {
	res, err := s.DB.Exec("UPDATE integration_tokens SET revoked_at=? WHERE name=? AND revoked_at=''", Now(), name)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// MediaFingerprint returns the stored file size and SHA-256 for a video.
func (s *Store) MediaFingerprint(videoID string) (size int64, sha256 string, err error) {
	err = s.DB.QueryRow("SELECT COALESCE(file_size,0),COALESCE(sha256,'') FROM media_fingerprints WHERE video_id=?", videoID).Scan(&size, &sha256)
	if err == sql.ErrNoRows {
		return 0, "", nil
	}
	return size, sha256, err
}

// SetVideoDuration updates the stored duration after a transcode apply.
func (s *Store) SetVideoDuration(videoID string, duration int64) error {
	_, err := s.DB.Exec("UPDATE videos SET duration_seconds=?,updated_at=? WHERE id=?", duration, Now(), videoID)
	return err
}

// TranscodeResult is a validated-but-uncommitted transcode output staged by an
// external service. HWatch does not manage runner jobs; this is only a result
// record.
type TranscodeResult struct {
	ID                 int64  `json:"id"`
	VideoID            string `json:"video_id"`
	RunnerJobID        string `json:"runner_job_id"`
	SourceSHA256       string `json:"source_sha256"`
	OutputRelativePath string `json:"output_relative_path"`
	Profile            string `json:"profile"`
	ProbeJSON          string `json:"probe_json"`
	Status             string `json:"status"`
	CreatedAt          string `json:"created_at"`
}

func (s *Store) AddTranscodeResult(r TranscodeResult) (int64, error) {
	if r.Status == "" {
		r.Status = "pending"
	}
	if r.ProbeJSON == "" {
		r.ProbeJSON = "{}"
	}
	res, err := s.DB.Exec("INSERT INTO transcode_results(video_id,runner_job_id,source_sha256,output_relative_path,profile,probe_json,status,created_at) VALUES(?,?,?,?,?,?,?,?)",
		r.VideoID, r.RunnerJobID, r.SourceSHA256, r.OutputRelativePath, r.Profile, r.ProbeJSON, r.Status, Now())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// TransRunnerSubmission records that a media operation was sent to TransRunner.
// It is intentionally not a task table: no progress, lease or runner state.
type TransRunnerSubmission struct {
	ID          int64  `json:"id"`
	VideoID     string `json:"video_id"`
	Operation   string `json:"operation"`
	Profile     string `json:"profile"`
	RunnerJobID string `json:"runner_job_id"`
	CreatedAt   string `json:"created_at"`
}

func (s *Store) AddTransRunnerSubmission(videoID, operation, profile, runnerJobID string) (int64, error) {
	res, err := s.DB.Exec("INSERT INTO transrunner_submissions(video_id,operation,profile,runner_job_id,created_at) VALUES(?,?,?,?,?)", videoID, operation, profile, runnerJobID, Now())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// LatestTransRunnerSubmission returns the most recent submission for a media
// operation (or no rows).
func (s *Store) LatestTransRunnerSubmission(videoID, operation, profile string) (TransRunnerSubmission, error) {
	var t TransRunnerSubmission
	err := s.DB.QueryRow("SELECT id,video_id,operation,profile,runner_job_id,created_at FROM transrunner_submissions WHERE video_id=? AND operation=? AND profile=? ORDER BY id DESC LIMIT 1", videoID, operation, profile).
		Scan(&t.ID, &t.VideoID, &t.Operation, &t.Profile, &t.RunnerJobID, &t.CreatedAt)
	return t, err
}

// TranscodeApplyOp is the durable idempotency/state ledger for a TransRunner
// transcode apply. The file replacement and the SQLite write cannot be one
// atomic operation, so this row is claimed before applying and updated during
// and after it, allowing a crash to be reconciled instead of guessed at.
type TranscodeApplyOp struct {
	OpKey              string `json:"op_key"`
	RunnerJobID        string `json:"runner_job_id"`
	VideoID            string `json:"video_id"`
	SourceSHA256       string `json:"source_sha256"`
	OutputSHA256       string `json:"output_sha256"`
	OutputRelativePath string `json:"output_relative_path"`
	Profile            string `json:"profile"`
	ContentHash        string `json:"content_hash"`
	Status             string `json:"status"`
	ResultID           int64  `json:"result_id"`
	SourceBackupPath   string `json:"source_backup_path"`
	Error              string `json:"error"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

const transcodeApplyColumns = "op_key,runner_job_id,video_id,source_sha256,output_sha256,output_relative_path,profile,content_hash,status,result_id,source_backup_path,error,created_at,updated_at"

func scanTranscodeApply(scanner interface{ Scan(...any) error }) (TranscodeApplyOp, error) {
	var op TranscodeApplyOp
	err := scanner.Scan(&op.OpKey, &op.RunnerJobID, &op.VideoID, &op.SourceSHA256, &op.OutputSHA256, &op.OutputRelativePath, &op.Profile, &op.ContentHash, &op.Status, &op.ResultID, &op.SourceBackupPath, &op.Error, &op.CreatedAt, &op.UpdatedAt)
	return op, err
}

// BeginTranscodeApply atomically claims an operation. It returns claimed=true
// only for the caller that inserted the row; every other caller receives the
// existing row (used for idempotent replay and conflict detection).
func (s *Store) BeginTranscodeApply(op TranscodeApplyOp) (TranscodeApplyOp, bool, error) {
	if op.Status == "" {
		op.Status = "applying"
	}
	res, err := s.DB.Exec("INSERT OR IGNORE INTO transcode_apply_ops("+transcodeApplyColumns+") VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		op.OpKey, op.RunnerJobID, op.VideoID, op.SourceSHA256, op.OutputSHA256, op.OutputRelativePath, op.Profile, op.ContentHash, op.Status, op.ResultID, op.SourceBackupPath, op.Error, Now(), Now())
	if err != nil {
		return TranscodeApplyOp{}, false, err
	}
	if n, _ := res.RowsAffected(); n == 1 {
		return op, true, nil
	}
	existing, err := s.TranscodeApplyByKey(op.OpKey)
	return existing, false, err
}

func (s *Store) TranscodeApplyByKey(opKey string) (TranscodeApplyOp, error) {
	return scanTranscodeApply(s.DB.QueryRow("SELECT "+transcodeApplyColumns+" FROM transcode_apply_ops WHERE op_key=?", opKey))
}

func (s *Store) LatestTranscodeApplyByJob(runnerJobID string) (TranscodeApplyOp, error) {
	return scanTranscodeApply(s.DB.QueryRow("SELECT "+transcodeApplyColumns+" FROM transcode_apply_ops WHERE runner_job_id=? ORDER BY updated_at DESC,rowid DESC LIMIT 1", runnerJobID))
}

// ReclaimTranscodeApply re-opens a failed/uncertain operation for one retry,
// but never re-opens an applied or in-flight one.
func (s *Store) ReclaimTranscodeApply(opKey string) error {
	_, err := s.DB.Exec("UPDATE transcode_apply_ops SET status='applying',error='',updated_at=? WHERE op_key=? AND status IN ('failed')", Now(), opKey)
	return err
}

func (s *Store) UpdateTranscodeApply(opKey, status string, resultID int64, errText, backupPath string) error {
	_, err := s.DB.Exec("UPDATE transcode_apply_ops SET status=?,result_id=CASE WHEN ?>0 THEN ? ELSE result_id END,error=?,source_backup_path=CASE WHEN ?<>'' THEN ? ELSE source_backup_path END,updated_at=? WHERE op_key=?",
		status, resultID, resultID, errText, backupPath, backupPath, Now(), opKey)
	return err
}

func (s *Store) LatestTranscodeResultByJob(runnerJobID string) (TranscodeResult, error) {
	var r TranscodeResult
	err := s.DB.QueryRow("SELECT id,video_id,runner_job_id,source_sha256,output_relative_path,profile,probe_json,status,created_at FROM transcode_results WHERE runner_job_id=? ORDER BY id DESC LIMIT 1", runnerJobID).
		Scan(&r.ID, &r.VideoID, &r.RunnerJobID, &r.SourceSHA256, &r.OutputRelativePath, &r.Profile, &r.ProbeJSON, &r.Status, &r.CreatedAt)
	return r, err
}
