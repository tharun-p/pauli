package beacon

import (
	"bytes"
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Client is an HTTP client for the Beacon Node API.
type Client struct {
	baseURL    string
	apiKey     string
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
		apiKey:     cfg.BeaconAPIKey,
		httpClient: httpClient,
		limiter:    limiter,
		maxRetries: cfg.HTTP.MaxRetries,
	}
}

// doRequest performs an HTTP request with rate limiting and retries.
// body is JSON-encoded once and re-read per attempt so retries are safe. Pass nil for GET.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	url := c.baseURL + path

	var bodyJSON []byte
	if body != nil {
		var err error
		bodyJSON, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to encode request body: %w", err)
		}
	}

	var lastErr error
	b := backoff.NewDefault()

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Wait for rate limiter with timeout
		// Use a shorter timeout to avoid context deadline issues
		limiterCtx, limiterCancel := context.WithTimeout(ctx, 15*time.Second)
		err := c.limiter.Wait(limiterCtx)
		limiterCancel()
		if err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("rate limiter error: context cancelled: %w", err)
			}
			return fmt.Errorf("rate limiter error: rate: Wait(n=1) exceeded timeout: %w", err)
		}

		var reqBody io.Reader
		if len(bodyJSON) > 0 {
			reqBody = bytes.NewReader(bodyJSON)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Accept", "application/json")
		if len(bodyJSON) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}

		// Set API key if provided (for providers like Tatum)
		if c.apiKey != "" {
			req.Header.Set("x-api-key", c.apiKey)
		}

		log.Debug().
			Str("method", method).
			Str("url", url).
			Int("attempt", attempt+1).
			Msg("Sending beacon API request")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < c.maxRetries {
				log.Debug().Err(err).Str("url", url).Int("attempt", attempt+1).Msg("request failed, retrying")
				if !b.Wait(ctx) {
					return ctx.Err()
				}
				continue
			}
			log.Error().Err(err).Str("url", url).Int("attempts", attempt+1).Msg("beacon request failed after retries")
			return fmt.Errorf("request failed after %d attempts: %w", attempt+1, err)
		}

		retry, err := c.readDoRequestResponse(resp, method, path, result)
		if retry {
			lastErr = err
			log.Debug().
				Int("status", resp.StatusCode).
				Str("url", url).
				Int("attempt", attempt+1).
				Msg("retryable HTTP error, backing off")
			if attempt < c.maxRetries {
				if !b.Wait(ctx) {
					return ctx.Err()
				}
				continue
			}
			log.Error().Err(err).Str("url", url).Int("status", resp.StatusCode).Msg("beacon retryable error, retries exhausted")
			return err
		}
		if err != nil {
			return err
		}
		return nil
	}

	return lastErr
}

// readDoRequestResponse reads and closes resp.Body exactly once. If retry is true, err is a *backoff.RetryableError and the caller may re-issue the request after backoff.
func (c *Client) readDoRequestResponse(resp *http.Response, method, path string, result interface{}) (retry bool, err error) {
	defer resp.Body.Close()

	if backoff.ShouldRetry(resp.StatusCode) {
		b, _ := io.ReadAll(resp.Body)
		return true, &backoff.RetryableError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(b)),
		}
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Str("path", path).Msg("beacon response body read failed")
		return false, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyPreview := string(bodyBytes)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "..."
		}
		log.Error().
			Int("status", resp.StatusCode).
			Str("path", path).
			Str("body_preview", bodyPreview).
			Msg("beacon API non-success status")
		return false, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, bodyPreview)
	}

	log.Debug().
		Str("method", method).
		Str("path", path).
		Int("status", resp.StatusCode).
		Int("body_size", len(bodyBytes)).
		Str("body_preview", string(bodyBytes[:min(200, len(bodyBytes))])).
		Msg("Beacon API response received")

	if result == nil {
		return false, nil
	}

	if err := json.Unmarshal(bodyBytes, result); err != nil {
		log.Error().
			Err(err).
			Str("path", path).
			Str("body", string(bodyBytes[:min(500, len(bodyBytes))])).
			Msg("failed to decode beacon response")
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	log.Debug().
		Str("method", method).
		Str("path", path).
		Int("status", resp.StatusCode).
		Msg("Beacon API request successful and parsed")

	return false, nil
}

// get performs a GET request.
func (c *Client) get(ctx context.Context, path string, result interface{}) error {
	return c.doRequest(ctx, http.MethodGet, path, nil, result)
}

// post performs a POST request with a JSON body.
func (c *Client) post(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.doRequest(ctx, http.MethodPost, path, body, result)
}

// Close releases resources held by the client.
func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}
