// Package transrunner is a small HTTP client for the external TransRunner
// computation service. HWatch submits work to TransRunner's own task system; it
// never mirrors TransRunner's queue, progress, leases or runner state.
package transrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		token:   strings.TrimSpace(token),
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

type Health struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type TestResult struct {
	Version      string          `json:"version"`
	Platform     string          `json:"platform,omitempty"`
	Capabilities json.RawMessage `json:"capabilities,omitempty"`
}

// HWInput carries an HWatch-relative media reference. HWatch never sends its own
// absolute media path; TransRunner maps the relative path with its own media
// root.
type HWInput struct {
	Type              string `json:"type"`
	VideoID           string `json:"video_id"`
	MediaRelativePath string `json:"media_relative_path"`
}

type CreateJobRequest struct {
	Kind     string  `json:"kind,omitempty"`
	Input    HWInput `json:"input"`
	Profile  string  `json:"profile,omitempty"`
	Priority string  `json:"priority,omitempty"`
}

type Job struct {
	ID      string `json:"id"`
	State   string `json:"state"`
	Profile string `json:"profile"`
}

type ErrorKind int

const (
	ErrNetwork ErrorKind = iota
	ErrUnauthorized
	ErrForbidden
	ErrBadRequest
	ErrServer
)

type Error struct {
	Kind    ErrorKind
	Status  int
	Message string
}

func (e *Error) Error() string { return e.Message }

func (c *Client) configured() bool {
	return c.baseURL != ""
}

func (c *Client) do(ctx context.Context, method, path string, body any) ([]byte, error) {
	if !c.configured() {
		return nil, &Error{Kind: ErrNetwork, Message: "尚未配置 TransRunner 地址"}
	}
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, &Error{Kind: ErrBadRequest, Message: err.Error()}
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, &Error{Kind: ErrNetwork, Message: "无法构造 TransRunner 请求"}
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, &Error{Kind: ErrNetwork, Message: "无法连接 TransRunner"}
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return data, nil
	}
	message := extractErrorMessage(data, res.StatusCode)
	switch {
	case res.StatusCode == http.StatusUnauthorized:
		return nil, &Error{Kind: ErrUnauthorized, Status: res.StatusCode, Message: "TransRunner API Token 无效"}
	case res.StatusCode == http.StatusForbidden:
		return nil, &Error{Kind: ErrForbidden, Status: res.StatusCode, Message: "TransRunner 拒绝该请求"}
	case res.StatusCode >= 500:
		return nil, &Error{Kind: ErrServer, Status: res.StatusCode, Message: message}
	default:
		return nil, &Error{Kind: ErrBadRequest, Status: res.StatusCode, Message: message}
	}
}

func extractErrorMessage(data []byte, status int) string {
	var env struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(data, &env) == nil {
		if env.Error.Message != "" {
			return env.Error.Message
		}
		if env.Message != "" {
			return env.Message
		}
	}
	if trimmed := strings.TrimSpace(string(data)); trimmed != "" {
		if len(trimmed) > 300 {
			trimmed = trimmed[:300]
		}
		return trimmed
	}
	return fmt.Sprintf("TransRunner 返回 %d", status)
}

// TestConnection verifies the URL is reachable (/health) and the token is valid
// (/api/v1/capabilities requires auth).
func (c *Client) TestConnection(ctx context.Context) (TestResult, error) {
	healthData, err := c.do(ctx, http.MethodGet, "/health", nil)
	if err != nil {
		return TestResult{}, err
	}
	var health Health
	_ = json.Unmarshal(healthData, &health)
	capsData, err := c.do(ctx, http.MethodGet, "/api/v1/capabilities", nil)
	if err != nil {
		return TestResult{}, err
	}
	result := TestResult{Version: health.Version, Capabilities: json.RawMessage(capsData)}
	var caps map[string]any
	if json.Unmarshal(capsData, &caps) == nil {
		if platform, ok := caps["platform"].(string); ok {
			result.Platform = platform
		}
	}
	return result, nil
}

// CreateJob submits a transcode job using an HWatch-relative input.
func (c *Client) CreateJob(ctx context.Context, req CreateJobRequest) (Job, error) {
	data, err := c.do(ctx, http.MethodPost, "/api/v1/jobs", req)
	if err != nil {
		return Job{}, err
	}
	var job Job
	if err := json.Unmarshal(data, &job); err != nil || job.ID == "" {
		return Job{}, &Error{Kind: ErrServer, Message: "TransRunner 未返回任务 ID"}
	}
	return job, nil
}

func IsUnauthorized(err error) bool {
	var target *Error
	return errors.As(err, &target) && target.Kind == ErrUnauthorized
}
