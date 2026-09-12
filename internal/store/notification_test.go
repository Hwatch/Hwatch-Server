package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNotificationCooldownAndRead(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "notifications.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	created, err := s.Notify("storage_low", "error", "low", "media", time.Hour)
	if err != nil || !created {
		t.Fatalf("first notify created=%v err=%v", created, err)
	}
	created, err = s.Notify("storage_low", "error", "low", "media", time.Hour)
	if err != nil || created {
		t.Fatalf("duplicate notify created=%v err=%v", created, err)
	}
	if unread, _ := s.UnreadNotifications(); unread != 1 {
		t.Fatalf("unread=%d", unread)
	}
	if err = s.MarkNotificationsRead(); err != nil {
		t.Fatal(err)
	}
	if unread, _ := s.UnreadNotifications(); unread != 0 {
		t.Fatalf("unread after read=%d", unread)
	}
}
