// Package pocketcasts is a minimal client for the unofficial Pocket Casts
// web API at api.pocketcasts.com. The API isn't documented publicly;
// behavior here is derived from observed traffic and prior art.
package pocketcasts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// DefaultBaseURL is the public root of the unofficial Pocket Casts web API.
	DefaultBaseURL = "https://api.pocketcasts.com"

	// defaultTimeout is generous — these endpoints sometimes take a few
	// seconds to return the full history.
	defaultTimeout = 30 * time.Second
)

// Client wraps the Pocket Casts unofficial web API. The zero value is not
// usable; construct one with New.
type Client struct {
	baseURL string
	http    *http.Client
	token   string
}

// Option configures a Client at construction time.
type Option func(*Client)

// WithBaseURL overrides the API root. Primarily for tests.
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = u }
}

// WithHTTPClient overrides the underlying http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// New constructs a Client with sensible defaults.
func New(opts ...Option) *Client {
	c := &Client{
		baseURL: DefaultBaseURL,
		http:    &http.Client{Timeout: defaultTimeout},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Token returns the current bearer token, or "" if none has been set.
func (c *Client) Token() string { return c.token }

// SetToken installs a previously-obtained bearer token. Intended for
// callers that cache tokens across runs.
func (c *Client) SetToken(t string) { c.token = t }

// Episode is the API representation of a podcast episode. Fields not
// present in a given response decode to zero values.
type Episode struct {
	UUID         string `json:"uuid"`
	URL          string `json:"url"`
	Title        string `json:"title"`
	PodcastTitle string `json:"podcastTitle"`
	PodcastUUID  string `json:"podcastUuid"`
	Published    string `json:"published"`
	PlayedUpTo   int    `json:"playedUpTo"`
	Duration     int    `json:"duration"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Scope    string `json:"scope"`
}

type loginResponse struct {
	Token string `json:"token"`
}

type episodeListResponse struct {
	Episodes []Episode `json:"episodes"`
}

// Login exchanges email/password for a bearer token and stores it on the
// client. The token is also returned for callers that want to cache it.
func (c *Client) Login(ctx context.Context, email, password string) (string, error) {
	var out loginResponse
	body := loginRequest{Email: email, Password: password, Scope: "webplayer"}
	if err := c.do(ctx, "login", "/user/login", body, false, &out); err != nil {
		return "", err
	}
	if out.Token == "" {
		return "", fmt.Errorf("pocketcasts login: response missing token")
	}
	c.token = out.Token
	return out.Token, nil
}

// History returns recent listening history. The exact size of the window
// is determined by the server.
func (c *Client) History(ctx context.Context) ([]Episode, error) {
	var out episodeListResponse
	if err := c.do(ctx, "history", "/user/history", struct{}{}, true, &out); err != nil {
		return nil, err
	}
	return out.Episodes, nil
}

// Starred returns episodes the user has starred. The endpoint path is
// currently a guess based on convention; Phase 4 will verify and adjust.
func (c *Client) Starred(ctx context.Context) ([]Episode, error) {
	var out episodeListResponse
	if err := c.do(ctx, "starred", "/user/starred", struct{}{}, true, &out); err != nil {
		return nil, err
	}
	return out.Episodes, nil
}

// do performs a POST to baseURL+path with body marshaled as JSON and
// decodes the response into out. When authed is true and no token is set,
// it returns ErrNotAuthenticated without making a request.
func (c *Client) do(ctx context.Context, op, path string, body any, authed bool, out any) error {
	if authed && c.token == "" {
		return ErrNotAuthenticated
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("pocketcasts %s: marshal: %w", op, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("pocketcasts %s: new request: %w", op, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if authed {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("pocketcasts %s: %w", op, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Capture up to 4KiB of response body for diagnostics.
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return &APIError{Op: op, StatusCode: resp.StatusCode, Body: string(snippet)}
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("pocketcasts %s: decode: %w", op, err)
	}
	return nil
}
