package core

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	lru "github.com/hashicorp/golang-lru"
)

type txNoncerMap struct {
	fallback *state.StateDB
	nonces   map[common.Address]uint64
	lock     sync.Mutex
}

func (txn *txNoncerMap) get(addr common.Address) uint64 {
	txn.lock.Lock()
	defer txn.lock.Unlock()

	if _, ok := txn.nonces[addr]; !ok {
		txn.nonces[addr] = txn.fallback.GetNonce(addr)
	}
	return txn.nonces[addr]
}

func (txn *txNoncerMap) set(addr common.Address, nonce uint64) {
	txn.lock.Lock()
	defer txn.lock.Unlock()

	txn.nonces[addr] = nonce
}

type txNoncerLRU struct {
	fallback *state.StateDB
	nonces   *lru.Cache
}

func newTxNoncerMap(statedb *state.StateDB) *txNoncerMap {
	return &txNoncerMap{
		fallback: statedb.Copy(),
		nonces:   make(map[common.Address]uint64),
	}
}

func newTxNoncerLRU(statedb *state.StateDB) *txNoncerLRU {
	// lru cache size 1024 * 50 allocated 10 ~ 20 MB
	cache, _ := lru.New(1024 * 50)
	return &txNoncerLRU{
		fallback: statedb.Copy(),
		nonces:   cache,
	}
}

func TestTxNoncerMap(t *testing.T) {
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	txNoncer := newTxNoncerMap(statedb)

	var m runtime.MemStats
	runtime.GC()
	for i := 0; i < 10000000; i++ {
		var b [20]byte
		rand.Read(b[:])
		addr := common.Address(b)
		txNoncer.set(addr, uint64(0))
	}
	runtime.ReadMemStats(&m)
	fmt.Printf("Object memory: %.3f MB current\n", float64(m.Alloc)/1024/1024)
	fmt.Printf("System memory: %.3f MB current\n", float64(m.Sys)/1024/1024)
	fmt.Printf("Allocations:   %.3f million\n", float64(m.Mallocs)/1000000)
}

func TestTxNoncerLRU(t *testing.T) {
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	txNoncer := newTxNoncerLRU(statedb)

	var m runtime.MemStats
	runtime.GC()
	for i := 0; i < 10000000; i++ {
		var b [20]byte
		rand.Read(b[:])
		addr := common.Address(b)
		txNoncer.nonces.Add(addr, uint64(1))
	}
	runtime.ReadMemStats(&m)
	fmt.Printf("Object memory: %.3f MB current\n", float64(m.Alloc)/1024/1024)
	fmt.Printf("System memory: %.3f MB current\n", float64(m.Sys)/1024/1024)
	fmt.Printf("Allocations:   %.3f million\n", float64(m.Mallocs)/1000000)
}

func BenchmarkMapStore(b *testing.B) {
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	txNoncer := newTxNoncerMap(statedb)
	for i := 0; i < b.N; i++ {
		var byt [20]byte
		rand.Read(byt[:])
		addr := common.Address(byt)
		txNoncer.set(addr, uint64(1))
	}
}

func BenchmarkLRUStore(b *testing.B) {
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	txNoncer := newTxNoncerLRU(statedb)
	for i := 0; i < b.N; i++ {
		var byt [20]byte
		rand.Read(byt[:])
		addr := common.Address(byt)
		txNoncer.nonces.Add(addr, uint64(1))
	}
}

func BenchmarkMapLoad(b *testing.B) {
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	txNoncer := newTxNoncerMap(statedb)
	var byt [20]byte
	rand.Read(byt[:])
	addr := common.Address(byt)
	txNoncer.set(addr, uint64(1))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = txNoncer.get(addr)
	}
}

func BenchmarkLRULoad(b *testing.B) {
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	txNoncer := newTxNoncerLRU(statedb)
	var byt [20]byte
	rand.Read(byt[:])
	addr := common.Address(byt)
	txNoncer.nonces.Add(addr, uint64(1))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v, _ := txNoncer.nonces.Get(addr)
		_ = v.(uint64)
	}
}
