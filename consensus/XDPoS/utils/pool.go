package utils

import (
	"github.com/XinFinOrg/XDPoSChain/common"
)

type PoolObj interface {
	Hash() common.Hash
	PoolKey() string
}
type Pool struct {
	objList   map[string]map[common.Hash]PoolObj
	threshold int
}

func NewPool(threshold int) *Pool {
	return &Pool{
		objList:   make(map[string]map[common.Hash]PoolObj),
		threshold: threshold,
	}
}

// return true if it has reached threshold
func (p *Pool) Add(obj PoolObj) (bool, int, map[common.Hash]PoolObj) {
	poolKey := obj.PoolKey()
	objListKeyed, ok := p.objList[poolKey]
	if !ok {
		p.objList[poolKey] = make(map[common.Hash]PoolObj)
		objListKeyed = p.objList[poolKey]
	}
	objListKeyed[obj.Hash()] = obj
	numOfItems := len(objListKeyed)
	if numOfItems >= p.threshold {
		delete(p.objList, poolKey)
		return true, numOfItems, objListKeyed
	}
	return false, numOfItems, objListKeyed
}
func (p *Pool) Size(obj PoolObj) int {
	poolKey := obj.PoolKey()
	objListKeyed, ok := p.objList[poolKey]
	if !ok {
		return 0
	}
	return len(objListKeyed)
}

func (p *Pool) Clear() {
	p.objList = make(map[string]map[common.Hash]PoolObj)
}

func (p *Pool) SetThreshold(t int) {
	p.threshold = t
}
