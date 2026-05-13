package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/tharun/pauli/internal/storage"
)

// NewHandler returns an http.Handler for the pauli REST API.
func NewHandler(store storage.Store) http.Handler {
	mux := http.NewServeMux()
	s := &server{store: store}
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /v1/validators/{validatorIndex}/snapshots/latest", s.latestSnapshot)
	return mux
}

type server struct {
	store storage.Store
}

func (s *server) healthz(w http.ResponseWriter, r *http.Request) {
	if err := s.store.HealthCheck(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *server) latestSnapshot(w http.ResponseWriter, r *http.Request) {
	idxStr := r.PathValue("validatorIndex")
	idx, err := strconv.ParseUint(idxStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid validator index", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	snap, err := s.store.Repository().GetLatestSnapshot(ctx, idx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(snap); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}
