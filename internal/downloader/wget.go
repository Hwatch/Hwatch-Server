package downloader

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

type wgetTask struct {
	cmd              *exec.Cmd
	path, state, err string
	lastSize         int64
	lastCheck        time.Time
}

type Wget struct {
	mu        sync.Mutex
	tasks     map[string]*wgetTask
	max       int
	limitKBPS int64
}

func NewWget() *Wget     { return &Wget{tasks: map[string]*wgetTask{}, max: 2} }
func (*Wget) ID() string { return "wget" }
func (w *Wget) Add(_ context.Context, r Request) (string, error) {
	w.mu.Lock()
	active := 0
	for _, t := range w.tasks {
		if t.state == "active" || t.state == "waiting" {
			active++
		}
	}
	if active >= w.max {
		w.mu.Unlock()
		return "", fmt.Errorf("wget capacity reached: %w", ErrCapacityReached)
	}
	if _, err := exec.LookPath("wget"); err != nil {
		w.mu.Unlock()
		return "", err
	}
	path := filepath.Join(r.Directory, r.FileName)
	args := []string{"-c", "--no-verbose", "--tries=12", "--timeout=60", "-O", path}
	if r.UserAgent != "" {
		args = append(args, "--user-agent", r.UserAgent)
	}
	if r.Referer != "" {
		args = append(args, "--referer", r.Referer)
	}
	if w.limitKBPS > 0 {
		args = append(args, "--limit-rate", strconv.FormatInt(w.limitKBPS, 10)+"k")
	}
	args = append(args, r.URL)
	cmd := exec.Command("wget", args...)
	cmd.Stdout, cmd.Stderr = nil, nil
	cmd.Env = os.Environ()
	if r.Proxy != "" {
		cmd.Env = append(cmd.Env, "http_proxy="+r.Proxy, "https_proxy="+r.Proxy)
	}
	if err := cmd.Start(); err != nil {
		w.mu.Unlock()
		return "", err
	}
	id := uuid.NewString()
	t := &wgetTask{cmd: cmd, path: path, state: "active", lastCheck: time.Now()}
	w.tasks[id] = t
	w.mu.Unlock()
	go func() {
		err := cmd.Wait()
		w.mu.Lock()
		if err != nil {
			t.state, t.err = "error", err.Error()
		} else {
			t.state = "complete"
		}
		w.mu.Unlock()
	}()
	return id, nil
}
func (w *Wget) Tell(_ context.Context, id string) (Status, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	t, ok := w.tasks[id]
	if !ok {
		return Status{}, errors.New("wget task is not attached to this process")
	}
	now := time.Now()
	size := int64(0)
	if info, err := os.Stat(t.path); err == nil {
		size = info.Size()
	}
	speed := int64(0)
	if elapsed := now.Sub(t.lastCheck).Seconds(); elapsed > 0 {
		speed = int64(float64(size-t.lastSize) / elapsed)
	}
	t.lastSize, t.lastCheck = size, now
	total := int64(0)
	if t.state == "complete" {
		total = size
	}
	return Status{ExternalID: id, State: t.state, ErrorMessage: t.err, Completed: size, Total: total, DownloadSpeed: speed, Connections: 1, Files: []string{t.path}}, nil
}
func (w *Wget) signal(id string, sig syscall.Signal, state string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	t, ok := w.tasks[id]
	if !ok || t.cmd.Process == nil {
		return errors.New("wget task not found")
	}
	if err := t.cmd.Process.Signal(sig); err != nil {
		return err
	}
	t.state = state
	return nil
}
func (w *Wget) Pause(_ context.Context, id string) error {
	return w.signal(id, syscall.SIGSTOP, "paused")
}
func (w *Wget) Resume(_ context.Context, id string) error {
	return w.signal(id, syscall.SIGCONT, "active")
}
func (w *Wget) Remove(_ context.Context, id string) error {
	return w.signal(id, syscall.SIGTERM, "removed")
}
func (w *Wget) PauseAll(ctx context.Context) error {
	w.mu.Lock()
	ids := []string{}
	for id, t := range w.tasks {
		if t.state == "active" {
			ids = append(ids, id)
		}
	}
	w.mu.Unlock()
	for _, id := range ids {
		_ = w.Pause(ctx, id)
	}
	return nil
}
func (w *Wget) ResumeAll(ctx context.Context) error {
	w.mu.Lock()
	ids := []string{}
	for id, t := range w.tasks {
		if t.state == "paused" {
			ids = append(ids, id)
		}
	}
	w.mu.Unlock()
	for _, id := range ids {
		_ = w.Resume(ctx, id)
	}
	return nil
}
func (w *Wget) SetGlobal(_ context.Context, concurrent int, limit int64) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if concurrent < 1 {
		concurrent = 1
	}
	w.max, w.limitKBPS = concurrent, limit
	return nil
}
func (w *Wget) Health(ctx context.Context) Health {
	path, err := exec.LookPath("wget")
	h := Health{ID: "wget", Name: "wget", Online: err == nil}
	if err != nil {
		h.Error = err.Error()
		return h
	}
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		h.Error = err.Error()
		h.Online = false
	} else {
		h.Version = strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	}
	w.mu.Lock()
	for _, task := range w.tasks {
		if task.state == "active" || task.state == "waiting" {
			h.Active++
			h.Connections++
		}
	}
	w.mu.Unlock()
	return h
}

var _ Engine = (*Wget)(nil)
var _ = fmt.Sprintf
