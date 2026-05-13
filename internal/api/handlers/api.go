// Package handlers implements HTTP handlers for the pauli read API.
package handlers

import (
	"time"

	"github.com/tharun/pauli/internal/storage"
)

// Default limits for list endpoints (also enforced server-side).
const (
	defaultListLimit = 100
	maxListLimit     = 1000
	requestTimeout   = 30 * time.Second
)

// API holds dependencies for HTTP handlers.
type API struct {
	Store storage.Store
}

// New constructs an API backed by the given store.
func New(store storage.Store) *API {
	return &API{Store: store}
}
