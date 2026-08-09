package cliclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// MessageSummary mirrors internal/api's messageSummary JSON shape
// (SPEC.md §5.1). Duplicated here rather than imported, since /internal/api
// is not meant to be depended on as a library (SPEC.md §2.3 point 3).
type MessageSummary struct {
	ID              string   `json:"id"`
	From            string   `json:"from"`
	To              []string `json:"to"`
	Cc              []string `json:"cc"`
	Subject         string   `json:"subject"`
	SizeBytes       int64    `json:"size_bytes"`
	HasAttachments  bool     `json:"has_attachments"`
	AttachmentCount int      `json:"attachment_count"`
	ReceivedAt      string   `json:"received_at"`
	ParseWarning    bool     `json:"parse_warning"`
	Read            bool     `json:"read"`
}

// Header is a single message header (name/value).
type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// AttachmentInfo describes one attachment/inline image in MessageDetail.
type AttachmentInfo struct {
	ID          string  `json:"id"`
	Filename    string  `json:"filename"`
	ContentType string  `json:"content_type"`
	SizeBytes   int64   `json:"size_bytes"`
	ContentID   *string `json:"content_id"`
}

// MessageDetail mirrors internal/api's messageDetail JSON shape.
type MessageDetail struct {
	MessageSummary
	Headers      []Header         `json:"headers"`
	TextBody     string           `json:"text_body"`
	HTMLBody     string           `json:"html_body"`
	Attachments  []AttachmentInfo `json:"attachments"`
	RawSizeBytes int64            `json:"raw_size_bytes"`
}

// ListResponse mirrors internal/api's listResponse envelope.
type ListResponse struct {
	Messages []MessageSummary `json:"messages"`
	Total    int              `json:"total"`
	Limit    int              `json:"limit"`
	Offset   int              `json:"offset"`
}

// HTTPError is returned when the API responds with a non-2xx status,
// distinguishing "server reachable but rejected the request" from a
// transport-level failure (connection refused, DNS, timeout, ...).
type HTTPError struct {
	Status  int
	Code    string
	Message string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("API returned %d %s: %s", e.Status, e.Code, e.Message)
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// ListParams controls GET /api/v1/messages query params (SPEC.md §5.2).
type ListParams struct {
	Query   string
	From    string
	To      string
	Subject string
	Limit   int
	Offset  int
	Since   string
	Until   string
	Sort    string
}

// Client is a thin REST client for maelsink's /api/v1 surface.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient builds a Client with a sane default *http.Client.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{BaseURL: baseURL, APIKey: apiKey, HTTPClient: http.DefaultClient}
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values) (*http.Response, error) {
	u := c.BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		return nil, err
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach maelsink API at %s: %w", c.BaseURL, err)
	}
	return resp, nil
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// asHTTPError reads and classifies a non-2xx response, returning an
// *HTTPError. The caller is responsible for closing resp.Body beforehand is
// not required; this consumes it.
func asHTTPError(resp *http.Response) error {
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var env errorEnvelope
	if err := json.Unmarshal(body, &env); err == nil && env.Error.Code != "" {
		return &HTTPError{Status: resp.StatusCode, Code: env.Error.Code, Message: env.Error.Message}
	}
	return &HTTPError{Status: resp.StatusCode, Code: "unknown_error", Message: string(body)}
}

// List calls GET /api/v1/messages.
func (c *Client) List(ctx context.Context, p ListParams) (*ListResponse, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/messages", listQuery(p))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, asHTTPError(resp)
	}
	defer resp.Body.Close()

	var out ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode list response: %w", err)
	}
	return &out, nil
}

// Get calls GET /api/v1/messages/{id}.
func (c *Client) Get(ctx context.Context, id string) (*MessageDetail, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/messages/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, asHTTPError(resp)
	}
	defer resp.Body.Close()

	var out MessageDetail
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode message detail: %w", err)
	}
	return &out, nil
}

// Delete calls DELETE /api/v1/messages/{id}.
func (c *Client) Delete(ctx context.Context, id string) error {
	resp, err := c.do(ctx, http.MethodDelete, "/api/v1/messages/"+url.PathEscape(id), nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		return asHTTPError(resp)
	}
	resp.Body.Close()
	return nil
}

// Clear calls DELETE /api/v1/messages?confirm=true.
func (c *Client) Clear(ctx context.Context) error {
	resp, err := c.do(ctx, http.MethodDelete, "/api/v1/messages", url.Values{"confirm": {"true"}})
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		return asHTTPError(resp)
	}
	resp.Body.Close()
	return nil
}

// ExportRaw calls GET /api/v1/messages/{id}/export and returns the raw .eml
// bytes.
func (c *Client) ExportRaw(ctx context.Context, id string) ([]byte, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/messages/"+url.PathEscape(id)+"/export", nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, asHTTPError(resp)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func setIfNotEmpty(q url.Values, key, value string) {
	if value != "" {
		q.Set(key, value)
	}
}
