package onemap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const baseURL = "https://www.onemap.gov.sg/api"

type Config struct {
	Email    string
	Password string
}

// Client is a thread-safe OneMap API client that handles token refresh.
type Client struct {
	cfg       Config
	mu        sync.Mutex
	token     string
	expiresAt time.Time
	http      *http.Client
}

func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// get makes an authenticated GET request and decodes the JSON response into dst.
func (c *Client) get(ctx context.Context, path string, params url.Values, dst any) error {
	url := baseURL + path + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	return c.do(ctx, req, dst)
}

// post makes an authenticated POST request with a JSON body and decodes the response into dst.
func (c *Client) post(ctx context.Context, path string, body any, dst any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(ctx, req, dst)
}

// do attaches the auth token, executes the request, and decodes the response.
func (c *Client) do(ctx context.Context, req *http.Request, dst any) error {
	token, err := c.getToken(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", token)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("onemap %s %s: status %d: %s", req.Method, req.URL.Path, resp.StatusCode, body)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

// --- auth ---

type tokenResponse struct {
	AccessToken     string      `json:"access_token"`
	ExpiryTimestamp json.Number `json:"expiry_timestamp"`
}

func (c *Client) getToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.expiresAt.Add(-5*time.Minute)) {
		return c.token, nil
	}
	return c.refreshToken(ctx)
}

func (c *Client) refreshToken(ctx context.Context) (string, error) {
	body := map[string]string{"email": c.cfg.Email, "password": c.cfg.Password}
	b, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/auth/post/getToken", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("onemap auth: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("onemap auth: status %d", resp.StatusCode)
	}

	var tok tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", fmt.Errorf("onemap auth: %w", err)
	}

	expiry, err := tok.ExpiryTimestamp.Int64()
	if err != nil {
		return "", fmt.Errorf("onemap auth: bad expiry: %w", err)
	}

	c.token = tok.AccessToken
	c.expiresAt = time.Unix(expiry, 0)
	return c.token, nil
}


