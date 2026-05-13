package handlers

import (
	"context"

	"github.com/gin-gonic/gin"
)

// ListAttestationRewardsQuery lists attestation rewards (all validators unless validator_index is set).
func (a *API) ListAttestationRewardsQuery(c *gin.Context) {
	a.listAttestationRewards(c, nil)
}

// ListAttestationRewardsScoped lists attestation rewards for the validator in the path.
func (a *API) ListAttestationRewardsScoped(c *gin.Context) {
	idx, err := parseUintPath(c, "validatorIndex")
	if err != nil {
		writeBadRequest(c, err.Error())
		return
	}
	a.listAttestationRewards(c, &idx)
}

func (a *API) listAttestationRewards(c *gin.Context, pathValidator *uint64) {
	scope, err := mergeValidatorScope(c, pathValidator)
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
	rows, err := a.Store.Repository().ListAttestationRewards(ctx, scope, fromE, toE, limit, offset)
	if err != nil {
		writeInternal(c)
		return
	}
	writeListJSON(c, rows, limit, offset, len(rows))
}

// ListBlockProposerRewardsQuery lists block proposer rewards by slot window.
func (a *API) ListBlockProposerRewardsQuery(c *gin.Context) {
	a.listBlockProposerRewards(c, nil)
}

// ListBlockProposerRewardsScoped scopes block proposer rewards to the path validator.
func (a *API) ListBlockProposerRewardsScoped(c *gin.Context) {
	idx, err := parseUintPath(c, "validatorIndex")
	if err != nil {
		writeBadRequest(c, err.Error())
		return
	}
	a.listBlockProposerRewards(c, &idx)
}

func (a *API) listBlockProposerRewards(c *gin.Context, pathValidator *uint64) {
	scope, err := mergeValidatorScope(c, pathValidator)
	if err != nil {
		writeBadRequest(c, err.Error())
		return
	}
	fromS, toS, err := parseSlotWindow(c)
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
	rows, err := a.Store.Repository().ListBlockProposerRewards(ctx, scope, fromS, toS, limit, offset)
	if err != nil {
		writeInternal(c)
		return
	}
	writeListJSON(c, rows, limit, offset, len(rows))
}

// ListSyncCommitteeRewardsQuery lists sync committee rewards by slot window.
func (a *API) ListSyncCommitteeRewardsQuery(c *gin.Context) {
	a.listSyncCommitteeRewards(c, nil)
}

// ListSyncCommitteeRewardsScoped scopes sync committee rewards to the path validator.
func (a *API) ListSyncCommitteeRewardsScoped(c *gin.Context) {
	idx, err := parseUintPath(c, "validatorIndex")
	if err != nil {
		writeBadRequest(c, err.Error())
		return
	}
	a.listSyncCommitteeRewards(c, &idx)
}

func (a *API) listSyncCommitteeRewards(c *gin.Context, pathValidator *uint64) {
	scope, err := mergeValidatorScope(c, pathValidator)
	if err != nil {
		writeBadRequest(c, err.Error())
		return
	}
	fromS, toS, err := parseSlotWindow(c)
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
	rows, err := a.Store.Repository().ListSyncCommitteeRewards(ctx, scope, fromS, toS, limit, offset)
	if err != nil {
		writeInternal(c)
		return
	}
	writeListJSON(c, rows, limit, offset, len(rows))
}
