package utils

import (
	"sync"

	"github.com/XinFinOrg/XDPoSChain/common"
)

type PoolObj interface {
	Hash() common.Hash
	PoolKey() string
	GetSigner() common.Address
}
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
	return p.objList
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
	return numOfItems, objListKeyed
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

func (p *Pool) GetObjsByKey(poolKey string) []PoolObj {
	p.lock.Lock()
	defer p.lock.Unlock()

	objListKeyed, ok := p.objList[poolKey]
	if !ok {
		return []PoolObj{}
	}
	objList := make([]PoolObj, len(objListKeyed))
	cnt := 0
	for _, obj := range objListKeyed {
		objList[cnt] = obj
		cnt += 1
	}
	return objList
}
