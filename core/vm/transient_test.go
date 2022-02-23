package vm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

var (
	alice = common.Address{1}
	bob   = common.Address{2}
)

type pair struct {
	key, value int
}

func TestAsMap(t *testing.T) {
	store := NewTransientStore()
	store.Store(*uint256.NewInt(1), *uint256.NewInt(2), alice)
	if !isIncluded(store, []pair{{key: 1, value: 2}}, alice) {
		t.Fail()
	}
}

func TestCommit(t *testing.T) {
	store := NewTransientStore()
	writepair(store, 1, 2, alice)
	store.Call()
	writepair(store, 3, 4, alice)
	store.Commit()
	if !isIncluded(store, []pair{
		{key: 1, value: 2},
		{key: 3, value: 4},
	}, alice) {
		t.Fail()
	}

}

func TestRevert(t *testing.T) {
	store := NewTransientStore()
	writepair(store, 1, 2, alice)
	store.Call()
	writepair(store, 1, 4, alice)
	store.Revert()
	if !isIncluded(store, []pair{{key: 1, value: 2}}, alice) {
		t.Fail()
	}
}

func isIncluded(store *transientStore, pairs []pair, who common.Address) bool {
	for _, kv := range pairs {
		key := *uint256.NewInt(uint64(kv.key))
		val := store.Load(key, who)
		if !val.Eq(uint256.NewInt(uint64(kv.value))) {
			return false
		}
	}
	return true
}

func writepair(s *transientStore, key int, val int, who common.Address) {
	s.Store(*uint256.NewInt(uint64(key)), *uint256.NewInt(uint64(val)), who)
}

func TestIsolation(t *testing.T) {
	store := NewTransientStore()
	writepair(store, 1, 2, alice)
	writepair(store, 1, 3, bob)
	if !isIncluded(store, []pair{{key: 1, value: 2}}, alice) {
		t.Fail()
	}
	if !isIncluded(store, []pair{{key: 1, value: 3}}, bob) {
		t.Fail()
	}
}
