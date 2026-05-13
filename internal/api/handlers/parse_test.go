package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func testContext(t *testing.T, target string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, target, nil)
	c.Request = req
	return c
}

func TestParseEpochWindow_epochShorthand(t *testing.T) {
	c := testContext(t, "/?epoch=7")
	from, to, err := parseEpochWindow(c)
	require.NoError(t, err)
	require.Equal(t, uint64(7), from)
	require.Equal(t, uint64(7), to)
}

func TestParseEpochWindow_fromTo(t *testing.T) {
	c := testContext(t, "/?from_epoch=1&to_epoch=3")
	from, to, err := parseEpochWindow(c)
	require.NoError(t, err)
	require.Equal(t, uint64(1), from)
	require.Equal(t, uint64(3), to)
}

func TestParseEpochWindow_conflict(t *testing.T) {
	c := testContext(t, "/?epoch=1&from_epoch=1&to_epoch=2")
	_, _, err := parseEpochWindow(c)
	require.Error(t, err)
}

func TestParseEpochWindow_missing(t *testing.T) {
	c := testContext(t, "/")
	_, _, err := parseEpochWindow(c)
	require.Error(t, err)
}

func TestParseSlotWindow_slotShorthand(t *testing.T) {
	c := testContext(t, "/?slot=99")
	from, to, err := parseSlotWindow(c)
	require.NoError(t, err)
	require.Equal(t, uint64(99), from)
	require.Equal(t, uint64(99), to)
}

func TestParseLimitOffset_defaults(t *testing.T) {
	c := testContext(t, "/")
	limit, offset, err := parseLimitOffset(c)
	require.NoError(t, err)
	require.Equal(t, defaultListLimit, limit)
	require.Equal(t, 0, offset)
}

func TestParseLimitOffset_clampMax(t *testing.T) {
	c := testContext(t, "/?limit=999999")
	limit, _, err := parseLimitOffset(c)
	require.NoError(t, err)
	require.Equal(t, maxListLimit, limit)
}

func TestMergeValidatorScope_queryMismatch(t *testing.T) {
	c := testContext(t, "/?validator_index=2")
	v := uint64(1)
	_, err := mergeValidatorScope(c, &v)
	require.Error(t, err)
}

func TestMergeValidatorScope_queryMatchesPath(t *testing.T) {
	c := testContext(t, "/?validator_index=1")
	v := uint64(1)
	scope, err := mergeValidatorScope(c, &v)
	require.NoError(t, err)
	require.NotNil(t, scope)
	require.Equal(t, uint64(1), *scope)
}
