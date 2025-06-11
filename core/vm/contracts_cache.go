package vm

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
)

var (
	caches map[common.Address]*lru.Cache[string, []byte]
	mu     sync.RWMutex
)

func init() {
	caches = make(map[common.Address]*lru.Cache[string, []byte])
}

func addCache(precompile common.Address, input, output []byte) {
	mu.Lock()
	defer mu.Unlock()
	cache, ok := caches[precompile]
	if !ok {
		cache = lru.NewCache[string, []byte](128)
		caches[precompile] = cache
	}
	cache.Add(string(input), output)
}

func getCache(precompile common.Address, input []byte) ([]byte, bool) {
	mu.RLock()
	defer mu.RUnlock()
	cache, ok := caches[precompile]
	if !ok {
		caches[precompile] = lru.NewCache[string, []byte](128)
		return nil, false
	}
	return cache.Get(string(input))
}
