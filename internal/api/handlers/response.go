package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// writeError sends a JSON error body with a stable shape for clients.
func writeError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

func writeBadRequest(c *gin.Context, message string) {
	writeError(c, http.StatusBadRequest, "bad_request", message)
}

func writeInternal(c *gin.Context) {
	writeError(c, http.StatusInternalServerError, "internal_error", "internal server error")
}

// listMeta accompanies paginated list responses.
type listMeta struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Count  int `json:"count"`
}

func writeListJSON(c *gin.Context, data any, limit, offset, count int) {
	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": listMeta{
			Limit:  limit,
			Offset: offset,
			Count:  count,
		},
	})
}
