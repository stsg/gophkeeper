package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_GetNoExt(t *testing.T) {

	hst := Host{}

	res, err := hst.Get()
	require.NoError(t, err)
	t.Logf("%+v", res)
	assert.True(t, res.MemPercent > 0)
	assert.True(t, res.Loads.One > 0)
	assert.True(t, res.Uptime > 0)
}
