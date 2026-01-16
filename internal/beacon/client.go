package beacon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/pkg/backoff"
	"golang.org/x/time/rate"
)

// Client is an HTTP client for the Beacon Node API.
type Client struct {
	baseURL    string
	httpClient *http.Client
	limiter    *rate.Limiter
	maxRetries int
}

// NewClient creates a new Beacon API client with rate limiting and connection pooling.
func NewClient(cfg *config.Config) *Client {
	transport := &http.Transport{
		MaxIdleConns:        cfg.HTTP.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.HTTP.MaxIdleConns,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   cfg.HTTP.Timeout(),
	}

	limiter := rate.NewLimiter(
		rate.Limit(cfg.RateLimit.RequestsPerSecond),
		cfg.RateLimit.Burst,
	)

	return &Client{
		baseURL:    cfg.BeaconNodeURL,
		httpClient: httpClient,
		limiter:    limiter,
		maxRetries: cfg.ScyllaDB.MaxRetries,
	}
}

// doRequest performs an HTTP request with rate limiting and retries.
func (c *Client) doRequest(ctx context.Context, method, path string, result interface{}) error {
	url := c.baseURL + path

	var lastErr error
	b := backoff.NewDefault()

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Wait for rate limiter
		if err := c.limiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limiter error: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < c.maxRetries {
				log.Warn().Err(err).Str("url", url).Int("attempt", attempt+1).Msg("Request failed, retrying")
				if !b.Wait(ctx) {
					return ctx.Err()
				}
				continue
			}
			return fmt.Errorf("request failed after %d attempts: %w", attempt+1, err)
		}

		defer resp.Body.Close()

		// Check for retryable errors
		if backoff.ShouldRetry(resp.StatusCode) {
			body, _ := io.ReadAll(resp.Body)
			lastErr = &backoff.RetryableError{
				StatusCode: resp.StatusCode,
				Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
			}

			log.Warn().
				Int("status", resp.StatusCode).
				Str("url", url).
				Int("attempt", attempt+1).
				Msg("Retryable error, backing off")

			if attempt < c.maxRetries {
				if !b.Wait(ctx) {
					return ctx.Err()
				}
				continue
			}
			return lastErr
		}

		// Check for other errors
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
		}

		// Decode response
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}

		return nil
	}

	return lastErr
}

// get performs a GET request.
func (c *Client) get(ctx context.Context, path string, result interface{}) error {
	return c.doRequest(ctx, http.MethodGet, path, result)
}

// post performs a POST request with a JSON body.
func (c *Client) post(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.doRequestWithBody(ctx, http.MethodPost, path, body, result)
}

// doRequestWithBody performs an HTTP request with a JSON body.
func (c *Client) doRequestWithBody(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	url := c.baseURL + path

	// Wait for rate limiter
	if err := c.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter error: %w", err)
	}

	var bodyReader io.Reader
	if body != nil {
		pr, pw := io.Pipe()
		go func() {
			err := json.NewEncoder(pw).Encode(body)
			pw.CloseWithError(err)
		}()
		bodyReader = pr
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// Close releases resources held by the client.
func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}
