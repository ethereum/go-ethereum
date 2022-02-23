package vm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type storekey struct {
	key uint256.Int
	who common.Address
}
type transientStore struct {
	store    map[storekey]uint256.Int
	rollback []rollbackEntry
	shadow   []int
}

type rollbackEntry struct {
	address uint256.Int
	who     common.Address
	value   uint256.Int
}

func NewTransientStore() *transientStore {
	return &transientStore{
		store:    make(map[storekey]uint256.Int),
		rollback: nil,
		shadow:   nil,
	}
}

func (ts *transientStore) Store(key uint256.Int, value uint256.Int, who common.Address) {
	ts.rollback = append(ts.rollback, rollbackEntry{
		address: key,
		value:   ts.store[storekey{key, who}],
		who:     who,
	})
	ts.store[storekey{key, who}] = value
}

func (ts *transientStore) Load(key uint256.Int, who common.Address) uint256.Int {
	return ts.store[storekey{key, who}]
}

func (ts *transientStore) Call() {
	ts.shadow = append(ts.shadow, len(ts.rollback))
}

func (ts *transientStore) Revert() {
	oldlen := ts.shadow[len(ts.shadow)-1]
	reverts := ts.rollback[oldlen:len(ts.rollback)]
	ts.rollback = ts.rollback[0:oldlen]
	for i := len(reverts) - 1; i >= 0; i-- {
		ts.store[storekey{reverts[i].address, reverts[i].who}] = reverts[i].value
	}
}

func (ts *transientStore) Commit() {
	oldlen := ts.shadow[len(ts.shadow)-1]
	ts.rollback = ts.rollback[0:oldlen]
}
