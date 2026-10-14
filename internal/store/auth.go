package store

import (
	"database/sql"
	"errors"
)

var ErrAdminExists = errors.New("管理员账号已存在")

type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	Role         string `json:"role"`
	PasswordHash string `json:"-"`
	CreatedAt    string `json:"created_at"`
}

type Session struct {
	TokenHash, CSRFToken, ExpiresAt string
	UserID                          int64
	Username                        string
	Role                            string
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
	result, err := tx.Exec("INSERT INTO users(username,password_hash,role,created_at) VALUES(?,?,?,?)", username, passwordHash, "admin", Now())
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
	err := s.DB.QueryRow("SELECT id,username,password_hash,role,created_at FROM users WHERE username=?", username).
		Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt)
	return user, err
}

func (s *Store) CreateSession(tokenHash string, userID int64, csrfToken, expiresAt string) error {
	now := Now()
	_, err := s.DB.Exec("INSERT INTO sessions(token_hash,user_id,csrf_token,created_at,expires_at,last_seen_at) VALUES(?,?,?,?,?,?)", tokenHash, userID, csrfToken, now, expiresAt, now)
	return err
}

func (s *Store) Session(tokenHash string) (Session, error) {
	var session Session
	err := s.DB.QueryRow("SELECT s.token_hash,s.user_id,u.username,u.role,s.csrf_token,s.expires_at FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=? AND s.expires_at>?", tokenHash, Now()).
		Scan(&session.TokenHash, &session.UserID, &session.Username, &session.Role, &session.CSRFToken, &session.ExpiresAt)
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

// --- multi-user management -------------------------------------------------

func (s *Store) CreateUser(username, passwordHash, role string) (int64, error) {
	if role != "admin" && role != "viewer" {
		role = "viewer"
	}
	result, err := s.DB.Exec("INSERT INTO users(username,password_hash,role,created_at) VALUES(?,?,?,?)", username, passwordHash, role, Now())
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) Users() ([]User, error) {
	rows, err := s.DB.Query("SELECT id,username,role,created_at FROM users ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) UserByID(id int64) (User, error) {
	var u User
	err := s.DB.QueryRow("SELECT id,username,password_hash,role,created_at FROM users WHERE id=?", id).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	return u, err
}

func (s *Store) SetUserRole(id int64, role string) error {
	if role != "admin" && role != "viewer" {
		return errors.New("无效的角色")
	}
	_, err := s.DB.Exec("UPDATE users SET role=? WHERE id=?", role, id)
	return err
}

func (s *Store) UpdateUserPassword(id int64, passwordHash string) error {
	_, err := s.DB.Exec("UPDATE users SET password_hash=? WHERE id=?", passwordHash, id)
	return err
}

func (s *Store) DeleteUser(id int64) error {
	_, err := s.DB.Exec("DELETE FROM users WHERE id=?", id)
	return err
}

func (s *Store) CountAdmins() (int64, error) {
	var n int64
	err := s.DB.QueryRow("SELECT count(*) FROM users WHERE role='admin'").Scan(&n)
	return n, err
}
