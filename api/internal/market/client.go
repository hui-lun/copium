package market

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

// Hosts that ban aggressive clients; requests to them are throttled harder.
var throttledHosts = map[string]bool{
	"www.twse.com.tw": true,
	"www.tpex.org.tw": true,
}

// Client is a shared HTTP client that enforces a minimum interval between
// requests to the same host.
type Client struct {
	http        *http.Client
	webInterval time.Duration

	mu    sync.Mutex
	gates map[string]*gate
}

type gate struct {
	mu       sync.Mutex
	next     time.Time
	interval time.Duration
}

func NewClient(webInterval time.Duration) *Client {
	return &Client{
		http:        &http.Client{Timeout: 30 * time.Second},
		webInterval: webInterval,
		gates:       map[string]*gate{},
	}
}

func (c *Client) gateFor(host string) *gate {
	c.mu.Lock()
	defer c.mu.Unlock()
	g, ok := c.gates[host]
	if !ok {
		interval := 200 * time.Millisecond
		if throttledHosts[host] {
			interval = c.webInterval
		}
		g = &gate{interval: interval}
		c.gates[host] = g
	}
	return g
}

// wait blocks until this caller's reserved time slot arrives.
func (g *gate) wait(ctx context.Context) error {
	g.mu.Lock()
	start := g.next
	if now := time.Now(); start.Before(now) {
		start = now
	}
	g.next = start.Add(g.interval)
	g.mu.Unlock()

	d := time.Until(start)
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// getJSON performs a throttled GET and decodes the JSON body into out.
func (c *Client) getJSON(ctx context.Context, rawURL string, out any) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if err := c.gateFor(u.Host).wait(ctx); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; copium/1.0)")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: status %d", rawURL, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	// Some endpoints prepend a UTF-8 BOM.
	body = bytes.TrimPrefix(body, []byte("\xef\xbb\xbf"))
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}
