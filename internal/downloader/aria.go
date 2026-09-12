package downloader

import (
	"context"

	"r34watch/internal/aria2"
)

type Aria struct{ client *aria2.Client }

func NewAria(rawURL, secret string) *Aria { return &Aria{client: aria2.New(rawURL, secret)} }
func (*Aria) ID() string                  { return "aria2" }
func (a *Aria) Add(ctx context.Context, r Request) (string, error) {
	return a.client.Add(ctx, r.URL, r.Directory, r.FileName, r.Referer, r.UserAgent, r.Proxy, r.Connections)
}
func (a *Aria) Tell(ctx context.Context, id string) (Status, error) {
	s, err := a.client.Tell(ctx, id)
	return Status{ExternalID: s.GID, State: s.State, ErrorCode: s.ErrorCode, ErrorMessage: s.ErrorMessage, Directory: s.Dir, Completed: s.Completed, Total: s.Total, DownloadSpeed: s.DownloadSpeed, Connections: s.Connections, Files: s.Files}, err
}
func (a *Aria) Pause(ctx context.Context, id string) error  { return a.client.Pause(ctx, id) }
func (a *Aria) Resume(ctx context.Context, id string) error { return a.client.Resume(ctx, id) }
func (a *Aria) Remove(ctx context.Context, id string) error { return a.client.Remove(ctx, id) }
func (a *Aria) PauseAll(ctx context.Context) error          { return a.client.PauseAll(ctx) }
func (a *Aria) ResumeAll(ctx context.Context) error         { return a.client.ResumeAll(ctx) }
func (a *Aria) SetGlobal(ctx context.Context, concurrent int, limit int64) error {
	return a.client.SetGlobal(ctx, concurrent, limit)
}
func (a *Aria) Health(ctx context.Context) Health {
	err := a.client.Version(ctx)
	h := Health{ID: "aria2", Name: "aria2", Online: err == nil, SupportsMulti: true}
	if err != nil {
		h.Error = err.Error()
	}
	return h
}
