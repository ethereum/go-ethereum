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
	assert.Nil(pool.Add(&timeout1))
	assert.Nil(pool.Add(&timeout1))
	assert.Equal(ret, 0)
	assert.Nil(pool.Add(&timeout2))
	assert.Equal(ret, 2)
	assert.Nil(pool.Add(&timeout3))
	assert.Equal(ret, 2)
	pool = NewPool(3) // 3 is the cert size
	ret = 0
	pool.SetOnThresholdFn(onThresholdFn)
	assert.Nil(pool.Add(&timeout1))
	assert.Nil(pool.Add(&timeout2))
	assert.Equal(ret, 0)
	pool.Clear()
	assert.Nil(pool.Add(&timeout3))
	assert.Equal(ret, 0)
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
	assert.Nil(pool.Add(&vote1))
	assert.Nil(pool.Add(&vote1))
	assert.Equal(ret, 0)
	assert.Nil(pool.Add(&vote2))
	assert.Equal(ret, 0)
	assert.Nil(pool.Add(&vote3))
	assert.Equal(ret, 2)
	pool = NewPool(3) // 3 is the cert size
	ret = 0
	pool.SetOnThresholdFn(onThresholdFn)
	assert.Nil(pool.Add(&vote1))
	assert.Nil(pool.Add(&vote2))
	assert.Nil(pool.Add(&vote3))
	assert.Equal(ret, 0)
	pool.Clear()
	assert.Empty(pool.objList)
	pool = NewPool(2) // 2 is the cert size
	ret = 0
	pool.SetOnThresholdFn(onThresholdFn)
	assert.Nil(pool.Add(&vote1))
	assert.Nil(pool.Add(&vote2))
	assert.Nil(pool.Add(&vote3))
	assert.Equal(len(pool.objList), 1) //vote for one hash is cleared, but another remains
	pool.Clear()
	assert.Empty(pool.objList)
}
