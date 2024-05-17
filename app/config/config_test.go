package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	{
		_, err := New("testdata/invalid.yml")
		require.Error(t, err)
		assert.EqualErrorf(t, err, "can't read config file testdata/invalid.yml: open testdata/invalid.yml: no such file or directory", "expected error")
	}

	{
		p, err := New("testdata/config.yml")
		require.NoError(t, err)
		assert.Equal(t, []Volume{{Name: "root", Path: "/hostroot"}, {Name: "data", Path: "/data"}}, p.Volumes)
	}
}

func TestParameters_MarshalVolumes(t *testing.T) {
	p, err := New("testdata/config.yml")
	require.NoError(t, err)
	assert.Equal(t, []string{"root:/hostroot", "data:/data"}, p.MarshalVolumes())
}

func TestParameters_String(t *testing.T) {
	p, err := New("testdata/config.yml")
	require.NoError(t, err)

	exp := Parameters{
		Volumes:  []Volume{{Name: "root", Path: "/hostroot"}, {Name: "data", Path: "/data"}},
		filename: "testdata/config.yml",
	}
	assert.Equal(t, exp, *p)
}
