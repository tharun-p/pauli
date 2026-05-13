package handlers

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

// parseUintPath returns a path parameter as uint64 or an error suitable for bad_request.
func parseUintPath(c *gin.Context, name string) (uint64, error) {
	s := c.Param(name)
	if s == "" {
		return 0, fmt.Errorf("missing path parameter %q", name)
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %q", name)
	}
	return v, nil
}

// parseLimitOffset reads limit and offset query params with defaults and a hard max.
func parseLimitOffset(c *gin.Context) (limit, offset int, err error) {
	limit = defaultListLimit
	if s := c.Query("limit"); s != "" {
		limit, err = strconv.Atoi(s)
		if err != nil || limit < 1 {
			return 0, 0, fmt.Errorf("invalid limit")
		}
		if limit > maxListLimit {
			limit = maxListLimit
		}
	}
	offset = 0
	if s := c.Query("offset"); s != "" {
		offset, err = strconv.Atoi(s)
		if err != nil || offset < 0 {
			return 0, 0, fmt.Errorf("invalid offset")
		}
	}
	return limit, offset, nil
}

// parseEpochWindow resolves inclusive epoch bounds from epoch shorthand or from_epoch/to_epoch.
func parseEpochWindow(c *gin.Context) (from, to uint64, err error) {
	epoch := c.Query("epoch")
	fromS := c.Query("from_epoch")
	toS := c.Query("to_epoch")
	if epoch != "" {
		if fromS != "" || toS != "" {
			return 0, 0, fmt.Errorf("use either epoch or from_epoch and to_epoch, not both")
		}
		v, perr := strconv.ParseUint(epoch, 10, 64)
		if perr != nil {
			return 0, 0, fmt.Errorf("invalid epoch")
		}
		return v, v, nil
	}
	if fromS == "" || toS == "" {
		return 0, 0, fmt.Errorf("from_epoch and to_epoch are required, or pass epoch for a single epoch")
	}
	from, err = strconv.ParseUint(fromS, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid from_epoch")
	}
	to, err = strconv.ParseUint(toS, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid to_epoch")
	}
	if from > to {
		return 0, 0, fmt.Errorf("from_epoch must be less than or equal to to_epoch")
	}
	return from, to, nil
}

// parseSlotWindow resolves inclusive slot bounds from slot shorthand or from_slot/to_slot.
func parseSlotWindow(c *gin.Context) (from, to uint64, err error) {
	slot := c.Query("slot")
	fromS := c.Query("from_slot")
	toS := c.Query("to_slot")
	if slot != "" {
		if fromS != "" || toS != "" {
			return 0, 0, fmt.Errorf("use either slot or from_slot and to_slot, not both")
		}
		v, perr := strconv.ParseUint(slot, 10, 64)
		if perr != nil {
			return 0, 0, fmt.Errorf("invalid slot")
		}
		return v, v, nil
	}
	if fromS == "" || toS == "" {
		return 0, 0, fmt.Errorf("from_slot and to_slot are required, or pass slot for a single slot")
	}
	from, err = strconv.ParseUint(fromS, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid from_slot")
	}
	to, err = strconv.ParseUint(toS, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid to_slot")
	}
	if from > to {
		return 0, 0, fmt.Errorf("from_slot must be less than or equal to to_slot")
	}
	return from, to, nil
}

// optionalValidatorQuery parses optional validator_index query param; empty means nil (all validators).
func optionalValidatorQuery(c *gin.Context) (*uint64, error) {
	s := c.Query("validator_index")
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid validator_index")
	}
	return &v, nil
}

// mergeValidatorScope applies path validator when present; rejects conflicting query validator_index.
func mergeValidatorScope(c *gin.Context, pathValidator *uint64) (*uint64, error) {
	q, err := optionalValidatorQuery(c)
	if err != nil {
		return nil, err
	}
	if pathValidator == nil {
		return q, nil
	}
	if q != nil && *q != *pathValidator {
		return nil, fmt.Errorf("validator_index query does not match path validator")
	}
	return pathValidator, nil
}
