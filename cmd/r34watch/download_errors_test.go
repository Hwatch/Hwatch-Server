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
