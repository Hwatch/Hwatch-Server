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
