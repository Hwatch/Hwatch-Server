package main

import (
	"errors"
	"testing"

	"r34watch/internal/downloader"
)

func TestClassifyDownloadErrors(t *testing.T) {
	tests := []struct {
		message, code string
		retryable     bool
	}{
		{"HTTP 429", "rate_limited", true},
		{"context deadline exceeded", "timeout", true},
		{"downloader unavailable: aria2", "downloader_offline", true},
		{"aria2 RPC -1: GID was not found", "detached_task", true},
		{"permission denied", "permission_denied", false},
		{"no space left on device", "storage_unavailable", true},
		{"媒体目录不可用或挂载已断开", "storage_unavailable", true},
		{"存储目录未挂载，拒绝在本机文件系统创建目录", "storage_unavailable", true},
		{"HTTP 403", "source_expired", true},
	}
	for _, tt := range tests {
		got := classifyDownloadError(errors.New(tt.message))
		if got.Code != tt.code || got.Retryable != tt.retryable {
			t.Errorf("%q = %+v", tt.message, got)
		}
	}
	got := classifyDownloadError(downloader.ErrCapacityReached)
	if got.Code != "capacity_reached" || !got.Retryable {
		t.Fatalf("capacity = %+v", got)
	}
	if got := classifyDownloaderStatus("22", "aria2 error 22"); got.Code != "source_expired" || !got.Retryable {
		t.Fatalf("aria status = %+v", got)
	}
}

func TestClassifyTransientNetworkFailures(t *testing.T) {
	for _, message := range []string{
		"SSL/TLS handshake failure: The TLS connection was non-proper",
		"No URI available.",
		"Too slow Downloading speed: 0 <= 1024(B/s), host:eu-cdn08-pr",
		"unexpected EOF",
		"connection reset by peer",
	} {
		got := classifyDownloaderStatus("1", message)
		if got.Code != "transient_network" || !got.Retryable {
			t.Errorf("%q = %+v, want transient_network retryable", message, got)
		}
	}
	// aria2 speeds/network codes.
	if got := classifyDownloaderStatus("5", "aria2 error 5"); got.Code != "transient_network" || !got.Retryable {
		t.Fatalf("code 5 = %+v", got)
	}
	if got := classifyDownloaderStatus("6", "aria2 error 6"); got.Code != "transient_network" || !got.Retryable {
		t.Fatalf("code 6 = %+v", got)
	}
	// Resume must never overwrite partial data.
	if got := classifyDownloaderStatus("8", "aria2 error 8"); got.Code != "resume_unsupported" || got.Retryable {
		t.Fatalf("code 8 = %+v", got)
	}
	if got := classifyDownloaderStatus("10", "aria2 error 10"); got.Code != "source_changed" || got.Retryable {
		t.Fatalf("code 10 = %+v", got)
	}
	// 429 keeps its own path.
	if got := classifyDownloaderStatus("1", "The response status is not successful. status=429"); got.Code != "rate_limited" {
		t.Fatalf("429 = %+v", got)
	}
}
