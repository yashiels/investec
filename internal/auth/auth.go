package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yashiels/investec/internal/config"
)

// expirySafetyMargin is subtracted from the token lifetime so a token about to
// expire is refreshed before a request can race the expiry.
const expirySafetyMargin = 60 * time.Second

// Token is the cached bearer token plus metadata.
type Token struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	Scope       string    `json:"scope"`
	ExpiresAt   time.Time `json:"expires_at"`
	TokenURL    string    `json:"token_url"`
}

// Valid reports whether the token is present and not within the safety margin.
func (t *Token) Valid(now time.Time) bool {
	return t != nil && t.AccessToken != "" && now.Add(expirySafetyMargin).Before(t.ExpiresAt)
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// Client mints and caches tokens for one credential set + environment.
type Client struct {
	cfg  *config.Config
	http *http.Client
	now  func() time.Time
}

func New(cfg *config.Config, httpClient *http.Client) *Client {
	return &Client{cfg: cfg, http: httpClient, now: time.Now}
}

func (c *Client) tokenURL() string {
	return c.cfg.Base + "/identity/v2/oauth2/token"
}

// cacheKey partitions the cache by environment (token URL) and client_id, and
// by a hash of the api key so a rotated key never reuses a stale token. The api
// key itself is never written to disk.
func (c *Client) cacheKey() string {
	h := sha256.New()
	io.WriteString(h, c.tokenURL())
	io.WriteString(h, "\x00")
	io.WriteString(h, c.cfg.Creds.ClientID)
	io.WriteString(h, "\x00")
	io.WriteString(h, c.cfg.Creds.APIKey)
	return hex.EncodeToString(h.Sum(nil))[:24]
}

func cacheDir() string {
	if x := os.Getenv("XDG_CACHE_HOME"); x != "" {
		return filepath.Join(x, "investec")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "investec")
}

func (c *Client) cachePath() string {
	return filepath.Join(cacheDir(), "token-"+c.cacheKey()+".json")
}

// Get returns a valid token, minting a fresh one only when the cache is empty
// or stale.
func (c *Client) Get(ctx context.Context) (*Token, error) {
	if t := c.readCache(); t.Valid(c.now()) {
		return t, nil
	}
	return c.Refresh(ctx)
}

// Refresh always mints a new token and replaces the cache.
func (c *Client) Refresh(ctx context.Context) (*Token, error) {
	tr, err := c.mint(ctx)
	if err != nil {
		return nil, err
	}
	tok := &Token{
		AccessToken: tr.AccessToken,
		TokenType:   tr.TokenType,
		Scope:       tr.Scope,
		ExpiresAt:   c.now().Add(time.Duration(tr.ExpiresIn) * time.Second),
		TokenURL:    c.tokenURL(),
	}
	c.writeCache(tok)
	return tok, nil
}

func (c *Client) mint(ctx context.Context) (*tokenResponse, error) {
	form := url.Values{"grant_type": {"client_credentials"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL(),
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	basic := base64.StdEncoding.EncodeToString(
		[]byte(c.cfg.Creds.ClientID + ":" + c.cfg.Creds.ClientSecret))
	req.Header.Set("Authorization", "Basic "+basic)
	req.Header.Set("x-api-key", c.cfg.Creds.APIKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode != http.StatusOK {
		return nil, &AuthError{Status: resp.StatusCode, Body: string(body)}
	}
	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("token response parse failed: %w", err)
	}
	if tr.AccessToken == "" {
		return nil, &AuthError{Status: resp.StatusCode, Body: "empty access_token in response"}
	}
	return &tr, nil
}

func (c *Client) readCache() *Token {
	b, err := os.ReadFile(c.cachePath())
	if err != nil {
		return nil
	}
	var t Token
	if json.Unmarshal(b, &t) != nil {
		return nil
	}
	return &t
}

// writeCache persists the token atomically with restrictive permissions. Cache
// failures are non-fatal — the caller still holds a usable token.
func (c *Client) writeCache(t *Token) {
	dir := cacheDir()
	if os.MkdirAll(dir, 0o700) != nil {
		return
	}
	b, err := json.Marshal(t)
	if err != nil {
		return
	}
	tmp, err := os.CreateTemp(dir, "token-*.tmp")
	if err != nil {
		return
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return
	}
	if err := tmp.Close(); err != nil {
		return
	}
	os.Rename(tmpName, c.cachePath())
}

// AuthError is a failed token mint (exit code 3 territory).
type AuthError struct {
	Status int
	Body   string
}

func (e *AuthError) Error() string {
	msg := strings.TrimSpace(e.Body)
	if msg == "" {
		msg = http.StatusText(e.Status)
	}
	return fmt.Sprintf("token mint failed (HTTP %d): %s", e.Status, msg)
}
