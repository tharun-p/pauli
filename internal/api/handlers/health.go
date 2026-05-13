package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Healthz reports process liveness and database connectivity.
func (a *API) Healthz(c *gin.Context) {
	if err := a.Store.HealthCheck(); err != nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}
	c.Status(http.StatusOK)
}
