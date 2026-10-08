package main

import (
	"os"
	"path/filepath"
	"testing"

	"r34watch/internal/store"
)

func seedVPDFixture(t *testing.T) (*App, *store.Store, string) {
	t.Helper()
	root := t.TempDir()
	a, s := newIntegrityApp(t, root, false)
	return a, s, root
}

func TestVDFCrossGroupKeepRemoveConflictBlocksPlan(t *testing.T) {
	a, s, root := seedVPDFixture(t)
	dup := filepath.Join(root, "A", "dup.mp4")
	keep1 := filepath.Join(root, "A", "keep1.mp4")
	keep2 := filepath.Join(root, "A", "keep2.mp4")
	seedCompletedMedia(t, s, "rule34video:d", "rule34video", dup, true)
	seedCompletedMedia(t, s, "rule34video:k1", "rule34video", keep1, true)
	seedCompletedMedia(t, s, "rule34video:k2", "rule34video", keep2, true)

	export := vdfGUIExport{Groups: []vdfGUIGroup{
		{GroupID: "g1", RemoveItems: []vdfGUIItem{vdfItem(dup)}, KeepItems: []vdfGUIItem{vdfItem(keep1)}},
		// d is kept here but removed in g1 -> cross-group conflict.
		{GroupID: "g2", RemoveItems: []vdfGUIItem{vdfItem(keep2)}, KeepItems: []vdfGUIItem{vdfItem(dup)}},
	}}
	preview := a.previewVDFGUI(export)
	if !preview.HasConflicts || preview.CrossGroupConflicts == 0 {
		t.Fatalf("expected cross-group conflict: %+v", preview)
	}
	if preview.ConflictGroups != 2 {
		t.Fatalf("both groups should be conflict, got %d", preview.ConflictGroups)
	}
	// Execute must be blocked entirely, with zero side effects.
	if _, err := a.executeVDFGUIGroups(export, true); err == nil {
		t.Fatal("execute must be blocked on conflict")
	}
	if _, err := os.Stat(dup); err != nil {
		t.Fatal("remove file must be untouched")
	}
	if s.IsMediaExcluded("rule34video:d") {
		t.Fatal("no exclusion may be written on conflict")
	}
	plan := vdfPlanID(export)
	if items, _ := s.DeletionPlanItems(plan); len(items) != 0 {
		t.Fatalf("no plan items may be written on conflict: %+v", items)
	}
	if _, err := os.Stat(filepath.Join(root, quarantineDirName, plan)); !os.IsNotExist(err) {
		t.Fatal("no quarantine dir may be created on conflict")
	}
}

func TestVDFDuplicateRemoveConflict(t *testing.T) {
	a, s, root := seedVPDFixture(t)
	dup := filepath.Join(root, "A", "dup.mp4")
	k1 := filepath.Join(root, "A", "k1.mp4")
	k2 := filepath.Join(root, "A", "k2.mp4")
	seedCompletedMedia(t, s, "rule34video:d", "rule34video", dup, true)
	seedCompletedMedia(t, s, "rule34video:k1", "rule34video", k1, true)
	seedCompletedMedia(t, s, "rule34video:k2", "rule34video", k2, true)
	export := vdfGUIExport{Groups: []vdfGUIGroup{
		{GroupID: "g1", RemoveItems: []vdfGUIItem{vdfItem(dup)}, KeepItems: []vdfGUIItem{vdfItem(k1)}},
		{GroupID: "g2", RemoveItems: []vdfGUIItem{vdfItem(dup)}, KeepItems: []vdfGUIItem{vdfItem(k2)}},
	}}
	preview := a.previewVDFGUI(export)
	found := false
	for _, cf := range preview.Conflicts {
		if cf.Kind == "duplicate_remove" && cf.VideoID == "rule34video:d" {
			found = true
		}
	}
	if !found {
		t.Fatalf("duplicate remove conflict not detected: %+v", preview.Conflicts)
	}
	if _, err := a.executeVDFGUIGroups(export, true); err == nil {
		t.Fatal("execute must be blocked")
	}
}

func TestVDFSharedPhysicalPathConflict(t *testing.T) {
	a, s, root := seedVPDFixture(t)
	sharedPath := filepath.Join(root, "A", "shared.mp4")
	keep := filepath.Join(root, "A", "keep.mp4")
	seedCompletedMedia(t, s, "rule34video:d1", "rule34video", sharedPath, true)
	seedCompletedMedia(t, s, "rule34video:d2", "rule34video", sharedPath, false) // same physical path
	seedCompletedMedia(t, s, "rule34video:k", "rule34video", keep, true)
	export := vdfGUIExport{Groups: []vdfGUIGroup{{
		GroupID: "g1", RemoveItems: []vdfGUIItem{vdfItem(sharedPath)}, KeepItems: []vdfGUIItem{vdfItem(keep)},
	}}}
	preview := a.previewVDFGUI(export)
	if !preview.HasConflicts {
		t.Fatalf("shared physical path must conflict: %+v", preview)
	}
	if _, err := a.executeVDFGUIGroups(export, true); err == nil {
		t.Fatal("execute must be blocked")
	}
	if _, err := os.Stat(sharedPath); err != nil {
		t.Fatal("shared file must be untouched")
	}
}

func TestVDFAlreadyDoneReimportIsIdempotent(t *testing.T) {
	a, s, root := seedVPDFixture(t)
	dup := filepath.Join(root, "A", "dup.mp4")
	keep := filepath.Join(root, "A", "keep.mp4")
	seedCompletedMedia(t, s, "rule34video:d", "rule34video", dup, true)
	seedCompletedMedia(t, s, "rule34video:k", "rule34video", keep, true)
	export := vdfGUIExport{CreatedAt: "2026-05-05T00:00:00Z", Groups: []vdfGUIGroup{{
		GroupID: "g1", RemoveItems: []vdfGUIItem{vdfItem(dup)}, KeepItems: []vdfGUIItem{vdfItem(keep)},
	}}}
	if res, err := a.executeVDFGUIGroups(export, true); err != nil || res.Executed != 1 {
		t.Fatalf("first execute: %+v err=%v", res, err)
	}
	plan := vdfPlanID(export)
	q := quarantineTarget(root, plan, dup)

	preview := a.previewVDFGUI(export)
	if preview.HasConflicts {
		t.Fatalf("re-import must not report conflict: %+v", preview.Conflicts)
	}
	if preview.RemoveAlreadyDone != 1 || preview.AlreadyDoneGroups != 1 || preview.ReadyGroups != 0 {
		t.Fatalf("re-import should be already_done: %+v", preview)
	}
	// Second execute: no new ops.
	res, err := a.executeVDFGUIGroups(export, true)
	if err != nil || res.Executed != 0 {
		t.Fatalf("second execute must be a no-op: %+v err=%v", res, err)
	}
	if _, err := os.Stat(q); err != nil {
		t.Fatal("quarantined file must remain")
	}
	if n := countExclusions(t, s, "rule34video:d"); n != 1 {
		t.Fatalf("exclusion must not be duplicated: %d", n)
	}
}

func TestVDFAlreadyDoneDoesNotMaskKeepConflict(t *testing.T) {
	a, s, root := seedVPDFixture(t)
	dup := filepath.Join(root, "A", "dup.mp4")
	other := filepath.Join(root, "A", "other.mp4")
	seedCompletedMedia(t, s, "rule34video:d", "rule34video", dup, true)
	seedCompletedMedia(t, s, "rule34video:o", "rule34video", other, true)
	// First run removes d.
	first := vdfGUIExport{CreatedAt: "2026-06-06T00:00:00Z", Groups: []vdfGUIGroup{{
		GroupID: "g1", RemoveItems: []vdfGUIItem{vdfItem(dup)}, KeepItems: []vdfGUIItem{vdfItem(other)},
	}}}
	if _, err := a.executeVDFGUIGroups(first, true); err != nil {
		t.Fatal(err)
	}
	// Second export keeps the already-deleted d while removing other.
	second := vdfGUIExport{CreatedAt: "2026-06-07T00:00:00Z", Groups: []vdfGUIGroup{{
		GroupID: "g2", RemoveItems: []vdfGUIItem{vdfItem(other)}, KeepItems: []vdfGUIItem{vdfItem(dup)},
	}}}
	preview := a.previewVDFGUI(second)
	if !preview.HasConflicts {
		t.Fatalf("keep of an already-removed video must be a conflict: %+v", preview.Conflicts)
	}
	if _, err := a.executeVDFGUIGroups(second, true); err == nil {
		t.Fatal("execute must be blocked")
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatal("other file must remain untouched")
	}
}

func TestVDFPreviewThenStateChangeBlocks(t *testing.T) {
	a, s, root := seedVPDFixture(t)
	dup := filepath.Join(root, "A", "dup.mp4")
	keep := filepath.Join(root, "A", "keep.mp4")
	seedCompletedMedia(t, s, "rule34video:d", "rule34video", dup, false) // absent -> ready(absent)
	seedCompletedMedia(t, s, "rule34video:k", "rule34video", keep, true)
	export := vdfGUIExport{CreatedAt: "2026-07-07T00:00:00Z", Groups: []vdfGUIGroup{{
		GroupID: "g1", RemoveItems: []vdfGUIItem{{Path: dup, SizeBytes: 7}}, KeepItems: []vdfGUIItem{vdfItem(keep)},
	}}}
	if preview := a.previewVDFGUI(export); !preview.HasConflicts && preview.ReadyGroups != 1 {
		t.Fatalf("expected ready preview: %+v", preview)
	}
	// Keep video is removed out-of-band after preview.
	if err := s.RemoveAndExclude("rule34video:k", store.MediaExclusion{VideoID: "rule34video:k", Source: "rule34video", Reason: "user_delete"}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.executeVDFGUIGroups(export, true); err == nil {
		t.Fatal("execute must re-verify and block after keep removed")
	}
	if s.IsMediaExcluded("rule34video:d") {
		t.Fatal("remove target must not be processed")
	}
}

func TestVDFLegacyGroupIDCompat(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.AddSubscription(store.Subscription{Slug: "rule34video:c", Name: "c", URL: "https://example.invalid/models/c/", Platform: "rule34video", Enabled: true, CreatedAt: store.Now()}); err != nil {
		t.Fatal(err)
	}
	// Legacy row written without group_id (column added by migration).
	if _, err := s.DB.Exec("INSERT INTO deletion_plan_items(plan_id,video_id,original_path,status,reason,updated_at) VALUES(?,?,?,?,?,?)", "vdf-gui-old", "rule34video:old", "/mnt/x/old.mp4", "db_excluded", "legacy", store.Now()); err != nil {
		t.Fatal(err)
	}
	items, err := s.DeletionPlanItems("vdf-gui-old")
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	if items[0].GroupID != "" {
		t.Fatalf("legacy group_id must default to empty, got %q", items[0].GroupID)
	}
	byStatus, err := s.DeletionPlanItemsByStatus("db_excluded")
	if err != nil || len(byStatus) != 1 {
		t.Fatalf("byStatus=%+v err=%v", byStatus, err)
	}
}

func countExclusions(t *testing.T, s *store.Store, videoID string) int {
	t.Helper()
	var n int
	if err := s.DB.QueryRow("SELECT count(*) FROM media_exclusions WHERE video_id=?", videoID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
