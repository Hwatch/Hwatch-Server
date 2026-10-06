package store

import "encoding/json"

type TaskEvent struct {
	ID         int64  `json:"id"`
	VideoID    string `json:"video_id"`
	FromStatus string `json:"from_status"`
	ToStatus   string `json:"to_status"`
	Message    string `json:"message"`
	CreatedAt  string `json:"created_at"`
}

func (s *Store) AddAudit(actor, action, targetType, targetID, result string, details map[string]any) error {
	raw, _ := json.Marshal(details)
	_, err := s.DB.Exec("INSERT INTO audit_logs(actor,action,target_type,target_id,result,details_json,created_at) VALUES(?,?,?,?,?,?,?)", actor, action, targetType, targetID, result, string(raw), Now())
	return err
}

func (s *Store) AuditLogs(limit, offset int) ([]AuditLog, error) {
	rows, err := s.DB.Query("SELECT id,actor,action,target_type,target_id,result,details_json,created_at FROM audit_logs ORDER BY id DESC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AuditLog{}
	for rows.Next() {
		var x AuditLog
		if err = rows.Scan(&x.ID, &x.Actor, &x.Action, &x.TargetType, &x.TargetID, &x.Result, &x.Details, &x.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *Store) AddTaskEvent(videoID, fromStatus, toStatus, message string) error {
	if fromStatus == toStatus {
		return nil
	}
	_, err := s.DB.Exec("INSERT INTO task_events(video_id,from_status,to_status,message,created_at) VALUES(?,?,?,?,?)", videoID, fromStatus, toStatus, message, Now())
	return err
}

func (s *Store) TaskEvents(videoID string, limit int) ([]TaskEvent, error) {
	rows, err := s.DB.Query("SELECT id,video_id,from_status,to_status,message,created_at FROM task_events WHERE video_id=? ORDER BY id DESC LIMIT ?", videoID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TaskEvent{}
	for rows.Next() {
		var x TaskEvent
		if err = rows.Scan(&x.ID, &x.VideoID, &x.FromStatus, &x.ToStatus, &x.Message, &x.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
