package transrunner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func fakeServer(t *testing.T, captured *CreateJobRequest) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok","version":"0.2.1"}`))
	})
	mux.HandleFunc("/api/v1/capabilities", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"UNAUTHORIZED","message":"invalid token"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"ffmpeg":"8.1","videotoolbox":true,"profiles":["hevc-1080p","h264-1080p"]}`))
	})
	mux.HandleFunc("/api/v1/jobs", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"UNAUTHORIZED","message":"invalid token"}}`))
			return
		}
		if captured != nil {
			_ = json.NewDecoder(r.Body).Decode(captured)
		}
		_, _ = w.Write([]byte(`{"id":"job_1","state":"queued","profile":"hevc-1080p"}`))
	})
	return httptest.NewServer(mux)
}

func TestClientTestConnection(t *testing.T) {
	server := fakeServer(t, nil)
	defer server.Close()
	result, err := New(server.URL, "secret-token").TestConnection(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != "0.2.1" {
		t.Fatalf("version=%q", result.Version)
	}
}

func TestClientTestConnectionBadToken(t *testing.T) {
	server := fakeServer(t, nil)
	defer server.Close()
	_, err := New(server.URL, "wrong").TestConnection(context.Background())
	if !IsUnauthorized(err) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestClientNetworkError(t *testing.T) {
	server := fakeServer(t, nil)
	url := server.URL
	server.Close()
	_, err := New(url, "secret-token").TestConnection(context.Background())
	var target *Error
	if err == nil || !asError(err, &target) || target.Kind != ErrNetwork {
		t.Fatalf("expected network error, got %v", err)
	}
}

func TestClientCreateJobUsesHWatchInput(t *testing.T) {
	var captured CreateJobRequest
	server := fakeServer(t, &captured)
	defer server.Close()
	job, err := New(server.URL, "secret-token").CreateJob(context.Background(), CreateJobRequest{
		Input:   HWInput{Type: "hwatch", VideoID: "rule34video:1", MediaRelativePath: "Rule34Video/A/file.mp4"},
		Profile: "hevc-1080p",
	})
	if err != nil || job.ID != "job_1" {
		t.Fatalf("job=%+v err=%v", job, err)
	}
	if captured.Input.Type != "hwatch" || captured.Input.MediaRelativePath == "" || captured.Input.MediaRelativePath[0] == '/' {
		t.Fatalf("payload=%+v", captured)
	}
}

func TestClientServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"boom"}`))
	}))
	defer server.Close()
	_, err := New(server.URL, "secret-token").CreateJob(context.Background(), CreateJobRequest{})
	var target *Error
	if err == nil || !asError(err, &target) || target.Kind != ErrServer {
		t.Fatalf("expected server error, got %v", err)
	}
}

func asError(err error, target **Error) bool {
	for err != nil {
		if e, ok := err.(*Error); ok {
			*target = e
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
