package plex

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"r34watch/internal/store"
)

type Layout struct{ Root, ShowDir, SeasonDir, FileName, FullPath string }

func Build(root string, sub store.Subscription, v store.Video) Layout {
	show := safe(sub.Name)
	if show == "" {
		show = safe(sub.Slug)
	}
	season := fmt.Sprintf("Season %02d", v.PlexSeason)
	name := fmt.Sprintf("%s - S%02dE%03d - %s.mp4", show, v.PlexSeason, v.PlexEpisode, safe(v.Title))
	showDir := filepath.Join(root, show)
	seasonDir := filepath.Join(showDir, season)
	return Layout{Root: root, ShowDir: showDir, SeasonDir: seasonDir, FileName: name, FullPath: filepath.Join(seasonDir, name)}
}
func Existing(root, fullPath string) Layout {
	seasonDir := filepath.Dir(fullPath)
	showDir := filepath.Dir(seasonDir)
	return Layout{Root: root, ShowDir: showDir, SeasonDir: seasonDir, FileName: filepath.Base(fullPath), FullPath: fullPath}
}
func Generate(layout Layout, sub store.Subscription, v store.Video, proxy string) error {
	if err := os.MkdirAll(layout.SeasonDir, 0755); err != nil {
		return err
	}
	if err := GenerateNFO(layout, sub, v); err != nil {
		return err
	}
	if sub.AvatarURL != "" {
		_ = fetchImage(sub.AvatarURL, filepath.Join(layout.ShowDir, "poster.jpg"), proxy)
	}
	if v.ThumbnailURL != "" {
		base := strings.TrimSuffix(layout.FullPath, filepath.Ext(layout.FullPath))
		_ = fetchImage(v.ThumbnailURL, base+".jpg", proxy)
	}
	return nil
}
func GenerateNFO(layout Layout, sub store.Subscription, v store.Video) error {
	if err := os.MkdirAll(layout.SeasonDir, 0755); err != nil {
		return err
	}
	showNFO := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<tvshow><title>" + xml(sub.Name) + "</title><plot>" + xml("Videos by "+sub.Name) + "</plot><studio>" + xml(sub.Name) + "</studio></tvshow>\n"
	if err := writeIfMissing(filepath.Join(layout.ShowDir, "tvshow.nfo"), showNFO); err != nil {
		return err
	}
	base := strings.TrimSuffix(layout.FullPath, filepath.Ext(layout.FullPath))
	episode := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<episodedetails><title>" + xml(v.Title) + "</title><showtitle>" + xml(sub.Name) + "</showtitle><season>" + fmt.Sprint(v.PlexSeason) + "</season><episode>" + fmt.Sprint(v.PlexEpisode) + "</episode><aired>" + xml(v.PublishedAt) + "</aired><runtime>" + fmt.Sprint(v.DurationSeconds/60) + "</runtime><plot>" + xml("Source: "+v.PageURL) + "</plot></episodedetails>\n"
	if err := os.WriteFile(base+".nfo", []byte(episode), 0644); err != nil {
		return err
	}
	return nil
}
func writeIfMissing(path, value string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return os.WriteFile(path, []byte(value), 0644)
}
func fetchImage(raw, destination, proxy string) error {
	if _, err := os.Stat(destination); err == nil {
		return nil
	}
	transport := &http.Transport{}
	if proxy != "" {
		if p, err := url.Parse(proxy); err == nil {
			transport.Proxy = http.ProxyURL(p)
		}
	}
	client := &http.Client{Transport: transport, Timeout: 30 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, raw, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("image HTTP %d", res.StatusCode)
	}
	tmp := destination + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, io.LimitReader(res.Body, 20<<20))
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(tmp, destination)
}
func safe(value string) string {
	value = html.UnescapeString(value)
	value = regexp.MustCompile("[<>:\"/\\\\|?*\\x00-\\x1F]").ReplaceAllString(value, "_")
	value = strings.Trim(value, " .")
	if value == "" {
		return "Untitled"
	}
	r := []rune(value)
	if len(r) > 120 {
		return string(r[:120])
	}
	return value
}
func xml(value string) string { return html.EscapeString(value) }
