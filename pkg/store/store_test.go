package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Function successfully connects to the database using provided configuration
func TestNewSuccessDatabaseConnection(t *testing.T) {
	cfg := &Config{
		ConnectTimeout:   5 * time.Second,
		ConnectionString: "host=localhost port=5432 user=postgres dbname=postgres password=postgres sslmode=disable",
		MigrationVersion: 1,
	}
	storage, err := New(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, storage)
}
