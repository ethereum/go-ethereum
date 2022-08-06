package utils

import (
	"testing"

	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/stretchr/testify/assert"
)

func TestPoolAdd(t *testing.T) {
	assert := assert.New(t)

	pool := NewPool(2) // 2 is the cert threshold
	timeout1 := types.Timeout{Round: 1, Signature: []byte{1}}
	timeout2 := types.Timeout{Round: 1, Signature: []byte{2}}
	timeout3 := types.Timeout{Round: 1, Signature: []byte{3}}
	timeout4 := types.Timeout{Round: 1, Signature: []byte{4}}
	thresholdReached, numOfItems, pooledTimeouts := pool.Add(&timeout1)
	assert.NotNil(pooledTimeouts)
	assert.Equal(1, numOfItems)
	assert.False(thresholdReached)
	thresholdReached, numOfItems, pooledTimeouts = pool.Add(&timeout1)
	assert.NotNil(pooledTimeouts)
	assert.False(thresholdReached)
	// Duplicates should not be added
	assert.Equal(1, numOfItems)

	// Should add the one that is not a duplicates
	thresholdReached, numOfItems, pooledTimeouts = pool.Add(&timeout2)
	assert.True(thresholdReached)
	assert.NotNil(pooledTimeouts)
	assert.Equal(2, numOfItems)

	// Try to add one more to the same round, it should also trigger threshold
	thresholdReached, numOfItems, pooledTimeouts = pool.Add(&timeout3)
	assert.True(thresholdReached)
	assert.NotNil(pooledTimeouts)
	assert.Equal(3, numOfItems)

	// Only after manually clearned the pool at its objKey, we shall not have any value for this particular key
	pool.ClearPoolKeyByObj(&timeout3)
	thresholdReached, numOfItems, pooledTimeouts = pool.Add(&timeout4)
	assert.False(thresholdReached)
	assert.NotNil(pooledTimeouts)
	assert.Equal(1, numOfItems)

	pool = NewPool(3) // 3 is the cert size
	thresholdReached, numOfItems, pooledTimeouts = pool.Add(&timeout1)
	assert.False(thresholdReached)
	assert.NotNil(pooledTimeouts)
	assert.Equal(1, numOfItems)

	thresholdReached, numOfItems, pooledTimeouts = pool.Add(&timeout2)
	assert.False(thresholdReached)
	assert.Equal(2, numOfItems)
	assert.NotNil(pooledTimeouts)
	pool.Clear()

	// Pool has been cleared. Start from 0 again
	thresholdReached, numOfItems, pooledTimeouts = pool.Add(&timeout3)
	assert.False(thresholdReached)
	assert.Equal(1, numOfItems)
	assert.NotNil(pooledTimeouts)
}
