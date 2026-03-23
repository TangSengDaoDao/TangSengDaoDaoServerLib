package wkhttp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTokenCacheInfoJSONKeepsRoleBoundaries(t *testing.T) {
	raw := EncodeTokenCacheInfo("u1", "hacker@superAdmin", "user")
	info, err := ParseTokenCacheInfo(raw)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "u1", info.UID)
	assert.Equal(t, "hacker@superAdmin", info.Name)
	assert.Equal(t, "user", info.Role)
}

func TestParseTokenCacheInfoLegacyRoleTokenRejected(t *testing.T) {
	info, err := ParseTokenCacheInfo("u1@hacker@superAdmin@user")
	require.Error(t, err)
	assert.Nil(t, info)
}

func TestParseTokenCacheInfoLegacyUIDAndNameStillAllowed(t *testing.T) {
	info, err := ParseTokenCacheInfo("u1@normal")
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "u1", info.UID)
	assert.Equal(t, "normal", info.Name)
	assert.Empty(t, info.Role)
}
