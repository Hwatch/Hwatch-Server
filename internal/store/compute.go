package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type ComputeNode struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	PermissionsJSON  string `json:"permissions_json"`
	CapabilitiesJSON string `json:"capabilities_json"`
	Version          string `json:"version"`
	Platform         string `json:"platform"`
	Status           string `json:"status"`
	LastSeenAt       string `json:"last_seen_at"`
	CreatedAt        string `json:"created_at"`
	RevokedAt        string `json:"revoked_at"`
}

type ComputeJob struct {
	ID                string  `json:"id"`
	Type              string  `json:"type"`
	VideoID           string  `json:"video_id"`
	Status            string  `json:"status"`
	NodeID            string  `json:"node_id"`
	IdempotencyKey    string  `json:"idempotency_key"`
	AlgorithmVersion  string  `json:"algorithm_version"`
	MediaRelativePath string  `json:"media_relative_path"`
	InputJSON         string  `json:"input_json"`
	ResultJSON        string  `json:"result_json"`
	ErrorCode         string  `json:"error_code"`
	ErrorMessage      string  `json:"error_message"`
	LeaseUntil        string  `json:"lease_until"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
	CompletedAt       string  `json:"completed_at"`
	Progress          float64 `json:"progress"`
	Attempts          int     `json:"attempts"`
}

var ErrNoComputeJob = errors.New("没有可领取的计算任务")
var ErrComputeLease = errors.New("任务租约无效或已被其他节点接管")

const computeJobColumns = "id,job_type,video_id,status,node_id,idempotency_key,algorithm_version,media_relative_path,input_json,result_json,progress,error_code,error_message,lease_until,attempts,created_at,updated_at,completed_at"

func scanComputeJob(scanner interface{ Scan(...any) error }) (ComputeJob, error) {
	var j ComputeJob
	err := scanner.Scan(&j.ID, &j.Type, &j.VideoID, &j.Status, &j.NodeID, &j.IdempotencyKey, &j.AlgorithmVersion, &j.MediaRelativePath, &j.InputJSON, &j.ResultJSON, &j.Progress, &j.ErrorCode, &j.ErrorMessage, &j.LeaseUntil, &j.Attempts, &j.CreatedAt, &j.UpdatedAt, &j.CompletedAt)
	return j, err
}

func (s *Store) CreateComputeNode(id, name, tokenHash, permissions string) error {
	_, err := s.DB.Exec("INSERT INTO compute_nodes(id,name,token_hash,permissions_json,capabilities_json,status,created_at) VALUES(?,?,?,?,?,'offline',?)", id, name, tokenHash, permissions, permissions, Now())
	return err
}

func (s *Store) ComputeNodes() ([]ComputeNode, error) {
	rows, err := s.DB.Query("SELECT id,name,permissions_json,capabilities_json,version,platform,status,last_seen_at,created_at,revoked_at FROM compute_nodes ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ComputeNode{}
	for rows.Next() {
		var n ComputeNode
		if err = rows.Scan(&n.ID, &n.Name, &n.PermissionsJSON, &n.CapabilitiesJSON, &n.Version, &n.Platform, &n.Status, &n.LastSeenAt, &n.CreatedAt, &n.RevokedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Store) ComputeNodeByToken(hash string) (ComputeNode, error) {
	var n ComputeNode
	err := s.DB.QueryRow("SELECT id,name,permissions_json,capabilities_json,version,platform,status,last_seen_at,created_at,revoked_at FROM compute_nodes WHERE token_hash=? AND revoked_at=''", hash).Scan(&n.ID, &n.Name, &n.PermissionsJSON, &n.CapabilitiesJSON, &n.Version, &n.Platform, &n.Status, &n.LastSeenAt, &n.CreatedAt, &n.RevokedAt)
	return n, err
}

func (s *Store) HeartbeatComputeNode(id, capabilities, version, platform string) error {
	res, err := s.DB.Exec("UPDATE compute_nodes SET capabilities_json=?,version=?,platform=?,status='online',last_seen_at=? WHERE id=? AND revoked_at=''", capabilities, version, platform, Now(), id)
	if err == nil {
		if n, _ := res.RowsAffected(); n == 0 {
			return sql.ErrNoRows
		}
	}
	return err
}

func (s *Store) RevokeComputeNode(id string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := Now()
	if _, err = tx.Exec("UPDATE compute_nodes SET status='revoked',revoked_at=? WHERE id=? AND revoked_at=''", now, id); err != nil {
		return err
	}
	if _, err = tx.Exec("UPDATE compute_jobs SET status='pending',node_id='',lease_until='',error_code='node_revoked',error_message='节点已撤销，任务等待重新分派',updated_at=? WHERE node_id=? AND status IN ('assigned','analyzing','matching','transcoding')", now, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) CreateComputeJob(j ComputeJob) (ComputeJob, bool, error) {
	now := Now()
	res, err := s.DB.Exec("INSERT OR IGNORE INTO compute_jobs(id,job_type,video_id,status,idempotency_key,algorithm_version,media_relative_path,input_json,created_at,updated_at) VALUES(?,?,?,'pending',?,?,?,?,?,?)", j.ID, j.Type, j.VideoID, j.IdempotencyKey, j.AlgorithmVersion, j.MediaRelativePath, j.InputJSON, now, now)
	if err != nil {
		return ComputeJob{}, false, err
	}
	n, _ := res.RowsAffected()
	existing, err := s.ComputeJobByIdempotency(j.IdempotencyKey)
	return existing, n > 0, err
}

func (s *Store) ComputeJobByIdempotency(key string) (ComputeJob, error) {
	return scanComputeJob(s.DB.QueryRow("SELECT "+computeJobColumns+" FROM compute_jobs WHERE idempotency_key=?", key))
}
func (s *Store) ComputeJob(id string) (ComputeJob, error) {
	return scanComputeJob(s.DB.QueryRow("SELECT "+computeJobColumns+" FROM compute_jobs WHERE id=?", id))
}

func (s *Store) ComputeJobs(limit, offset int) ([]ComputeJob, error) {
	rows, err := s.DB.Query("SELECT "+computeJobColumns+" FROM compute_jobs ORDER BY CASE status WHEN 'assigned' THEN 0 WHEN 'analyzing' THEN 1 WHEN 'matching' THEN 2 WHEN 'transcoding' THEN 3 WHEN 'verifying' THEN 4 WHEN 'pending' THEN 5 ELSE 6 END,created_at DESC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ComputeJob{}
	for rows.Next() {
		j, e := scanComputeJob(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *Store) ClaimComputeJob(nodeID string, types []string, lease time.Duration) (ComputeJob, error) {
	if len(types) == 0 {
		return ComputeJob{}, ErrNoComputeJob
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return ComputeJob{}, err
	}
	defer tx.Rollback()
	now := Now()
	if _, err = tx.Exec("UPDATE compute_jobs SET status='pending',node_id='',lease_until='',error_code='lease_expired',error_message='节点租约过期，任务等待重新分派',updated_at=? WHERE status IN ('assigned','analyzing','matching','transcoding') AND lease_until<>'' AND lease_until<?", now, now); err != nil {
		return ComputeJob{}, err
	}
	marks := strings.TrimRight(strings.Repeat("?,", len(types)), ",")
	args := make([]any, 0, len(types))
	for _, v := range types {
		args = append(args, v)
	}
	query := "SELECT " + computeJobColumns + " FROM compute_jobs WHERE status='pending' AND job_type IN (" + marks + ") ORDER BY created_at LIMIT 1"
	j, err := scanComputeJob(tx.QueryRow(query, args...))
	if err == sql.ErrNoRows {
		return ComputeJob{}, ErrNoComputeJob
	}
	if err != nil {
		return ComputeJob{}, err
	}
	until := time.Now().UTC().Add(lease).Format(time.RFC3339)
	res, err := tx.Exec("UPDATE compute_jobs SET status='assigned',node_id=?,lease_until=?,attempts=attempts+1,error_code='',error_message='',updated_at=? WHERE id=? AND status='pending'", nodeID, until, now, j.ID)
	if err != nil {
		return ComputeJob{}, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ComputeJob{}, ErrNoComputeJob
	}
	if err = tx.Commit(); err != nil {
		return ComputeJob{}, err
	}
	return s.ComputeJob(j.ID)
}

func (s *Store) RenewComputeLease(jobID, nodeID, status string, progress float64, lease time.Duration) error {
	allowed := map[string]bool{"assigned": true, "analyzing": true, "matching": true, "transcoding": true}
	if !allowed[status] {
		return fmt.Errorf("无效的运行状态 %q", status)
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	now := Now()
	until := time.Now().UTC().Add(lease).Format(time.RFC3339)
	res, err := s.DB.Exec("UPDATE compute_jobs SET status=?,progress=?,lease_until=?,updated_at=? WHERE id=? AND node_id=? AND status IN ('assigned','analyzing','matching','transcoding')", status, progress, until, now, jobID, nodeID)
	if err == nil {
		if n, _ := res.RowsAffected(); n != 1 {
			return ErrComputeLease
		}
	}
	return err
}

func (s *Store) CompleteComputeJob(jobID, nodeID, status, result string) error {
	if status != "completed" && status != "verifying" {
		return errors.New("无效的完成状态")
	}
	now := Now()
	res, err := s.DB.Exec("UPDATE compute_jobs SET status=?,result_json=?,progress=1,lease_until='',completed_at=CASE WHEN ?='completed' THEN ? ELSE '' END,updated_at=? WHERE id=? AND node_id=? AND status IN ('assigned','analyzing','matching','transcoding')", status, result, status, now, now, jobID, nodeID)
	if err == nil {
		if n, _ := res.RowsAffected(); n != 1 {
			return ErrComputeLease
		}
	}
	return err
}

func (s *Store) FailComputeJob(jobID, nodeID, code, message string, retry bool) error {
	status := "failed"
	assigned := nodeID
	if retry {
		status = "pending"
		assigned = ""
	}
	now := Now()
	res, err := s.DB.Exec("UPDATE compute_jobs SET status=?,node_id=?,lease_until='',error_code=?,error_message=?,updated_at=? WHERE id=? AND node_id=? AND status IN ('assigned','analyzing','matching','transcoding')", status, assigned, code, message, now, jobID, nodeID)
	if err == nil {
		if n, _ := res.RowsAffected(); n != 1 {
			return ErrComputeLease
		}
	}
	return err
}

func (s *Store) CancelComputeJob(id string) error {
	_, err := s.DB.Exec("UPDATE compute_jobs SET status='cancelled',lease_until='',updated_at=? WHERE id=? AND status NOT IN ('completed','cancelled')", Now(), id)
	return err
}
func (s *Store) CommitComputeJob(id string) error {
	now := Now()
	res, err := s.DB.Exec("UPDATE compute_jobs SET status='completed',completed_at=?,updated_at=? WHERE id=? AND status='verifying'", now, now, id)
	if err == nil {
		if n, _ := res.RowsAffected(); n != 1 {
			return errors.New("任务不在待验证状态")
		}
	}
	return err
}
func (s *Store) SetTranscodeStatus(videoID, status string) error {
	_, err := s.DB.Exec("UPDATE videos SET transcode_status=?,updated_at=? WHERE id=?", status, Now(), videoID)
	return err
}
