package adapters

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"r34watch/internal/collector"
	"r34watch/internal/store"
)

type Rule34Video struct{ collector collector.Collector }

func (Rule34Video) ID() string   { return "rule34video" }
func (Rule34Video) Name() string { return "Rule34Video" }
func (Rule34Video) MatchURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && strings.EqualFold(u.Hostname(), "rule34video.com") && strings.HasPrefix(u.Path, "/models/")
}
func (a Rule34Video) SearchCreators(_ context.Context, query, proxy string) ([]CreatorResult, error) {
	models, err := a.collector.SearchModels(query, proxy)
	if err != nil {
		return nil, err
	}
	out := make([]CreatorResult, 0, len(models))
	for _, model := range models {
		out = append(out, CreatorResult{Platform: a.ID(), SourceID: model.ID, Name: model.Name, URL: model.URL, AvatarURL: model.AvatarURL, VideoCount: model.VideoCount})
	}
	return out, nil
}
func (Rule34Video) ParseSubscription(_ context.Context, raw, _ string) (store.Subscription, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return store.Subscription{}, err
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 || parts[0] != "models" || parts[1] == "" {
		return store.Subscription{}, store.ErrInvalidSource
	}
	sourceID := parts[1]
	return store.Subscription{Slug: "rule34video:" + sourceID, Platform: "rule34video", Type: "creator", SourceID: sourceID, Name: sourceID, URL: raw, Enabled: true, ExternalLinks: "[]", AdapterConfig: "{}", Cursor: "{}"}, nil
}
func (a Rule34Video) ListVideos(_ context.Context, sub store.Subscription, proxy string) ([]store.Video, error) {
	items, err := a.collector.List(sub.URL, proxy)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].SourceID = items[i].ID
		items[i].Platform = a.ID()
		items[i].ID = a.ID() + ":" + items[i].SourceID
		items[i].SubscriptionSlug = sub.Slug
		relation, _ := json.Marshal(map[string]string{"type": "creator_listing", "platform": a.ID(), "subscription_slug": sub.Slug, "subscription_url": sub.URL, "creator_key": sub.SourceID})
		items[i].SourceRelationJSON = string(relation)
	}
	return items, nil
}
func (a Rule34Video) FetchDetails(_ context.Context, video store.Video, proxy string) (store.Video, error) {
	return a.collector.Detail(video, proxy)
}
func (a Rule34Video) ResolveDownload(_ context.Context, video store.Video, proxy string) (DownloadSource, error) {
	u, quality, err := a.collector.Highest(video.PageURL, proxy)
	return DownloadSource{URL: u, Quality: quality}, err
}
func (a Rule34Video) ParseImported(raw string) (store.Video, error) {
	v, err := a.collector.ParseImported(raw)
	if err == nil {
		v.SourceID = v.ID
		v.Platform = a.ID()
		v.ID = a.ID() + ":" + v.SourceID
	}
	return v, err
}
