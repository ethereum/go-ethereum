package utils

import (
	"math/big"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/stretchr/testify/assert"
)

func TestPoolWithTimeout(t *testing.T) {
	assert := assert.New(t)
	var ret int
	onThresholdFn := func(po map[common.Hash]PoolObj, currentPoolObj PoolObj) error {
		for _, m := range po {
			if _, ok := m.(*Timeout); ok {
				ret += 1
			} else {
				t.Fatalf("wrong type passed into pool: %v", m)
			}
		}
		return nil
	}

	pool := NewPool(2) // 2 is the cert threshold
	ret = 0
	pool.SetOnThresholdFn(onThresholdFn)
	timeout1 := Timeout{Round: 1, Signature: []byte{1}}
	timeout2 := Timeout{Round: 1, Signature: []byte{2}}
	timeout3 := Timeout{Round: 1, Signature: []byte{3}}
	_, numOfItems, err := pool.Add(&timeout1)
	assert.Nil(err)
	assert.Equal(1, numOfItems)
	_, numOfItems, err = pool.Add(&timeout1)
	assert.Nil(err)
	// Duplicates should not be added
	assert.Equal(1, numOfItems)
	assert.Equal(0, ret)
	_, numOfItems, err = pool.Add(&timeout2)
	assert.Nil(err)
	assert.Equal(2, ret)

	_, numOfItems, err = pool.Add(&timeout3)
	assert.Nil(err)
	assert.Equal(2, ret)
	pool = NewPool(3) // 3 is the cert size
	ret = 0
	pool.SetOnThresholdFn(onThresholdFn)
	_, numOfItems, err = pool.Add(&timeout1)
	assert.Nil(err)
	assert.Equal(1, numOfItems)
	_, numOfItems, err = pool.Add(&timeout2)
	assert.Nil(err)
	assert.Equal(2, numOfItems)
	assert.Equal(ret, 0)
	pool.Clear()
	_, numOfItems, err = pool.Add(&timeout3)
	assert.Nil(err)
	assert.Equal(1, numOfItems)
	assert.Equal(0, ret)
}

func TestPoolWithVote(t *testing.T) {
	assert := assert.New(t)
	var ret int
	onThresholdFn := func(po map[common.Hash]PoolObj, currentPoolObj PoolObj) error {
		for _, m := range po {
			if _, ok := m.(*Vote); ok {
				ret += 1
			} else {
				t.Fatalf("wrong type passed into pool: %v", m)
			}
		}
		return nil
	}

	pool := NewPool(2) // 2 is the cert threshold
	ret = 0
	pool.SetOnThresholdFn(onThresholdFn)
	blockInfo1 := BlockInfo{Hash: common.BigToHash(big.NewInt(2047)), Round: 1, Number: big.NewInt(1)}
	blockInfo2 := BlockInfo{Hash: common.BigToHash(big.NewInt(4095)), Round: 1, Number: big.NewInt(1)}
	vote1 := Vote{ProposedBlockInfo: blockInfo1, Signature: []byte{1}}
	vote2 := Vote{ProposedBlockInfo: blockInfo2, Signature: []byte{2}}
	vote3 := Vote{ProposedBlockInfo: blockInfo1, Signature: []byte{3}}
	_, numOfItems, err := pool.Add(&vote1)
	assert.Nil(err)
	assert.Equal(1, numOfItems)
	// Duplicates should not be added
	_, numOfItems, err = pool.Add(&vote1)
	assert.Nil(err)
	assert.Equal(1, numOfItems)
	assert.Equal(ret, 0)

	_, numOfItems, err = pool.Add(&vote2)
	assert.Nil(err)
	// vote2 is on a different blockInfo to vote1
	assert.Equal(1, numOfItems)
	assert.Equal(0, ret)

	_, numOfItems, err = pool.Add(&vote3)
	assert.Nil(err)
	assert.Equal(2, numOfItems)

	assert.Equal(2, ret)
	pool = NewPool(3) // 3 is the cert size
	ret = 0
	pool.SetOnThresholdFn(onThresholdFn)

	_, numOfItems, err = pool.Add(&vote1)
	assert.Nil(err)
	assert.Equal(1, numOfItems)

	// vote2 is on a different blockInfo to vote1
	_, numOfItems, err = pool.Add(&vote2)
	assert.Nil(err)
	assert.Equal(1, numOfItems)

	_, numOfItems, err = pool.Add(&vote3)
	assert.Nil(err)
	assert.Equal(2, numOfItems)

	assert.Equal(0, ret)
	pool.Clear()
	assert.Empty(pool.objList)
	pool = NewPool(2) // 2 is the cert size
	ret = 0
	pool.SetOnThresholdFn(onThresholdFn)

	_, numOfItems, err = pool.Add(&vote1)
	assert.Nil(err)
	assert.Equal(1, numOfItems)

	// vote2 is on a different blockInfo to vote1
	_, numOfItems, err = pool.Add(&vote2)
	assert.Nil(err)
	assert.Equal(1, numOfItems)

	_, numOfItems, err = pool.Add(&vote3)
	assert.Nil(err)
	assert.Equal(2, numOfItems)
	assert.Equal(1, len(pool.objList)) //vote for one hash is cleared, but another remains
	pool.Clear()
	assert.Empty(pool.objList)
}
