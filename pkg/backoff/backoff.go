package backoff

import (
	"context"
	"math"
	"math/rand"
	"net/http"
	"time"
)

// Config holds the backoff configuration.
type Config struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	JitterFactor float64 // 0.2 means +/- 20%
}

// DefaultConfig returns a default backoff configuration.
func DefaultConfig() Config {
	return Config{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.2,
	}
}

// Backoff implements exponential backoff with jitter.
type Backoff struct {
	cfg      Config
	attempts int
}

// New creates a new Backoff with the given configuration.
func New(cfg Config) *Backoff {
	return &Backoff{
		cfg:      cfg,
		attempts: 0,
	}
}

// NewDefault creates a new Backoff with default configuration.
func NewDefault() *Backoff {
	return New(DefaultConfig())
}

// Reset resets the backoff attempts counter.
func (b *Backoff) Reset() {
	b.attempts = 0
}

// Attempts returns the current number of attempts.
func (b *Backoff) Attempts() int {
	return b.attempts
}

// NextDelay calculates the next delay with exponential backoff and jitter.
func (b *Backoff) NextDelay() time.Duration {
	if b.attempts == 0 {
		b.attempts++
		return b.cfg.InitialDelay
	}

	// Calculate exponential delay
	delay := float64(b.cfg.InitialDelay) * math.Pow(b.cfg.Multiplier, float64(b.attempts))

	// Apply jitter (+/- jitterFactor)
	jitter := 1.0 + (rand.Float64()*2-1)*b.cfg.JitterFactor
	delay *= jitter

	// Cap at max delay
	if delay > float64(b.cfg.MaxDelay) {
		delay = float64(b.cfg.MaxDelay)
	}

	b.attempts++
	return time.Duration(delay)
}

// Wait waits for the next backoff delay or until context is cancelled.
// Returns true if the wait completed, false if context was cancelled.
func (b *Backoff) Wait(ctx context.Context) bool {
	delay := b.NextDelay()
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// ShouldRetry returns true if the HTTP status code indicates a retryable error.
// Specifically handles 429 (Too Many Requests) and 503 (Service Unavailable).
func ShouldRetry(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode == http.StatusServiceUnavailable
}

// RetryableError represents an error that can be retried.
type RetryableError struct {
	StatusCode int
	Message    string
}

func (e *RetryableError) Error() string {
	return e.Message
}

// IsRetryable checks if an error is a RetryableError.
func IsRetryable(err error) bool {
	_, ok := err.(*RetryableError)
	return ok
}

// Retry executes the given function with exponential backoff.
// It retries on RetryableError until maxRetries is reached or context is cancelled.
func Retry(ctx context.Context, maxRetries int, fn func() error) error {
	b := NewDefault()

	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if !IsRetryable(lastErr) {
			return lastErr
		}

		if i < maxRetries {
			if !b.Wait(ctx) {
				return ctx.Err()
			}
		}
	}

	return lastErr
}
