package utils

import (
	"sync"

	"github.com/XinFinOrg/XDPoSChain/common"
)

type PoolObj interface {
	Hash() common.Hash
	PoolKey() string
	GetSigner() common.Address
	DeepCopy() interface{}
}

// Note: must use `NewPool()` to create `Pool` since field `objList` is a map
type Pool struct {
	objList map[string]map[common.Hash]PoolObj
	lock    sync.RWMutex // Protects the pool fields
}

func NewPool() *Pool {
	return &Pool{
		objList: make(map[string]map[common.Hash]PoolObj),
	}
}

func (p *Pool) Get() map[string]map[common.Hash]PoolObj {
	p.lock.RLock()
	defer p.lock.RUnlock()
	dataCopy := make(map[string]map[common.Hash]PoolObj, len(p.objList))
	for k1, v1 := range p.objList {
		dataCopy[k1] = make(map[common.Hash]PoolObj, len(v1))
		for k2, v2 := range v1 {
			dataCopy[k1][k2] = v2.DeepCopy().(PoolObj)
		}
	}

	return dataCopy
}

// return true if it has reached threshold
func (p *Pool) Add(obj PoolObj) (int, map[common.Hash]PoolObj) {
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

	dataCopy := make(map[common.Hash]PoolObj, len(objListKeyed))
	for k, v := range objListKeyed {
		dataCopy[k] = v.DeepCopy().(PoolObj)
	}

	return numOfItems, dataCopy
}

func (p *Pool) Size(obj PoolObj) int {
	p.lock.RLock()
	defer p.lock.RUnlock()
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

func (p *Pool) GetObjsByKey(poolKey string) []PoolObj {
	p.lock.RLock()
	defer p.lock.RUnlock()

	objListKeyed, ok := p.objList[poolKey]
	if !ok {
		return []PoolObj{}
	}
	objList := make([]PoolObj, len(objListKeyed))
	cnt := 0
	for _, obj := range objListKeyed {
		objList[cnt] = obj.DeepCopy().(PoolObj)
		cnt++
	}
	return objList
}
