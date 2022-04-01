package utils

import (
	"sync"

	"github.com/XinFinOrg/XDPoSChain/common"
)

type PoolObj interface {
	Hash() common.Hash
	PoolKey() string
}
type Pool struct {
	objList   map[string]map[common.Hash]PoolObj
	threshold int
	lock      sync.RWMutex // Protects the pool fields
}

func NewPool(threshold int) *Pool {
	return &Pool{
		objList:   make(map[string]map[common.Hash]PoolObj),
		threshold: threshold,
	}
}

// return true if it has reached threshold
func (p *Pool) Add(obj PoolObj) (bool, int, map[common.Hash]PoolObj) {
	p.lock.Lock()
	defer p.lock.Unlock()
	poolKey := obj.PoolKey()
	objListKeyed, ok := p.objList[poolKey]
	if !ok {
		p.objList[poolKey] = make(map[common.Hash]PoolObj)
		objListKeyed = p.objList[poolKey]
	}
	objListKeyed[obj.Hash()] = obj
	numOfItems := len(objListKeyed)
	if numOfItems >= p.threshold {
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

func (p *Pool) PoolObjKeysList() []string {
	p.lock.RLock()
	defer p.lock.RUnlock()

	var keyList []string
	for key := range p.objList {
		keyList = append(keyList, key)
	}
	return keyList
}

// Given the pool object, clear all object under the same pool key
func (p *Pool) ClearPoolKeyByObj(obj PoolObj) {
	p.lock.Lock()
	defer p.lock.Unlock()

	poolKey := obj.PoolKey()
	delete(p.objList, poolKey)
}

// Given the pool key, clean its content
func (p *Pool) ClearByPoolKey(poolKey string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	delete(p.objList, poolKey)
}

func (p *Pool) Clear() {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.objList = make(map[string]map[common.Hash]PoolObj)
}

func (p *Pool) SetThreshold(t int) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.threshold = t
}
