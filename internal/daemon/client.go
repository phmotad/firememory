package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/phmotad/firememory/internal/firequery/contract"
)

// Client connects to a running daemon over localhost TCP.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(port int) *Client {
	if port == 0 {
		port = DefaultPort
	}
	return &Client{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Ping returns nil if the daemon is reachable and healthy.
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/ping", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("daemon: ping returned %d", resp.StatusCode)
	}
	return nil
}

// Request sends an ExternalRequest to the daemon and returns the response.
// For write operations the daemon returns 202 Accepted with status="queued".
func (c *Client) Request(ctx context.Context, req contract.ExternalRequest) (contract.ExternalResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return contract.ExternalResponse{}, fmt.Errorf("daemon client: encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/request", bytes.NewReader(body))
	if err != nil {
		return contract.ExternalResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return contract.ExternalResponse{}, fmt.Errorf("daemon client: request: %w", err)
	}
	defer httpResp.Body.Close()

	var resp contract.ExternalResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return contract.ExternalResponse{}, fmt.Errorf("daemon client: decode response: %w", err)
	}
	return resp, nil
}
