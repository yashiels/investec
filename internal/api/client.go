package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/yashiels/investec/internal/auth"
	"github.com/yashiels/investec/internal/config"
)

// Client talks to the Investec Private Bank API. It owns token acquisition and
// maps transport/HTTP failures onto CLI exit codes.
type Client struct {
	cfg  *config.Config
	http *http.Client
	auth *auth.Client
}

func New(cfg *config.Config, timeout time.Duration) *Client {
	hc := &http.Client{Timeout: timeout}
	return &Client{cfg: cfg, http: hc, auth: auth.New(cfg, hc)}
}

// Auth exposes the token client for `auth token` / `auth status`.
func (c *Client) Auth() *auth.Client { return c.auth }

// Raw is a decoded-yet-verbatim API envelope. The API wraps payloads as
// {data, links, meta}; we preserve all three.
type Raw struct {
	Data  json.RawMessage `json:"data"`
	Links json.RawMessage `json:"links"`
	Meta  json.RawMessage `json:"meta"`
	// Full is the untouched response body for --json output.
	Full json.RawMessage `json:"-"`
}

// get performs an authenticated GET and returns the parsed envelope. On a 401
// it invalidates the token and retries exactly once.
func (c *Client) get(ctx context.Context, path string, query url.Values) (*Raw, error) {
	raw, err := c.doGet(ctx, path, query, false)
	if ae, ok := err.(*CodedError); ok && ae.Code == ExitAuth && ae.retryable {
		return c.doGet(ctx, path, query, true)
	}
	return raw, err
}

func (c *Client) doGet(ctx context.Context, path string, query url.Values, forceRefresh bool) (*Raw, error) {
	var (
		tok *auth.Token
		err error
	)
	if forceRefresh {
		tok, err = c.auth.Refresh(ctx)
	} else {
		tok, err = c.auth.Get(ctx)
	}
	if err != nil {
		return nil, authErr(err)
	}

	u := c.cfg.Base + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, coded(ExitFailure, "%v", err)
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, coded(ExitUpstream, "request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<20))

	if resp.StatusCode/100 != 2 {
		return nil, httpErr(resp, body)
	}
	var r Raw
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, coded(ExitFailure, "response parse failed: %v", err)
	}
	r.Full = body
	return &r, nil
}

// getBinary streams a binary document body, validating status + content type.
func (c *Client) getBinary(ctx context.Context, path, wantType string) ([]byte, string, error) {
	tok, err := c.auth.Get(ctx)
	if err != nil {
		return nil, "", authErr(err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.Base+path, nil)
	if err != nil {
		return nil, "", coded(ExitFailure, "%v", err)
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("Accept", "*/*")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", coded(ExitUpstream, "request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if resp.StatusCode/100 != 2 {
		return nil, "", httpErr(resp, body)
	}
	ct := resp.Header.Get("Content-Type")
	if wantType != "" && !strings.Contains(ct, wantType) {
		return nil, "", coded(ExitFailure,
			"unexpected content-type %q (wanted %q); refusing to write", ct, wantType)
	}
	return body, ct, nil
}

func authErr(err error) error {
	if ae, ok := err.(*auth.AuthError); ok {
		e := coded(ExitAuth, "%s", ae.Error())
		e.retryable = false
		return e
	}
	return coded(ExitUpstream, "%v", err)
}

// httpErr maps an HTTP status onto the exit-code taxonomy.
func httpErr(resp *http.Response, body []byte) *CodedError {
	msg := apiMessage(body)
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		e := coded(ExitAuth, "auth rejected (HTTP %d): %s", resp.StatusCode, msg)
		e.retryable = resp.StatusCode == http.StatusUnauthorized
		return e
	case http.StatusNotFound:
		return coded(ExitNotFound, "not found (HTTP 404): %s", msg)
	case http.StatusTooManyRequests:
		e := coded(ExitRateLimit, "rate limited (HTTP 429): %s", msg)
		e.Retry = resp.Header.Get("Retry-After")
		return e
	}
	if resp.StatusCode/100 == 5 {
		return coded(ExitUpstream, "upstream error (HTTP %d): %s", resp.StatusCode, msg)
	}
	// 400 and any other 4xx are API-rejected requests.
	return coded(ExitFailure, "request rejected (HTTP %d): %s", resp.StatusCode, msg)
}

// apiMessage best-effort extracts a human message from an error body.
func apiMessage(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "(no response body)"
	}
	var probe struct {
		Message     string `json:"message"`
		Description string `json:"description"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if json.Unmarshal(body, &probe) == nil {
		for _, s := range []string{probe.ErrorDesc, probe.Description, probe.Message, probe.Error} {
			if s != "" {
				return s
			}
		}
	}
	if len(trimmed) > 400 {
		trimmed = trimmed[:400] + "…"
	}
	return trimmed
}
