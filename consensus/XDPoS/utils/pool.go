package utils

import (
	"fmt"

	"github.com/XinFinOrg/XDPoSChain/common"
)

type PoolObj interface {
	Hash() common.Hash
	PoolKey() string
}
type Pool struct {
	objList       map[string]map[common.Hash]PoolObj
	threshold     int
	onThresholdFn func(objsInPool map[common.Hash]PoolObj, currentObj PoolObj) error
}

func NewPool(threshold int) *Pool {
	return &Pool{
		objList:   make(map[string]map[common.Hash]PoolObj),
		threshold: threshold,
	}
}

// call the hook function onThresholdFn if reached threshold and return boolean to indicate whether pool has reached threshold
func (p *Pool) Add(obj PoolObj) (bool, int, error) {
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
		if p.onThresholdFn != nil {
			return true, numOfItems, p.onThresholdFn(objListKeyed, obj)
		} else {
			return true, numOfItems, fmt.Errorf("no call back function for pool")
		}
	}
	return false, numOfItems, nil
}

func (p *Pool) Clear() {
	p.objList = make(map[string]map[common.Hash]PoolObj)
}

func (p *Pool) SetThreshold(t int) {
	p.threshold = t
}

func (p *Pool) SetOnThresholdFn(f func(objsInPool map[common.Hash]PoolObj, currentObj PoolObj) error) {
	p.onThresholdFn = f
}
