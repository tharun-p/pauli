package handlers

import (
	"context"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// LatestSnapshot returns the newest snapshot for one validator.
func (a *API) LatestSnapshot(c *gin.Context) {
	idx, err := parseUintPath(c, "validatorIndex")
	if err != nil {
		writeBadRequest(c, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()
	snap, err := a.Store.Repository().GetLatestSnapshot(ctx, idx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(c, 404, "not_found", "no snapshot found for this validator")
			return
		}
		writeInternal(c)
		return
	}
	c.JSON(200, snap)
}

// ListSnapshots returns snapshots for a validator in a slot range (paginated).
func (a *API) ListSnapshots(c *gin.Context) {
	idx, err := parseUintPath(c, "validatorIndex")
	if err != nil {
		writeBadRequest(c, err.Error())
		return
	}
	fromS := c.Query("from_slot")
	toS := c.Query("to_slot")
	if fromS == "" || toS == "" {
		writeBadRequest(c, "from_slot and to_slot are required")
		return
	}
	fromSlot, err := strconv.ParseUint(fromS, 10, 64)
	if err != nil {
		writeBadRequest(c, "invalid from_slot")
		return
	}
	toSlot, err := strconv.ParseUint(toS, 10, 64)
	if err != nil {
		writeBadRequest(c, "invalid to_slot")
		return
	}
	if fromSlot > toSlot {
		writeBadRequest(c, "from_slot must be less than or equal to to_slot")
		return
	}
	limit, offset, err := parseLimitOffset(c)
	if err != nil {
		writeBadRequest(c, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()
	rows, err := a.Store.Repository().ListValidatorSnapshots(ctx, idx, fromSlot, toSlot, limit, offset)
	if err != nil {
		writeInternal(c)
		return
	}
	writeListJSON(c, rows, limit, offset, len(rows))
}

// CountSnapshots returns how many snapshot rows exist for a validator.
func (a *API) CountSnapshots(c *gin.Context) {
	idx, err := parseUintPath(c, "validatorIndex")
	if err != nil {
		writeBadRequest(c, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()
	n, err := a.Store.Repository().CountSnapshots(ctx, idx)
	if err != nil {
		writeInternal(c)
		return
	}
	c.JSON(200, gin.H{"count": n})
}
