package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func vdfGUIJSON(createdAt string, groups ...vdfGUIGroup) []byte {
	b, _ := json.Marshal(vdfGUIExport{CreatedAt: createdAt, Groups: groups})
	return b
}

func vdfItem(path string) vdfGUIItem {
	info, _ := os.Stat(path)
	size := int64(0)
	if info != nil {
		size = info.Size()
	}
	return vdfGUIItem{Path: path, SizeBytes: size, Resolution: "1920x1080"}
}

func TestDetectImportFormat(t *testing.T) {
	if got := detectImportFormat([]byte(`[{"video_id":"a"}]`)); got != importFormatLegacy {
		t.Fatalf("array => %s", got)
	}
	if got := detectImportFormat([]byte(`{"deletions":[{"video_id":"a"}]}`)); got != importFormatLegacy {
		t.Fatalf("deletions => %s", got)
	}
	if got := detectImportFormat([]byte(`{"kind":"transrunner-duplicate-deletion-plan","plan_version":1}`)); got != importFormatTransRunner {
		t.Fatalf("plan => %s", got)
	}
	if got := detectImportFormat([]byte(`{"CreatedAt":"x","Groups":[]}`)); got != importFormatVDFGUI {
		t.Fatalf("vdf => %s", got)
	}
	if got := detectImportFormat([]byte(`{bad`)); got != importFormatUnknown {
		t.Fatalf("invalid => %s", got)
	}
	if got := detectImportFormat([]byte(`{"foo":1}`)); got != importFormatUnknown {
		t.Fatalf("unknown => %s", got)
	}
}

func TestVDFGUISingleAndMultiRemove(t *testing.T) {
	root := t.TempDir()
	a, s := newIntegrityApp(t, root, false)
	dup1 := filepath.Join(root, "A", "dup1.mp4")
	dup2 := filepath.Join(root, "A", "dup2.mp4")
	keep := filepath.Join(root, "A", "keep.mp4")
	seedCompletedMedia(t, s, "rule34video:d1", "rule34video", dup1, true)
	seedCompletedMedia(t, s, "rule34video:d2", "rule34video", dup2, true)
	seedCompletedMedia(t, s, "rule34video:k", "rule34video", keep, true)

	single := vdfGUIExport{CreatedAt: "2026-01-01T00:00:00Z", Groups: []vdfGUIGroup{{
		GroupID: "g1", Reason: "duplicate",
		RemoveItems: []vdfGUIItem{vdfItem(dup1)},
		KeepItems:   []vdfGUIItem{vdfItem(keep)},
	}}}
	preview := a.previewVDFGUI(single)
	if preview.Format != importFormatVDFGUI || preview.Groups != 1 || preview.ReadyGroups != 1 || preview.MatchedItems != 2 {
		t.Fatalf("single preview = %+v", preview)
	}
	if preview.EstimatedFreed != int64(len("payload")) {
		t.Fatalf("freed=%d", preview.EstimatedFreed)
	}

	multi := vdfGUIExport{CreatedAt: "2026-01-02T00:00:00Z", Groups: []vdfGUIGroup{{
		GroupID: "g2", Reason: "duplicate",
		RemoveItems: []vdfGUIItem{vdfItem(dup1), vdfItem(dup2)},
		KeepItems:   []vdfGUIItem{vdfItem(keep)},
	}}}
	mp := a.previewVDFGUI(multi)
	if mp.ReadyGroups != 1 || mp.MatchedItems != 3 {
		t.Fatalf("multi preview = %+v", mp)
	}
}

func TestVDFGUIPathMapping(t *testing.T) {
	root := t.TempDir()
	a, s := newIntegrityApp(t, root, false)
	dup := filepath.Join(root, "Sugardust", "Season 01", "clip.mp4")
	keep := filepath.Join(root, "Sugardust", "Season 01", "keep.mp4")
	seedCompletedMedia(t, s, "rule34video:d", "rule34video", dup, true)
	seedCompletedMedia(t, s, "rule34video:k", "rule34video", keep, true)

	// macOS-style prefix resolves via two-component suffix.
	export := vdfGUIExport{Groups: []vdfGUIGroup{{
		GroupID:     "g",
		RemoveItems: []vdfGUIItem{{Path: "/Users/alice/Media/Sugardust/Season 01/clip.mp4", SizeBytes: int64(len("payload"))}},
		KeepItems:   []vdfGUIItem{{Path: "/Users/alice/Media/Sugardust/Season 01/keep.mp4", SizeBytes: int64(len("payload"))}},
	}}}
	preview := a.previewVDFGUI(export)
	if preview.ReadyGroups != 1 {
		t.Fatalf("macOS mapping failed: %+v", preview.Items)
	}
	if preview.Items[0].RemoveItems[0].VideoID != "rule34video:d" {
		t.Fatalf("matched wrong video: %+v", preview.Items[0].RemoveItems[0])
	}
}

func TestVDFGUIUnmatchedAndAmbiguous(t *testing.T) {
	root := t.TempDir()
	a, s := newIntegrityApp(t, root, false)
	seedCompletedMedia(t, s, "rule34video:dupA", "rule34video", filepath.Join(root, "A", "same.mp4"), true)
	seedCompletedMedia(t, s, "rule34video:dupB", "rule34video", filepath.Join(root, "B", "same.mp4"), true)
	keep := filepath.Join(root, "A", "keep.mp4")
	seedCompletedMedia(t, s, "rule34video:k", "rule34video", keep, true)

	export := vdfGUIExport{Groups: []vdfGUIGroup{{
		GroupID:     "g",
		RemoveItems: []vdfGUIItem{{Path: "/mac/somewhere/same.mp4"}}, // ambiguous basename
		KeepItems:   []vdfGUIItem{vdfItem(keep)},
	}}}
	preview := a.previewVDFGUI(export)
	if preview.ReadyGroups != 0 || preview.Items[0].RemoveItems[0].MatchStatus != "ambiguous" {
		t.Fatalf("ambiguous not detected: %+v", preview.Items[0].RemoveItems[0])
	}

	// Unmanaged file => unmatched.
	export2 := vdfGUIExport{Groups: []vdfGUIGroup{{
		GroupID:     "g2",
		RemoveItems: []vdfGUIItem{{Path: "/outside/not-managed.mp4"}},
		KeepItems:   []vdfGUIItem{vdfItem(keep)},
	}}}
	p2 := a.previewVDFGUI(export2)
	if p2.ReadyGroups != 0 || p2.Items[0].RemoveItems[0].MatchStatus != "unmatched" {
		t.Fatalf("unmatched not detected: %+v", p2.Items[0].RemoveItems[0])
	}
}

func TestVDFGUIKeepMissingSizeChangeCrossSubAndNAS(t *testing.T) {
	root := t.TempDir()
	a, s := newIntegrityApp(t, root, false)
	dup := filepath.Join(root, "A", "dup.mp4")
	keep := filepath.Join(root, "A", "keep.mp4")
	seedCompletedMedia(t, s, "rule34video:d", "rule34video", dup, true)
	seedCompletedMedia(t, s, "rule34video:k", "rule34video", keep, true)
	seedCompletedMedia(t, s, "hanime1:other", "hanime1", filepath.Join(root, "B", "other.mp4"), true)

	// Keep missing (basename does not resolve to any managed media).
	p := a.previewVDFGUI(vdfGUIExport{Groups: []vdfGUIGroup{{GroupID: "g", RemoveItems: []vdfGUIItem{vdfItem(dup)}, KeepItems: []vdfGUIItem{{Path: "/nowhere/keep-missing-xyz.mp4"}}}}})
	if p.ReadyGroups != 0 {
		t.Fatalf("missing keep must be rejected: %+v", p.Items[0])
	}
	// KeepItems absent.
	p = a.previewVDFGUI(vdfGUIExport{Groups: []vdfGUIGroup{{GroupID: "g", RemoveItems: []vdfGUIItem{vdfItem(dup)}}}})
	if p.ReadyGroups != 0 {
		t.Fatalf("absent keep must be rejected: %+v", p.Items[0])
	}
	// Size change.
	bad := vdfItem(dup)
	bad.SizeBytes = 999999
	p = a.previewVDFGUI(vdfGUIExport{Groups: []vdfGUIGroup{{GroupID: "g", RemoveItems: []vdfGUIItem{bad}, KeepItems: []vdfGUIItem{vdfItem(keep)}}}})
	if p.ReadyGroups != 0 || p.Items[0].RemoveItems[0].MatchStatus != "size_mismatch" {
		t.Fatalf("size change not detected: %+v", p.Items[0].RemoveItems[0])
	}
	// Cross-subscription group (keep belongs to hanime1).
	cross := filepath.Join(root, "B", "other.mp4")
	p = a.previewVDFGUI(vdfGUIExport{Groups: []vdfGUIGroup{{GroupID: "g", RemoveItems: []vdfGUIItem{vdfItem(dup)}, KeepItems: []vdfGUIItem{vdfItem(cross)}}}})
	if p.ReadyGroups != 0 {
		t.Fatalf("cross-subscription must be rejected: %+v", p.Items[0])
	}
}

func TestVDFGUIStorageUnavailableAndInvalidJSON(t *testing.T) {
	root := t.TempDir()
	a, s := newIntegrityApp(t, filepath.Join(root, "gone"), false)
	seedCompletedMedia(t, s, "rule34video:d", "rule34video", filepath.Join(root, "A", "dup.mp4"), true)
	seedCompletedMedia(t, s, "rule34video:k", "rule34video", filepath.Join(root, "A", "keep.mp4"), true)
	p := a.previewVDFGUI(vdfGUIExport{Groups: []vdfGUIGroup{{GroupID: "g", RemoveItems: []vdfGUIItem{{Path: "x"}}, KeepItems: []vdfGUIItem{{Path: "y"}}}}})
	if p.StorageAvailable || p.ReadyGroups != 0 {
		t.Fatalf("storage unavailable must block: %+v", p)
	}
	if _, err := parseVDFGUIExport([]byte(`{bad`)); err == nil {
		t.Fatal("invalid JSON must error")
	}
	if _, err := parseVDFGUIExport([]byte(`{"CreatedAt":"x"}`)); err == nil {
		t.Fatal("missing Groups must error")
	}
}

func TestVDFGUIBasenameOnlyIsNotExecutable(t *testing.T) {
	root := t.TempDir()
	a, s := newIntegrityApp(t, root, false)
	dup := filepath.Join(root, "Show", "Season 01", "clip.mp4")
	keep := filepath.Join(root, "Show", "Season 01", "keep.mp4")
	seedCompletedMedia(t, s, "rule34video:d", "rule34video", dup, true)
	seedCompletedMedia(t, s, "rule34video:k", "rule34video", keep, true)
	// VDF path shares only the basename (no two-component suffix match).
	export := vdfGUIExport{Groups: []vdfGUIGroup{{
		GroupID:     "g",
		RemoveItems: []vdfGUIItem{{Path: "/mac/Elsewhere/clip.mp4", SizeBytes: int64(len("payload"))}},
		KeepItems:   []vdfGUIItem{{Path: keep, SizeBytes: int64(len("payload"))}},
	}}}
	preview := a.previewVDFGUI(export)
	if preview.ReadyGroups != 0 || preview.Items[0].RemoveItems[0].MatchStatus != "basename_only" {
		t.Fatalf("basename-only must not be executable: %+v", preview.Items[0])
	}
}

func TestVDFGUIAlreadyDeletedExcludesOnly(t *testing.T) {
	root := t.TempDir()
	a, s := newIntegrityApp(t, root, false)
	dup := filepath.Join(root, "A", "dup.mp4")
	keep := filepath.Join(root, "A", "keep.mp4")
	// Remove file already gone (created=false); keep present.
	seedCompletedMedia(t, s, "rule34video:d", "rule34video", dup, false)
	seedCompletedMedia(t, s, "rule34video:k", "rule34video", keep, true)
	export := vdfGUIExport{CreatedAt: "2026-04-04T00:00:00Z", Groups: []vdfGUIGroup{{
		GroupID: "g", RemoveItems: []vdfGUIItem{{Path: dup, SizeBytes: 7}}, KeepItems: []vdfGUIItem{vdfItem(keep)},
	}}}
	preview := a.previewVDFGUI(export)
	if preview.ReadyGroups != 1 || preview.Items[0].RemoveItems[0].MatchStatus != "absent" {
		t.Fatalf("already-deleted remove item must be ready: %+v", preview.Items[0])
	}
	// Dry run does nothing.
	if _, err := a.executeVDFGUIGroups(export, false); err != nil {
		t.Fatal(err)
	}
	if s.IsMediaExcluded("rule34video:d") {
		t.Fatal("dry run must not exclude")
	}
	res, err := a.executeVDFGUIGroups(export, true)
	if err != nil || res.Executed != 1 {
		t.Fatalf("execute: %+v err=%v", res, err)
	}
	if v, _ := s.Video("rule34video:d"); v.Status != "removed" || !s.IsMediaExcluded("rule34video:d") {
		t.Fatalf("already-deleted item not excluded: %+v", v)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatal("kept file must remain")
	}
	if _, err := os.Stat(quarantineTarget(root, vdfPlanID(export), dup)); !os.IsNotExist(err) {
		t.Fatal("nothing should be quarantined for an already-deleted file")
	}
}

func TestVDFGUIExecuteQuarantinesAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	a, s := newIntegrityApp(t, root, false)
	dup := filepath.Join(root, "A", "dup.mp4")
	keep := filepath.Join(root, "A", "keep.mp4")
	seedCompletedMedia(t, s, "rule34video:d", "rule34video", dup, true)
	seedCompletedMedia(t, s, "rule34video:k", "rule34video", keep, true)
	export := vdfGUIExport{CreatedAt: "2026-03-03T00:00:00Z", Groups: []vdfGUIGroup{{
		GroupID: "g", RemoveItems: []vdfGUIItem{vdfItem(dup)}, KeepItems: []vdfGUIItem{vdfItem(keep)},
	}}}

	if _, err := a.executeVDFGUIGroups(export, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dup); err != nil {
		t.Fatal("dry run moved the file")
	}

	res, err := a.executeVDFGUIGroups(export, true)
	if err != nil || res.Executed != 1 {
		t.Fatalf("execute: %+v err=%v", res, err)
	}
	if _, err := os.Stat(dup); !os.IsNotExist(err) {
		t.Fatal("file not quarantined")
	}
	if !s.IsMediaExcluded("rule34video:d") {
		t.Fatal("video not excluded")
	}
	if v, _ := s.Video("rule34video:d"); v.Status != "removed" {
		t.Fatalf("video not removed: %+v", v)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatal("kept file must remain")
	}
	// Idempotent second execute: already excluded => not a ready group.
	res2, _ := a.executeVDFGUIGroups(export, true)
	if res2.Executed != 0 {
		t.Fatalf("second execute not idempotent: %+v", res2)
	}
}
