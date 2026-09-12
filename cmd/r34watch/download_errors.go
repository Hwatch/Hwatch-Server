package main

import (
	"errors"
	"net"
	"os"
	"strings"
	"time"

	"r34watch/internal/downloader"
)

type downloadFailure struct {
	Code      string
	Retryable bool
	Delay     time.Duration
}

func classifyDownloadError(err error) downloadFailure {
	if err == nil {
		return downloadFailure{}
	}
	message := strings.ToLower(err.Error())
	switch {
	case errors.Is(err, downloader.ErrCapacityReached):
		return downloadFailure{Code: "capacity_reached", Retryable: true}
	case isRateLimited(err.Error()):
		return downloadFailure{Code: "rate_limited", Retryable: true, Delay: 30 * time.Second}
	case errors.Is(err, os.ErrPermission) || strings.Contains(message, "permission denied"):
		return downloadFailure{Code: "permission_denied"}
	case strings.Contains(message, "no space left") || strings.Contains(message, "低于保留值"):
		return downloadFailure{Code: "storage_unavailable", Retryable: true, Delay: 30 * time.Second}
	case strings.Contains(message, "downloader unavailable") || strings.Contains(message, "connection refused"):
		return downloadFailure{Code: "downloader_offline", Retryable: true}
	case strings.Contains(message, "not attached to this process") || (strings.Contains(message, "gid") && strings.Contains(message, "not found")):
		return downloadFailure{Code: "detached_task", Retryable: true}
	case strings.Contains(message, "status=403") || strings.Contains(message, "http 403") || strings.Contains(message, "status=404") || strings.Contains(message, "http 404"):
		return downloadFailure{Code: "source_expired", Retryable: true, Delay: 30 * time.Second}
	case strings.Contains(message, "timeout") || strings.Contains(message, "deadline exceeded"):
		return downloadFailure{Code: "timeout", Retryable: true, Delay: 10 * time.Second}
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return downloadFailure{Code: "timeout", Retryable: true, Delay: 10 * time.Second}
	}
	return downloadFailure{Code: "download_failed"}
}

func classifyDownloaderStatus(code, message string) downloadFailure {
	failure := classifyDownloadMessage(message)
	if failure.Code != "" && failure.Code != "download_failed" {
		return failure
	}
	switch code {
	case "2":
		return downloadFailure{Code: "timeout", Retryable: true, Delay: 10 * time.Second}
	case "3", "22":
		return downloadFailure{Code: "source_expired", Retryable: true, Delay: 30 * time.Second}
	case "19":
		return downloadFailure{Code: "downloader_offline", Retryable: true, Delay: 30 * time.Second}
	default:
		return downloadFailure{Code: "download_failed"}
	}
}

func classifyDownloadMessage(message string) downloadFailure {
	if strings.TrimSpace(message) == "" {
		return downloadFailure{}
	}
	return classifyDownloadError(errors.New(message))
}
