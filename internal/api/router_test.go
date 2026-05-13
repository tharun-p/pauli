package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tharun/pauli/internal/api"
	"github.com/tharun/pauli/internal/storage"
)

// okStore implements storage.Store for routes that never touch the repository.
type okStore struct{}

func (okStore) RunMigrations() error { return nil }
func (okStore) HealthCheck() error   { return nil }
func (okStore) Close()               {}
func (okStore) Repository() storage.Repository {
	return nil
}

func TestHealthz_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := api.NewRouter(okStore{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestV1_AttestationRewards_MissingEpochWindow_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := api.NewRouter(okStore{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/attestation-rewards", nil))
	require.Equal(t, http.StatusBadRequest, w.Code)
}
