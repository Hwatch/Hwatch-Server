package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"r34watch/internal/store"
)

func TestSecureMediaRelativeRejectsEscapeAndSymlink(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "video.mp4")
	if err := os.WriteFile(inside, []byte("video"), 0600); err != nil {
		t.Fatal(err)
	}
	rel, _, err := secureMediaRelative(root, inside)
	if err != nil || rel != "video.mp4" {
		t.Fatalf("rel=%q err=%v", rel, err)
	}
	outside := filepath.Join(t.TempDir(), "outside.mp4")
	if err = os.WriteFile(outside, []byte("video"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err = secureMediaRelative(root, outside); err == nil {
		t.Fatal("outside path accepted")
	}
	link := filepath.Join(root, "link.mp4")
	if err = os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if _, _, err = secureMediaRelative(root, link); err == nil {
		t.Fatal("escaping symlink accepted")
	}
}

func TestNodeAPIUsesBearerInsteadOfAdminSession(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "node.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	token := "test-fixture-token"
	if err = s.CreateComputeNode("node_test", "test", tokenHash(token), `{"dedupe":true,"transcode":false}`); err != nil {
		t.Fatal(err)
	}
	a := New(s)
	handler := a.router(t.TempDir())
	body := []byte(`{"capabilities":{"dedupe":true,"transcode":false},"version":"test","platform":"linux"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/node/v1/heartbeat", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("heartbeat status=%d body=%s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodPost, "/api/node/v1/heartbeat", bytes.NewReader(body))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous status=%d body=%s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet, "/api/nodes", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("admin endpoint status=%d", response.Code)
	}
}
