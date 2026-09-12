package store

import (
	"database/sql"
	"errors"
)

var ErrAdminExists = errors.New("管理员账号已存在")

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	CreatedAt    string
}

type Session struct {
	TokenHash, CSRFToken, ExpiresAt string
	UserID                          int64
	Username                        string
}

func (s *Store) HasUsers() (bool, error) {
	var count int
	err := s.DB.QueryRow("SELECT count(*) FROM users").Scan(&count)
	return count > 0, err
}

func (s *Store) CreateFirstAdmin(username, passwordHash string) (int64, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var count int
	if err = tx.QueryRow("SELECT count(*) FROM users").Scan(&count); err != nil {
		return 0, err
	}
	if count != 0 {
		return 0, ErrAdminExists
	}
	result, err := tx.Exec("INSERT INTO users(username,password_hash,created_at) VALUES(?,?,?)", username, passwordHash, Now())
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (s *Store) UserByName(username string) (User, error) {
	var user User
	err := s.DB.QueryRow("SELECT id,username,password_hash,created_at FROM users WHERE username=?", username).
		Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	return user, err
}

func (s *Store) CreateSession(tokenHash string, userID int64, csrfToken, expiresAt string) error {
	now := Now()
	_, err := s.DB.Exec("INSERT INTO sessions(token_hash,user_id,csrf_token,created_at,expires_at,last_seen_at) VALUES(?,?,?,?,?,?)", tokenHash, userID, csrfToken, now, expiresAt, now)
	return err
}

func (s *Store) Session(tokenHash string) (Session, error) {
	var session Session
	err := s.DB.QueryRow("SELECT s.token_hash,s.user_id,u.username,s.csrf_token,s.expires_at FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=? AND s.expires_at>?", tokenHash, Now()).
		Scan(&session.TokenHash, &session.UserID, &session.Username, &session.CSRFToken, &session.ExpiresAt)
	if err == nil {
		_, _ = s.DB.Exec("UPDATE sessions SET last_seen_at=? WHERE token_hash=?", Now(), tokenHash)
	}
	return session, err
}

func (s *Store) DeleteSession(tokenHash string) error {
	_, err := s.DB.Exec("DELETE FROM sessions WHERE token_hash=?", tokenHash)
	return err
}

func (s *Store) CleanupSessions() error {
	_, err := s.DB.Exec("DELETE FROM sessions WHERE expires_at<=?", Now())
	return err
}

func IsNotFound(err error) bool { return errors.Is(err, sql.ErrNoRows) }
