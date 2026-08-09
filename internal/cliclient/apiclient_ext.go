package cliclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
)

// Stats mirrors internal/api's /api/v1/stats response shape.
type Stats struct {
	TotalMessages    int     `json:"total_messages"`
	TotalSizeBytes   int64   `json:"total_size_bytes"`
	OldestReceivedAt *string `json:"oldest_received_at"`
	NewestReceivedAt *string `json:"newest_received_at"`
}

// Health mirrors internal/api's /api/v1/health response shape. It is
// returned with HTTP 200 when "ok" and HTTP 503 when "degraded" — both are
// valid, decodable bodies, not transport errors.
type Health struct {
	Status string `json:"status"`
	DB     string `json:"db"`
	SMTP   string `json:"smtp"`
}

// VersionInfo mirrors internal/api's /api/v1/version response shape
// (internal/version.Info).
type VersionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Go      string `json:"go"`
}

// listQuery builds the query params List() and BulkExport() share, from
// ListParams.
func listQuery(p ListParams) url.Values {
	q := url.Values{}
	setIfNotEmpty(q, "q", p.Query)
	setIfNotEmpty(q, "from", p.From)
	setIfNotEmpty(q, "to", p.To)
	setIfNotEmpty(q, "subject", p.Subject)
	setIfNotEmpty(q, "since", p.Since)
	setIfNotEmpty(q, "until", p.Until)
	setIfNotEmpty(q, "sort", p.Sort)
	if p.Limit > 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Offset > 0 {
		q.Set("offset", strconv.Itoa(p.Offset))
	}
	return q
}

// Stats calls GET /api/v1/stats.
func (c *Client) Stats(ctx context.Context) (*Stats, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/stats", nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, asHTTPError(resp)
	}
	defer resp.Body.Close()

	var out Stats
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode stats response: %w", err)
	}
	return &out, nil
}

// Health calls GET /api/v1/health. Unlike other methods, a non-2xx status
// (503 when degraded) is an expected, decodable response, not a transport
// or API error — so the body is decoded regardless of status code, and an
// error is only returned for genuine transport failures or a malformed
// body.
func (c *Client) Health(ctx context.Context) (*Health, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/health", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out Health
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode health response: %w", err)
	}
	return &out, nil
}

// Version calls GET /api/v1/version.
func (c *Client) Version(ctx context.Context) (*VersionInfo, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/version", nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, asHTTPError(resp)
	}
	defer resp.Body.Close()

	var out VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode version response: %w", err)
	}
	return &out, nil
}

// Attachment calls GET /api/v1/messages/{id}/attachments/{attId} and
// returns the raw attachment bytes along with its content type (from the
// Content-Type header) and filename (parsed from the Content-Disposition
// header, falling back to attID if absent or unparseable).
func (c *Client) Attachment(ctx context.Context, msgID, attID string) ([]byte, string, string, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/messages/"+url.PathEscape(msgID)+"/attachments/"+url.PathEscape(attID), nil)
	if err != nil {
		return nil, "", "", err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", "", asHTTPError(resp)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", "", fmt.Errorf("read attachment body: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")

	filename := attID
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			if fn, ok := params["filename"]; ok && fn != "" {
				filename = fn
			}
		}
	}

	return data, contentType, filename, nil
}

// BulkExport calls GET /api/v1/messages/export with the same query params
// List() builds from ListParams, and returns the raw .zip bytes.
func (c *Client) BulkExport(ctx context.Context, p ListParams) ([]byte, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/messages/export", listQuery(p))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, asHTTPError(resp)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
