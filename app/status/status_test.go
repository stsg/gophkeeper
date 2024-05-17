package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_GetNoExt(t *testing.T) {

	hst := Host{
		Volumes: []Volume{{Name: "root", Path: "/"}},
	}

	res, err := hst.Get()
	require.NoError(t, err)
	t.Logf("%+v", res)
	assert.Equal(t, 1, len(res.Volumes))
	assert.Equal(t, "root", res.Volumes["root"].Name)
	assert.Equal(t, "/", res.Volumes["root"].Path)
	assert.True(t, res.Volumes["root"].UsagePercent > 0)
	assert.True(t, res.MemPercent > 0)
	assert.True(t, res.Loads.One > 0)
	assert.True(t, res.Uptime > 0)
}
