package adapters

import (
	"context"
	"errors"
	"sync"

	"r34watch/internal/store"
)

type DownloadSource struct {
	URL       string
	Quality   int64
	Headers   map[string]string
	ExpiresAt string
}
type CreatorResult struct {
	Platform   string `json:"platform"`
	SourceID   string `json:"source_id"`
	Name       string `json:"name"`
	URL        string `json:"url"`
	AvatarURL  string `json:"avatar_url"`
	VideoCount int64  `json:"video_count"`
}

type Adapter interface {
	ID() string
	Name() string
	MatchURL(string) bool
	SearchCreators(context.Context, string, string) ([]CreatorResult, error)
	ParseSubscription(context.Context, string, string) (store.Subscription, error)
	ListVideos(context.Context, store.Subscription, string) ([]store.Video, error)
	FetchDetails(context.Context, store.Video, string) (store.Video, error)
	ResolveDownload(context.Context, store.Video, string) (DownloadSource, error)
	ParseImported(string) (store.Video, error)
}

// SubscriptionEnricher is optional for platforms whose display name or avatar
// can only be learned from the subscription page itself.
type SubscriptionEnricher interface {
	EnrichSubscription(context.Context, store.Subscription, string) store.Subscription
}

// ThumbnailCandidateProvider is optionally implemented by adapters that can
// enumerate cover candidates for a video, best-first. Enumerating candidates is
// expensive, so it is only used by the thumbnail maintenance pass, never by the
// hot /thumb read path.
type ThumbnailCandidateProvider interface {
	ThumbnailCandidates(context.Context, store.Video, string) ([]string, error)
}

type Registry struct {
	mu   sync.RWMutex
	byID map[string]Adapter
}

func NewRegistry(items ...Adapter) *Registry {
	r := &Registry{byID: map[string]Adapter{}}
	for _, item := range items {
		r.Register(item)
	}
	return r
}
func (r *Registry) Register(adapter Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[adapter.ID()] = adapter
}
func (r *Registry) ByID(id string) (Adapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.byID[id]
	if !ok {
		return nil, errors.New("没有可用的平台适配器: " + id)
	}
	return a, nil
}
func (r *Registry) Match(rawURL string) (Adapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, adapter := range r.byID {
		if adapter.MatchURL(rawURL) {
			return adapter, nil
		}
	}
	return nil, errors.New("暂不支持该平台地址")
}
func (r *Registry) All() []Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Adapter, 0, len(r.byID))
	for _, item := range r.byID {
		out = append(out, item)
	}
	return out
}
