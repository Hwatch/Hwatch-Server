package plex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"r34watch/internal/store"
)

func TestPlexLayoutAndNFO(t *testing.T) {
	root := t.TempDir()
	sub := store.Subscription{Slug: "artist", Name: "Artist"}
	v := store.Video{ID: "1", Title: "Episode: One", PageURL: "https://example.invalid", PlexSeason: 1, PlexEpisode: 2, DurationSeconds: 125, PublishedAt: "2026-09-11"}
	layout := Build(root, sub, v)
	if !strings.Contains(layout.FullPath, filepath.Join("Artist", "Season 01", "Artist - S01E002 - Episode_ One.mp4")) {
		t.Fatal(layout.FullPath)
	}
	if err := Generate(layout, sub, v, ""); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(layout.ShowDir, "tvshow.nfo"), strings.TrimSuffix(layout.FullPath, ".mp4") + ".nfo"} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
	translated := v
	translated.Title = "中文标题"
	if err := GenerateNFO(Existing(root, layout.FullPath), sub, translated); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(strings.TrimSuffix(layout.FullPath, ".mp4") + ".nfo")
	if err != nil || !strings.Contains(string(raw), "中文标题") {
		t.Fatalf("translated NFO not written: %v %s", err, raw)
	}
}
