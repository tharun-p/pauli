package handlers

import (
	"context"

	"github.com/gin-gonic/gin"
)

// ListValidators returns distinct validator indices that have snapshot rows.
func (a *API) ListValidators(c *gin.Context) {
	limit, offset, err := parseLimitOffset(c)
	if err != nil {
		writeBadRequest(c, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()
	indices, err := a.Store.Repository().ListValidators(ctx, limit, offset)
	if err != nil {
		writeInternal(c)
		return
	}
	type row struct {
		ValidatorIndex uint64 `json:"validator_index"`
	}
	out := make([]row, 0, len(indices))
	for _, idx := range indices {
		out = append(out, row{ValidatorIndex: idx})
	}
	writeListJSON(c, out, limit, offset, len(out))
}
