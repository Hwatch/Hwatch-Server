package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
	"r34watch/internal/store"
)

const sessionCookie = "hwatch_session"

type authContextKey struct{}

type authInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func randomToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (a *App) currentSession(r *http.Request) (store.Session, error) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return store.Session{}, errors.New("未登录")
	}
	return a.store.Session(tokenHash(cookie.Value))
}

func (a *App) authStatus(w http.ResponseWriter, r *http.Request) {
	configured, err := a.store.HasUsers()
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	response := map[string]any{"configured": configured, "authenticated": false}
	if session, err := a.currentSession(r); err == nil {
		response["authenticated"] = true
		response["username"] = session.Username
		response["role"] = session.Role
		response["csrf_token"] = session.CSRFToken
	}
	respond(w, http.StatusOK, response)
}

func validateCredentials(input authInput) error {
	input.Username = strings.TrimSpace(input.Username)
	if utf8.RuneCountInString(input.Username) < 3 || utf8.RuneCountInString(input.Username) > 64 {
		return errors.New("用户名长度需为 3 到 64 个字符")
	}
	if len(input.Password) < 10 || len(input.Password) > 256 {
		return errors.New("密码长度需为 10 到 256 个字符")
	}
	return nil
}

func (a *App) authSetup(w http.ResponseWriter, r *http.Request) {
	configured, err := a.store.HasUsers()
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	if configured {
		fail(w, http.StatusConflict, store.ErrAdminExists)
		return
	}
	var input authInput
	if err := decode(r, &input); err != nil || validateCredentials(input) != nil {
		if err == nil {
			err = validateCredentials(input)
		}
		fail(w, http.StatusBadRequest, err)
		return
	}
	input.Username = strings.TrimSpace(input.Username)
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	userID, err := a.store.CreateFirstAdmin(input.Username, string(hash))
	if err != nil {
		if errors.Is(err, store.ErrAdminExists) {
			fail(w, http.StatusConflict, err)
		} else {
			fail(w, http.StatusInternalServerError, err)
		}
		return
	}
	a.issueSession(w, r, userID, input.Username)
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func (a *App) loginAllowed(ip string) bool {
	a.authMu.Lock()
	defer a.authMu.Unlock()
	cutoff := time.Now().Add(-15 * time.Minute)
	kept := a.loginAttempts[ip][:0]
	for _, attempt := range a.loginAttempts[ip] {
		if attempt.After(cutoff) {
			kept = append(kept, attempt)
		}
	}
	a.loginAttempts[ip] = kept
	return len(kept) < 5
}

func (a *App) recordLoginFailure(ip string) {
	a.authMu.Lock()
	a.loginAttempts[ip] = append(a.loginAttempts[ip], time.Now())
	a.authMu.Unlock()
}

func (a *App) authLogin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !a.loginAllowed(ip) {
		w.Header().Set("Retry-After", "900")
		fail(w, http.StatusTooManyRequests, errors.New("登录尝试过多，请稍后重试"))
		return
	}
	var input authInput
	if decode(r, &input) != nil {
		fail(w, http.StatusBadRequest, errors.New("请求格式错误"))
		return
	}
	user, err := a.store.UserByName(strings.TrimSpace(input.Username))
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		a.recordLoginFailure(ip)
		fail(w, http.StatusUnauthorized, errors.New("用户名或密码错误"))
		return
	}
	a.authMu.Lock()
	delete(a.loginAttempts, ip)
	a.authMu.Unlock()
	a.issueSession(w, r, user.ID, user.Username)
}

func (a *App) issueSession(w http.ResponseWriter, r *http.Request, userID int64, username string) {
	token, err := randomToken()
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	csrf, err := randomToken()
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	expires := time.Now().UTC().Add(7 * 24 * time.Hour)
	if err = a.store.CreateSession(tokenHash(token), userID, csrf, expires.Format(time.RFC3339)); err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode, Expires: expires, MaxAge: 7 * 24 * 60 * 60})
	respond(w, http.StatusOK, map[string]any{"authenticated": true, "username": username, "csrf_token": csrf})
}

func (a *App) authLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		_ = a.store.DeleteSession(tokenHash(cookie.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
	respond(w, http.StatusOK, map[string]bool{"authenticated": false})
}

func (a *App) authMe(w http.ResponseWriter, r *http.Request) {
	session, _ := r.Context().Value(authContextKey{}).(store.Session)
	respond(w, http.StatusOK, map[string]any{"authenticated": true, "username": session.Username, "role": session.Role, "csrf_token": session.CSRFToken})
}

func (a *App) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.maintenance.Load() && (strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/media/") || strings.HasPrefix(r.URL.Path, "/thumb/")) {
			w.Header().Set("Retry-After", "5")
			fail(w, http.StatusServiceUnavailable, errors.New("服务正在执行数据库维护，请稍后重试"))
			return
		}
		if r.URL.Path == "/api/auth/status" || r.URL.Path == "/api/auth/setup" || r.URL.Path == "/api/auth/login" || strings.HasPrefix(r.URL.Path, "/api/transrunner/v1/") || !strings.HasPrefix(r.URL.Path, "/api/") && !strings.HasPrefix(r.URL.Path, "/media/") && !strings.HasPrefix(r.URL.Path, "/thumb/") {
			next.ServeHTTP(w, r)
			return
		}
		session, err := a.currentSession(r)
		if err != nil {
			if token, ok := a.integrationReadableRequest(r); ok {
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), integrationTokenKey{}, token)))
				return
			}
			fail(w, http.StatusUnauthorized, errors.New("请先登录"))
			return
		}
		if session.Role != "admin" && !viewerAllowed(r.Method, r.URL.Path) {
			fail(w, http.StatusForbidden, errors.New("仅观看账号无权执行该操作"))
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			provided := r.Header.Get("X-CSRF-Token")
			if len(provided) != len(session.CSRFToken) || subtle.ConstantTimeCompare([]byte(provided), []byte(session.CSRFToken)) != 1 {
				fail(w, http.StatusForbidden, errors.New("CSRF 校验失败，请刷新页面后重试"))
				return
			}
		}
		request := r.WithContext(context.WithValue(r.Context(), authContextKey{}, session))
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, request)
			return
		}
		recorder := &auditResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, request)
		result := "success"
		if recorder.status >= 400 {
			result = "failed"
		}
		// Only method and URL path are retained. Bodies, query strings, credentials,
		// proxy values and signed download URLs are deliberately never audited.
		_ = a.store.AddAudit(session.Username, r.Method+" "+r.URL.Path, "http", "", result, map[string]any{"status": recorder.status})
	})
}

type auditResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *auditResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// viewerAllowed reports whether a non-admin ("viewer") account may call a
// route. Viewers get read-only access, except for a few self-service writes.
func viewerAllowed(method, path string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return !adminOnlyRead(path)
	}
	switch {
	case method == http.MethodPost && path == "/api/auth/logout":
		return true
	case method == http.MethodPost && path == "/api/auth/password":
		return true
	case method == http.MethodPost && path == "/api/notifications/read-all":
		return true
	}
	return false
}

// adminOnlyRead lists read endpoints that expose secrets, credentials, local
// paths or destructive tooling and must stay admin-only.
func adminOnlyRead(path string) bool {
	switch {
	case path == "/api/settings":
		return true
	case path == "/api/backups", strings.HasPrefix(path, "/api/backups/"):
		return true
	case path == "/api/audit":
		return true
	case strings.HasPrefix(path, "/api/integrations/"):
		return true
	case path == "/api/recovery/latest":
		return true
	case strings.HasPrefix(path, "/api/media-integrity/"):
		return true
	case path == "/api/duplicates":
		return true
	case path == "/api/logs":
		return true
	case path == "/api/creators/search":
		return true
	case path == "/api/translations/export":
		return true
	case path == "/api/health":
		return true
	}
	return false
}

func (a *App) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.store.Users()
	if err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]any{"items": users})
}

func (a *App) createUser(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if decode(r, &input) != nil {
		fail(w, 400, errors.New("请求格式错误"))
		return
	}
	if err := validateCredentials(authInput{Username: input.Username, Password: input.Password}); err != nil {
		fail(w, 400, err)
		return
	}
	role := input.Role
	if role != "admin" {
		role = "viewer"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(w, 500, err)
		return
	}
	id, err := a.store.CreateUser(strings.TrimSpace(input.Username), string(hash), role)
	if err != nil {
		if strings.Contains(strings.ToUpper(err.Error()), "UNIQUE") {
			fail(w, 409, errors.New("用户名已存在"))
			return
		}
		fail(w, 500, err)
		return
	}
	respond(w, 201, map[string]any{"id": id, "username": strings.TrimSpace(input.Username), "role": role})
}

func (a *App) updateUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var input struct {
		Role string `json:"role"`
	}
	if decode(r, &input) != nil || (input.Role != "admin" && input.Role != "viewer") {
		fail(w, 400, errors.New("无效的角色"))
		return
	}
	user, err := a.store.UserByID(id)
	if err != nil {
		fail(w, 404, errors.New("用户不存在"))
		return
	}
	if user.Role == "admin" && input.Role != "admin" {
		if n, _ := a.store.CountAdmins(); n <= 1 {
			fail(w, 409, errors.New("不能降级最后一个管理员"))
			return
		}
	}
	if err := a.store.SetUserRole(id, input.Role); err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]any{"id": id, "role": input.Role})
}

func (a *App) setUserPassword(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var input struct {
		Password string `json:"password"`
	}
	if decode(r, &input) != nil {
		fail(w, 400, errors.New("请求格式错误"))
		return
	}
	if err := validateCredentials(authInput{Username: "placeholder", Password: input.Password}); err != nil {
		fail(w, 400, err)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(w, 500, err)
		return
	}
	if err := a.store.UpdateUserPassword(id, string(hash)); err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]bool{"updated": true})
}

func (a *App) deleteUser(w http.ResponseWriter, r *http.Request) {
	session, _ := r.Context().Value(authContextKey{}).(store.Session)
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if id == session.UserID {
		fail(w, 409, errors.New("不能删除当前登录账号"))
		return
	}
	user, err := a.store.UserByID(id)
	if err != nil {
		fail(w, 404, errors.New("用户不存在"))
		return
	}
	if user.Role == "admin" {
		if n, _ := a.store.CountAdmins(); n <= 1 {
			fail(w, 409, errors.New("不能删除最后一个管理员"))
			return
		}
	}
	if err := a.store.DeleteUser(id); err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]bool{"deleted": true})
}

func (a *App) changePassword(w http.ResponseWriter, r *http.Request) {
	session, _ := r.Context().Value(authContextKey{}).(store.Session)
	var input struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if decode(r, &input) != nil {
		fail(w, 400, errors.New("请求格式错误"))
		return
	}
	user, err := a.store.UserByID(session.UserID)
	if err != nil {
		fail(w, 500, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.CurrentPassword)) != nil {
		fail(w, 400, errors.New("当前密码错误"))
		return
	}
	if err := validateCredentials(authInput{Username: user.Username, Password: input.NewPassword}); err != nil {
		fail(w, 400, err)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		fail(w, 500, err)
		return
	}
	if err := a.store.UpdateUserPassword(session.UserID, string(hash)); err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]bool{"updated": true})
}
