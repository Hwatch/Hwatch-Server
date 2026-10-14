package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"r34watch/internal/store"
)

// saveUploadedAvatar reads an uploaded image (multipart field "file"), validates
// it is a supported image and stores it under <download_dir>/.hwatch-avatars.
func (a *App) saveUploadedAvatar(r *http.Request) (string, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 6<<20)
	if err := r.ParseMultipartForm(6 << 20); err != nil {
		return "", errors.New("图片读取失败（最大 6MB）")
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		return "", errors.New("缺少图片文件")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 6<<20))
	if err != nil || len(data) == 0 {
		return "", errors.New("图片为空或读取失败")
	}
	ext := map[string]string{
		"image/png":  ".png",
		"image/jpeg": ".jpg",
		"image/webp": ".webp",
		"image/gif":  ".gif",
	}[http.DetectContentType(data)]
	if ext == "" {
		return "", errors.New("仅支持 PNG / JPEG / WebP / GIF 图片")
	}
	cfg, _ := a.store.Settings()
	dir := filepath.Join(cfg.DownloadDir, ".hwatch-avatars")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	path := filepath.Join(dir, fmt.Sprintf("custom-%x%s", sum[:12], ext))
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) userAvatar(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	user, err := a.store.UserByID(id)
	if err != nil {
		a.avatarPlaceholder(w, "H")
		return
	}
	if user.Avatar != "" {
		if _, statErr := os.Stat(user.Avatar); statErr == nil {
			if kind := mime.TypeByExtension(filepath.Ext(user.Avatar)); kind != "" {
				w.Header().Set("Content-Type", kind)
			}
			w.Header().Set("Cache-Control", "private, max-age=300")
			http.ServeFile(w, r, user.Avatar)
			return
		}
	}
	a.avatarPlaceholder(w, user.Username)
}

func (a *App) uploadMyAvatar(w http.ResponseWriter, r *http.Request) {
	session, _ := r.Context().Value(authContextKey{}).(store.Session)
	path, err := a.saveUploadedAvatar(r)
	if err != nil {
		fail(w, 400, err)
		return
	}
	if err := a.store.SetUserAvatar(session.UserID, path); err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]any{"avatar_url": fmt.Sprintf("/avatar/user/%d", session.UserID)})
}

func (a *App) deleteMyAvatar(w http.ResponseWriter, r *http.Request) {
	session, _ := r.Context().Value(authContextKey{}).(store.Session)
	if err := a.store.SetUserAvatar(session.UserID, ""); err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]bool{"cleared": true})
}

func (a *App) uploadUserAvatar(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if _, err := a.store.UserByID(id); err != nil {
		fail(w, 404, errors.New("用户不存在"))
		return
	}
	path, err := a.saveUploadedAvatar(r)
	if err != nil {
		fail(w, 400, err)
		return
	}
	if err := a.store.SetUserAvatar(id, path); err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]any{"avatar_url": fmt.Sprintf("/avatar/user/%d", id)})
}

func (a *App) deleteUserAvatar(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := a.store.SetUserAvatar(id, ""); err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]bool{"cleared": true})
}

func (a *App) uploadSubscriptionAvatar(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if _, err := a.store.Subscription(slug); err != nil {
		fail(w, 404, errors.New("订阅不存在"))
		return
	}
	path, err := a.saveUploadedAvatar(r)
	if err != nil {
		fail(w, 400, err)
		return
	}
	if err := a.store.SetSubscriptionAvatar(slug, path); err != nil {
		fail(w, 500, err)
		return
	}
	sum := sha256.Sum256([]byte(path))
	respond(w, 200, map[string]any{"avatar_proxy_url": fmt.Sprintf("/avatar/%s?v=%x", slug, sum[:4])})
}

func (a *App) deleteSubscriptionAvatar(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if _, err := a.store.Subscription(slug); err != nil {
		fail(w, 404, errors.New("订阅不存在"))
		return
	}
	if err := a.store.SetSubscriptionAvatar(slug, ""); err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]bool{"cleared": true})
}
