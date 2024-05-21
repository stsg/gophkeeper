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
		assert.Equal(t, []Data{{Name: "data01", Value: "value01"}, {Name: "data02", Value: "value02"}}, p.DataSet)
	}
}

func TestParameters_MarshalDataSet(t *testing.T) {
	p, err := New("testdata/config.yml")
	require.NoError(t, err)
	assert.Equal(t, []string{"data01:value01", "data02:value02"}, p.MarshalDataSet())
}

func TestParameters_String(t *testing.T) {
	p, err := New("testdata/config.yml")
	require.NoError(t, err)

	exp := Parameters{
		DataSet:  []Data{{Name: "data01", Value: "value01"}, {Name: "data02", Value: "value02"}},
		filename: "testdata/config.yml",
	}
	assert.Equal(t, exp, *p)
}
