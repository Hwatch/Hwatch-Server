package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"r34watch/internal/store"
)

func newIntegrityApp(t *testing.T, mediaRoot string, mountRequired bool) (*App, *store.Store) {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "integrity.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	cfg, _ := s.Settings()
	cfg.DownloadDir = mediaRoot
	cfg.StorageMountRequired = mountRequired
	if err := s.SaveSettings(cfg); err != nil {
		t.Fatal(err)
	}
	return New(s), s
}

func seedCompletedMedia(t *testing.T, s *store.Store, id, platform, path string, create bool) store.Video {
	t.Helper()
	slug := platform + ":creator"
	if _, err := s.DB.Exec("INSERT OR IGNORE INTO subscriptions(slug,name,url,platform,subscription_type,source_id,created_at,enabled) VALUES(?,?,?,?,?,?,?,1)",
		slug, "c", "https://example.invalid/models/c/", platform, "creator", "c", store.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec("INSERT INTO videos(id,subscription_slug,platform,title,page_url,discovered_at,updated_at,status,file_path,plex_path,duration_seconds) VALUES(?,?,?,?,?,?,?,?,?,?,?)",
		id, slug, platform, "title "+id, "https://example.invalid/v/"+id, store.Now(), store.Now(), "completed", path, path, 60); err != nil {
		t.Fatal(err)
	}
	if create {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("payload"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	v, _ := s.Video(id)
	return v
}

func TestMediaScanClassifiesMissingAndPresent(t *testing.T) {
	root := t.TempDir()
	a, s := newIntegrityApp(t, root, false)
	seedCompletedMedia(t, s, "rule34video:1", "rule34video", filepath.Join(root, "A", "keep.mp4"), true)
	seedCompletedMedia(t, s, "rule34video:2", "rule34video", filepath.Join(root, "A", "gone.mp4"), false)
	seedCompletedMedia(t, s, "rule34video:3", "rule34video", "", false) // path invalid

	result := a.scanMediaLibrary(1000)
	if !result.StorageAvailable {
		t.Fatal("storage should be available")
	}
	if result.Total != 3 || result.Present != 1 || result.Missing != 1 || result.PathInvalid != 1 {
		t.Fatalf("scan result = %+v", result)
	}
}

func TestMediaScanRefusesWhenStorageUnavailable(t *testing.T) {
	root := t.TempDir()
	missingRoot := filepath.Join(root, "not-mounted")
	a, s := newIntegrityApp(t, missingRoot, false)
	// Seed against the real root but point settings at the missing mount.
	seedCompletedMedia(t, s, "rule34video:1", "rule34video", filepath.Join(root, "A", "gone.mp4"), false)
	result := a.scanMediaLibrary(1000)
	if result.StorageAvailable {
		t.Fatal("storage must be reported unavailable")
	}
	if result.Items == nil || len(result.Items) != 0 {
		t.Fatalf("must not classify anything missing while storage is down: %+v", result.Items)
	}
}

func TestMediaScanDetectsMovedFile(t *testing.T) {
	root := t.TempDir()
	a, s := newIntegrityApp(t, root, false)
	expected := filepath.Join(root, "A", "video.mp4")
	seedCompletedMedia(t, s, "rule34video:1", "rule34video", expected, false)
	// Same stem, different extension => moved, not deleted.
	if err := os.MkdirAll(filepath.Dir(expected), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "A", "video.mkv"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := a.scanMediaLibrary(1000)
	if result.Moved != 1 || result.Missing != 0 {
		t.Fatalf("expected moved, got %+v", result)
	}
}

func vdfJSON(items ...vdfDeletion) []byte {
	b, _ := json.Marshal(vdfList{Version: 1, Deletions: items})
	return b
}

func TestVDFPreviewStatuses(t *testing.T) {
	root := t.TempDir()
	a, s := newIntegrityApp(t, root, false)
	candPath := filepath.Join(root, "A", "dup.mp4")
	keepPath := filepath.Join(root, "A", "keep.mp4")
	seedCompletedMedia(t, s, "rule34video:dup", "rule34video", candPath, false)
	seedCompletedMedia(t, s, "rule34video:keep", "rule34video", keepPath, true)
	seedCompletedMedia(t, s, "rule34video:present", "rule34video", filepath.Join(root, "A", "present.mp4"), true)
	seedCompletedMedia(t, s, "rule34video:active", "rule34video", filepath.Join(root, "A", "act.mp4"), false)
	_, _ = s.DB.Exec("UPDATE videos SET status='active' WHERE id='rule34video:active'")
	seedCompletedMedia(t, s, "rule34video:excluded", "rule34video", filepath.Join(root, "A", "ex.mp4"), false)
	if err := s.AddMediaExclusion(store.MediaExclusion{VideoID: "rule34video:excluded", Source: "rule34video", Reason: "user_delete"}); err != nil {
		t.Fatal(err)
	}

	deletions := []vdfDeletion{
		{VideoID: "rule34video:dup", OriginalPath: candPath, Reason: "duplicate", KeptVideoID: "rule34video:keep"},
		{VideoID: "rule34video:present", OriginalPath: filepath.Join(root, "A", "present.mp4"), Reason: "duplicate"},
		{VideoID: "rule34video:active", OriginalPath: filepath.Join(root, "A", "act.mp4"), Reason: "duplicate"},
		{VideoID: "rule34video:missingid", Reason: "duplicate"},
		{VideoID: "rule34video:excluded", Reason: "duplicate"},
		{VideoID: "rule34video:dup", OriginalPath: filepath.Join(root, "wrong", "dup.mp4"), Reason: "duplicate"},
		{VideoID: "rule34video:keep", OriginalPath: filepath.Join(root, "A", "moved.mp4"), Reason: "duplicate"},
	}
	preview := a.previewMediaCleanup(deletions)
	if preview.Candidates != 1 {
		t.Fatalf("candidates=%d want 1 (%+v)", preview.Candidates, preview.Items)
	}
	got := map[string]string{}
	for _, item := range preview.Items {
		got[item.VideoID+"/"+item.Status] = item.Status
	}
	want := []string{
		"rule34video:dup/" + cleanupCandidate,
		"rule34video:present/" + cleanupFilePresent,
		"rule34video:active/" + cleanupActive,
		"rule34video:missingid/" + cleanupUnknownVideo,
		"rule34video:excluded/" + cleanupAlreadyExcluded,
	}
	for _, key := range want {
		if _, ok := got[key]; !ok {
			t.Fatalf("missing preview status %s in %+v", key, preview.Items)
		}
	}
	// The second dup entry has a mismatched path => path_mismatch.
	if got["rule34video:dup/"+cleanupCandidate] == "" {
		t.Fatal("expected candidate present")
	}
}

func TestApplyCleanupDryRunThenConfirmAndIdempotent(t *testing.T) {
	root := t.TempDir()
	a, s := newIntegrityApp(t, root, false)
	dupPath := filepath.Join(root, "A", "dup.mp4")
	seedCompletedMedia(t, s, "rule34video:dup", "rule34video", dupPath, false)
	seedCompletedMedia(t, s, "rule34video:keep", "rule34video", filepath.Join(root, "A", "keep.mp4"), true)
	deletions := []vdfDeletion{{VideoID: "rule34video:dup", OriginalPath: dupPath, Reason: "duplicate", KeptVideoID: "rule34video:keep"}}

	// Dry run must not change anything.
	if _, err := a.applyMediaCleanup(deletions, false); err != nil {
		t.Fatal(err)
	}
	if v, _ := s.Video("rule34video:dup"); v.Status != "completed" || s.IsMediaExcluded("rule34video:dup") {
		t.Fatalf("dry run mutated state: %+v excluded=%v", v, s.IsMediaExcluded("rule34video:dup"))
	}

	res, err := a.applyMediaCleanup(deletions, true)
	if err != nil || res.Executed != 1 {
		t.Fatalf("apply: res=%+v err=%v", res, err)
	}
	dup, _ := s.Video("rule34video:dup")
	if dup.Status != "removed" || dup.FilePath != "" || !s.IsMediaExcluded("rule34video:dup") {
		t.Fatalf("dup not removed/excluded: %+v excluded=%v", dup, s.IsMediaExcluded("rule34video:dup"))
	}
	// Kept duplicate stays intact and is NOT excluded.
	keep, _ := s.Video("rule34video:keep")
	if keep.Status != "completed" || s.IsMediaExcluded("rule34video:keep") {
		t.Fatalf("kept video was touched: %+v", keep)
	}
	// Re-importing the same list is idempotent.
	res2, _ := a.applyMediaCleanup(deletions, true)
	if res2.Skipped != 1 || res2.Executed != 0 {
		t.Fatalf("second apply not idempotent: %+v", res2)
	}
}

func TestApplyCleanupRefusesWhenStorageUnavailable(t *testing.T) {
	root := t.TempDir()
	missingRoot := filepath.Join(root, "gone")
	a, s := newIntegrityApp(t, missingRoot, false)
	dupPath := filepath.Join(root, "A", "dup.mp4")
	seedCompletedMedia(t, s, "rule34video:dup", "rule34video", dupPath, false)
	deletions := []vdfDeletion{{VideoID: "rule34video:dup", OriginalPath: dupPath, Reason: "duplicate"}}
	if _, err := a.applyMediaCleanup(deletions, true); err == nil {
		t.Fatal("apply must refuse when storage is unavailable")
	}
	if v, _ := s.Video("rule34video:dup"); v.Status != "completed" || s.IsMediaExcluded("rule34video:dup") {
		t.Fatal("storage-unavailable apply must not mutate state")
	}
}

func TestRemoveAndExcludeTransactionalRollback(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "tx.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	// Unknown video: must roll back and leave no orphan exclusion.
	err = s.RemoveAndExclude("rule34video:ghost", store.MediaExclusion{VideoID: "rule34video:ghost", Source: "rule34video", Reason: "vdf_duplicate"})
	if err == nil {
		t.Fatal("expected error for unknown video")
	}
	if s.IsMediaExcluded("rule34video:ghost") {
		t.Fatal("orphan exclusion was committed after a failed transaction")
	}
}

func TestFileReappearsBlocksExclusion(t *testing.T) {
	root := t.TempDir()
	a, s := newIntegrityApp(t, root, false)
	path := filepath.Join(root, "A", "dup.mp4")
	seedCompletedMedia(t, s, "rule34video:dup", "rule34video", path, false)
	deletions := []vdfDeletion{{VideoID: "rule34video:dup", OriginalPath: path, Reason: "duplicate"}}
	if preview := a.previewMediaCleanup(deletions); preview.Candidates != 1 {
		t.Fatalf("preview candidates=%d", preview.Candidates)
	}
	// File reappears before apply.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("back"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, _ := a.applyMediaCleanup(deletions, true)
	if res.Executed != 0 || res.Skipped != 1 {
		t.Fatalf("reappeared file must be skipped: %+v", res)
	}
	if s.IsMediaExcluded("rule34video:dup") {
		t.Fatal("reappeared file must not be excluded")
	}
}

func TestParseVDFAcceptsArrayAndObject(t *testing.T) {
	if _, err := parseVDFDeletionList([]byte(`[{"video_id":"a"}]`)); err != nil {
		t.Fatalf("array list: %v", err)
	}
	if _, err := parseVDFDeletionList(vdfJSON(vdfDeletion{VideoID: "a"})); err != nil {
		t.Fatalf("object list: %v", err)
	}
	if _, err := parseVDFDeletionList([]byte("   ")); err == nil {
		t.Fatal("empty list must error")
	}
	if _, err := parseVDFDeletionList([]byte("{bad")); err == nil {
		t.Fatal("invalid json must error")
	}
}
