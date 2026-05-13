package handlers

import (
	"context"

	"github.com/gin-gonic/gin"
)

// ListPenalties returns penalty rows for a validator in an epoch range.
func (a *API) ListPenalties(c *gin.Context) {
	idx, err := parseUintPath(c, "validatorIndex")
	if err != nil {
		writeBadRequest(c, err.Error())
		return
	}
	fromE, toE, err := parseEpochWindow(c)
	if err != nil {
		writeBadRequest(c, err.Error())
		return
	}
	limit, offset, err := parseLimitOffset(c)
	if err != nil {
		writeBadRequest(c, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()
	rows, err := a.Store.Repository().GetValidatorPenalties(ctx, idx, fromE, toE, limit, offset)
	if err != nil {
		writeInternal(c)
		return
	}
	writeListJSON(c, rows, limit, offset, len(rows))
}
