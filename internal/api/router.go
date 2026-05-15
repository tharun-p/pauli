// Package api wires the pauli HTTP read API (Gin).
package api

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/tharun/pauli/internal/api/handlers"
	"github.com/tharun/pauli/internal/storage"
)

// NewRouter builds the Gin engine with all public routes.
func NewRouter(store storage.Store) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	h := handlers.New(store)
	r.GET("/healthz", h.Healthz)

	mountOpenAPISpec(r)

	v1 := r.Group("/v1")
	{
		v1.GET("/validators", h.ListValidators)

		v1.GET("/attestation-rewards", h.ListAttestationRewardsQuery)
		v1.GET("/block-proposer-rewards", h.ListBlockProposerRewardsQuery)
		v1.GET("/sync-committee-rewards", h.ListSyncCommitteeRewardsQuery)

		v1.GET("/validators/:validatorIndex/snapshots/latest", h.LatestSnapshot)
		v1.GET("/validators/:validatorIndex/snapshots", h.ListSnapshots)
		v1.GET("/validators/:validatorIndex/snapshots/count", h.CountSnapshots)

		v1.GET("/validators/:validatorIndex/attestation-rewards", h.ListAttestationRewardsScoped)
		v1.GET("/validators/:validatorIndex/block-proposer-rewards", h.ListBlockProposerRewardsScoped)
		v1.GET("/validators/:validatorIndex/sync-committee-rewards", h.ListSyncCommitteeRewardsScoped)
	}

	return r
}

func mountOpenAPISpec(r *gin.Engine) {
	var spec []byte
	for _, p := range []string{"docs/openapi.yaml", "./docs/openapi.yaml"} {
		b, err := os.ReadFile(p)
		if err == nil && len(b) > 0 {
			spec = b
			break
		}
	}
	if len(spec) == 0 {
		return
	}
	r.GET("/openapi.yaml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml", spec)
	})
}
