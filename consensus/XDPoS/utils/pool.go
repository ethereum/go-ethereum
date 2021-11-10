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
	onThresholdFn func(map[common.Hash]PoolObj) error
}

func NewPool(threshold int) *Pool {
	return &Pool{
		objList:   make(map[string]map[common.Hash]PoolObj),
		threshold: threshold,
	}
}

func (p *Pool) Add(obj PoolObj) error {
	poolKey := obj.PoolKey()
	objListKeyed, ok := p.objList[poolKey]
	if !ok {
		p.objList[poolKey] = make(map[common.Hash]PoolObj)
		objListKeyed = p.objList[poolKey]
	}
	objListKeyed[obj.Hash()] = obj
	if len(objListKeyed) >= p.threshold {
		delete(p.objList, poolKey)
		if p.onThresholdFn != nil {
			return p.onThresholdFn(objListKeyed)
		} else {
			return fmt.Errorf("no call back function for pool")
		}
	}
	return nil
}

func (p *Pool) Clear() {
	p.objList = make(map[string]map[common.Hash]PoolObj)
}

func (p *Pool) SetThreshold(t int) {
	p.threshold = t
}

func (p *Pool) SetOnThresholdFn(f func(map[common.Hash]PoolObj) error) {
	p.onThresholdFn = f
}
