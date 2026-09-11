package aria2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type Client struct {
	URL, Secret string
	http        *http.Client
	id          atomic.Int64
}
type Status struct {
	GID, State, ErrorCode, ErrorMessage, Dir string
	Completed, Total                         int64
	DownloadSpeed                            int64
	Connections                              int
	Files                                    []string
}

func New(rawURL, secret string) *Client {
	return &Client{URL: rawURL, Secret: secret, http: &http.Client{Timeout: 15 * time.Second}}
}
func (c *Client) call(ctx context.Context, method string, params []any, result any) error {
	if c.URL == "" {
		return errors.New("aria2 RPC address is empty")
	}
	if params == nil {
		params = []any{}
	}
	if c.Secret != "" {
		params = append([]any{"token:" + c.Secret}, params...)
	}
	body := map[string]any{"jsonrpc": "2.0", "id": c.id.Add(1), "method": "aria2." + method, "params": params}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("aria2 RPC unavailable: %w", err)
	}
	defer res.Body.Close()
	var envelope struct {
		Result json.RawMessage
		Error  *struct {
			Code    int
			Message string
		}
	}
	if err = json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		return err
	}
	if envelope.Error != nil {
		return fmt.Errorf("aria2 RPC %d: %s", envelope.Error.Code, envelope.Error.Message)
	}
	if result != nil {
		return json.Unmarshal(envelope.Result, result)
	}
	return nil
}
func (c *Client) Version(ctx context.Context) error {
	var out map[string]any
	return c.call(ctx, "getVersion", nil, &out)
}
func (c *Client) Add(ctx context.Context, uri, dir, name, referer, userAgent, proxy string, connections int) (string, error) {
	opts := map[string]string{"dir": dir, "out": name, "continue": "true", "auto-file-renaming": "false", "allow-overwrite": "true", "max-tries": "12", "retry-wait": "30", "timeout": "60", "connect-timeout": "20", "lowest-speed-limit": "1K", "summary-interval": "1", "header": "Referer: " + referer + "\nUser-Agent: " + userAgent}
	if connections < 1 {
		connections = 1
	}
	if connections > 16 {
		connections = 16
	}
	opts["max-connection-per-server"] = strconv.Itoa(connections)
	opts["split"] = strconv.Itoa(connections)
	opts["min-split-size"] = "1M"
	if proxy != "" {
		opts["all-proxy"] = proxy
	}
	var gid string
	err := c.call(ctx, "addUri", []any{[]string{uri}, opts}, &gid)
	return gid, err
}
func (c *Client) Tell(ctx context.Context, gid string) (Status, error) {
	var raw map[string]any
	fields := []string{"gid", "status", "totalLength", "completedLength", "downloadSpeed", "connections", "errorCode", "errorMessage", "dir", "files"}
	if err := c.call(ctx, "tellStatus", []any{gid, fields}, &raw); err != nil {
		return Status{}, err
	}
	return decode(raw), nil
}
func (c *Client) Pause(ctx context.Context, gid string) error {
	var out string
	return c.call(ctx, "pause", []any{gid}, &out)
}
func (c *Client) Resume(ctx context.Context, gid string) error {
	var out string
	return c.call(ctx, "unpause", []any{gid}, &out)
}
func (c *Client) Remove(ctx context.Context, gid string) error {
	var out string
	err := c.call(ctx, "remove", []any{gid}, &out)
	if err != nil {
		err = c.call(ctx, "removeDownloadResult", []any{gid}, &out)
	}
	return err
}
func (c *Client) ReplaceURI(ctx context.Context, gid, uri string) error {
	var out []int
	return c.call(ctx, "changeUri", []any{gid, []int{1}, []string{}, []string{uri}}, &out)
}
func (c *Client) SetMaxConcurrent(ctx context.Context, value int) error {
	if value < 1 {
		value = 1
	}
	var out string
	return c.call(ctx, "changeGlobalOption", []any{map[string]string{"max-concurrent-downloads": strconv.Itoa(value)}}, &out)
}
func (c *Client) SetGlobal(ctx context.Context, concurrent int, limitKBPS int64) error {
	if concurrent < 1 {
		concurrent = 1
	}
	limit := "0"
	if limitKBPS > 0 {
		limit = strconv.FormatInt(limitKBPS, 10) + "K"
	}
	var out string
	return c.call(ctx, "changeGlobalOption", []any{map[string]string{"max-concurrent-downloads": strconv.Itoa(concurrent), "max-overall-download-limit": limit}}, &out)
}
func decode(raw map[string]any) Status {
	s := Status{GID: text(raw["gid"]), State: text(raw["status"]), ErrorCode: text(raw["errorCode"]), ErrorMessage: text(raw["errorMessage"]), Dir: text(raw["dir"])}
	s.Total = parse(text(raw["totalLength"]))
	s.Completed = parse(text(raw["completedLength"]))
	s.DownloadSpeed = parse(text(raw["downloadSpeed"]))
	connections, _ := strconv.Atoi(text(raw["connections"]))
	s.Connections = connections
	if files, ok := raw["files"].([]any); ok {
		for _, entry := range files {
			if m, ok := entry.(map[string]any); ok {
				s.Files = append(s.Files, text(m["path"]))
			}
		}
	}
	return s
}
func text(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
func parse(v string) int64 { n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64); return n }
