package beacon

import (
	"errors"
	"fmt"
	"net/http"
)

// HTTPResponseError is returned for Beacon API responses that are not HTTP 200
// (after retryable statuses are handled).
type HTTPResponseError struct {
	StatusCode int
	Path       string
	Body       string
}

func (e *HTTPResponseError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("unexpected status %d: %s", e.StatusCode, e.Body)
}

// IsNotFound reports whether err is or wraps an HTTPResponseError with status 404.
func IsNotFound(err error) bool {
	var he *HTTPResponseError
	return errors.As(err, &he) && he.StatusCode == http.StatusNotFound
}
