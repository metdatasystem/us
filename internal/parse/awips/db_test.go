package internal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDatabasePool(t *testing.T) {
	pool, err := newDatabasePool("postgres://mds:@localhost:5432/mds")
	assert.NoError(t, err)
	assert.NotNil(t, pool)

	err = pool.Ping(context.Background())
	assert.NoError(t, err)
}
