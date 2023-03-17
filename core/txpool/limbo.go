// Copyright 2023 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package txpool

import (
	"fmt"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// limboArea is a struct for maintaining 'not quite ready' transactions; it may be
// that transactions are broadcast out of order, making them arrive with
// nonce-gaps. In that case, they arrive in 'limboArea', and after a brief time,
// they are accepted as executable (the gaps have been resolved) or are discarded.
type limboArea struct {
	signer   types.Signer
	mu       sync.RWMutex
	txs      map[string]*types.Transaction
	txHashes map[common.Hash]string
	count    uint64

	wg      sync.WaitGroup
	closeCh chan any
}

func newLimbo(signer types.Signer) *limboArea {
	return &limboArea{
		signer:   signer,
		txs:      make(map[string]*types.Transaction),
		txHashes: make(map[common.Hash]string),
		closeCh:  make(chan any),
	}
}

// Add adds a transaction to limboArea. If the transaction already is present, this is
// a no-op.
// returns true if the transaction was added, false if already known
func (l *limboArea) Add(tx *types.Transaction) bool {
	from, _ := types.Sender(l.signer, tx)
	key := fmt.Sprintf("%x-%d", from[:10], tx.Nonce())
	hash := tx.Hash()
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.txHashes[hash]; ok {
		return false
	}
	l.count++
	l.txs[key] = tx
	l.txHashes[hash] = key
	return true
}

// Has returns true iff the tx with hash is in limboArea.
func (l *limboArea) Has(hash common.Hash) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, present := l.txHashes[hash]
	return present

}

func (l *limboArea) flush(fn func(*types.Transaction)) {
	// Copy-and-replace the txs
	l.mu.Lock()
	txs := l.txs
	//count := l.count
	l.txs = make(map[string]*types.Transaction)
	l.count = 0
	l.mu.Unlock()
	// Now flush them out
	var keys = make(sort.StringSlice, 0, len(txs))
	for k, _ := range txs {
		keys = append(keys, k)
	}
	sort.Sort(keys)
	for _, k := range keys {
		fn(txs[k])
	}
}

//func (l *limboArea) Start(pool TxPool) {
//	l.wg.Add(1)
//	go func() {
//		t := time.NewTimer(1 * time.Second)
//		defer l.wg.Done()
//		defer t.Stop()
//		for {
//			select {
//			case <-l.closeCh:
//				return
//			case <-t.C:
//				l.flush(pool.tryAddPending)
//				t.Reset(1 * time.Second)
//			}
//		}
//	}()
//}
//
//func (l *limboArea) Stop() {
//	close(l.closeCh)
//	l.wg.Wait()
//}
