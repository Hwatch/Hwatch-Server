package downloader

import (
	"context"
	"time"

	"r34watch/internal/aria2"
)

type Aria struct{ client *aria2.Client }

func NewAria(rawURL, secret string) *Aria { return &Aria{client: aria2.New(rawURL, secret)} }
func (*Aria) ID() string                  { return "aria2" }
func (a *Aria) Add(ctx context.Context, r Request) (string, error) {
	return a.client.Add(ctx, r.URL, r.Directory, r.FileName, r.Referer, r.UserAgent, r.Proxy, r.ExternalID, r.Connections)
}
func (a *Aria) Tell(ctx context.Context, id string) (Status, error) {
	s, err := a.client.Tell(ctx, id)
	return Status{ExternalID: s.GID, State: s.State, ErrorCode: s.ErrorCode, ErrorMessage: s.ErrorMessage, Directory: s.Dir, Completed: s.Completed, Total: s.Total, DownloadSpeed: s.DownloadSpeed, Connections: s.Connections, Files: s.Files}, err
}
func (a *Aria) Pause(ctx context.Context, id string) error  { return a.client.Pause(ctx, id) }
func (a *Aria) Resume(ctx context.Context, id string) error { return a.client.Resume(ctx, id) }
func (a *Aria) Remove(ctx context.Context, id string) error { return a.client.Remove(ctx, id) }
func (a *Aria) CleanResult(ctx context.Context, id string) error {
	return a.client.RemoveResult(ctx, id)
}
func (a *Aria) PauseAll(ctx context.Context) error  { return a.client.PauseAll(ctx) }
func (a *Aria) ResumeAll(ctx context.Context) error { return a.client.ResumeAll(ctx) }
func (a *Aria) SetGlobal(ctx context.Context, concurrent int, limit int64) error {
	return a.client.SetGlobal(ctx, concurrent, limit)
}
func (a *Aria) Health(ctx context.Context) Health {
	check := func() error {
		attempt, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		return a.client.Version(attempt)
	}
	err := check()
	if err != nil && ctx.Err() == nil {
		select {
		case <-time.After(150 * time.Millisecond):
			err = check()
		case <-ctx.Done():
			err = ctx.Err()
		}
	}
	h := Health{ID: "aria2", Name: "aria2", Online: err == nil, SupportsMulti: true}
	if err != nil {
		h.Error = err.Error()
	}
	return h
}
