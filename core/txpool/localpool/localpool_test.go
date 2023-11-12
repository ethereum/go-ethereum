package localpool

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

func (l *LocalPool) verifyConsistency() error {
	for _, list := range l.allAccounts {
		for _, tx := range list {
			if _, ok := l.allTxs[tx.Hash()]; !ok {
				return errors.New("tx in nonceOrderedList but not in all txs")
			}
		}
	}
	for _, tx := range l.allTxs {
		found := 0
		for _, list := range l.allAccounts {
			if tx2, ok := list[tx.Nonce()]; ok {
				if tx.Hash() == tx2.Hash() {
					found++
				}
			}
		}
		if found != 1 {
			return errors.New("tx in all txs but not in nonceOrderedList")
		}
	}
	return nil
}

func TestReset(t *testing.T) {
	// Generate a faucet
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)
	// Setup a mock blockchain
	bc := &MockBC{
		currentBlock: &types.Header{Root: common.Hash{}},
		dbs:          make(map[common.Hash]*state.StateDB),
	}
	initialDB, err := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	if err != nil {
		t.Fatal(err)
	}
	initialDB.CreateAccount(addr)
	initialDB.SetBalance(addr, big.NewInt(1_000_000_000_000_000_000))
	bc.SetState(common.Hash{}, initialDB)
	// Setup the txpool
	pool, err := NewLocalPool(bc, types.LatestSigner(params.AllDevChainProtocolChanges))
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Init(nil, &types.Header{Root: common.Hash{}, GasLimit: 200_000}, func(addr common.Address, reserve bool) error { return nil }); err != nil {
		t.Fatal(err)
	}
	// Queue transactions
	// TODO there might be an off-by-one error here
	for i := 1; i < 100; i++ {
		tx := types.NewTransaction(uint64(i), common.Address{}, new(big.Int), 100000, big.NewInt(1234), nil)
		signer := types.LatestSigner(params.AllDevChainProtocolChanges)
		signed, err := types.SignTx(tx, signer, key)
		if err != nil {
			t.Fatal(err)
		}
		if errs := pool.Add([]*types.Transaction{signed}, true, false); errs[0] != nil {
			t.Fatal(errs[0])
		}
	}
	// Verify integrity
	if err := pool.verifyConsistency(); err != nil {
		t.Fatal(err)
	}
	pending, queued := pool.ContentFrom(addr)
	if len(pending) != 99 || len(queued) != 0 {
		t.Fatal(len(pending), len(queued))
	}
	// Reset the pool
	newDB := initialDB.Copy()
	newDB.SetNonce(addr, uint64(100+1))
	bc.SetState(common.Hash{1}, newDB)
	pool.Reset(nil, &types.Header{Root: common.Hash{1}})
	// Verify post state
	if err := pool.verifyConsistency(); err != nil {
		t.Fatal(err)
	}
	pending, queued = pool.ContentFrom(addr)
	if len(pending) != 0 || len(queued) != 0 {
		t.Fatal(pending, queued)
	}
}

func BenchmarkReorg(b *testing.B) {
	// Generate a faucet
	key, err := crypto.GenerateKey()
	if err != nil {
		b.Fatal(err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)
	// Setup a mock blockchain
	bc := &MockBC{
		currentBlock: &types.Header{Root: common.Hash{}},
		dbs:          make(map[common.Hash]*state.StateDB),
	}
	initialDB, err := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	if err != nil {
		b.Fatal(err)
	}
	initialDB.CreateAccount(addr)
	initialDB.SetBalance(addr, big.NewInt(1_000_000_000_000_000_000))
	bc.SetState(common.Hash{}, initialDB)
	// Setup the txpool
	pool, err := NewLocalPool(bc, types.LatestSigner(params.AllDevChainProtocolChanges))
	if err != nil {
		b.Fatal(err)
	}
	if err := pool.Init(nil, &types.Header{Root: common.Hash{}, GasLimit: 200_000}, func(addr common.Address, reserve bool) error { return nil }); err != nil {
		b.Fatal(err)
	}
	// Queue transactions
	for i := 0; i < b.N; i++ {
		tx := types.NewTransaction(uint64(i), common.Address{}, new(big.Int), 100000, big.NewInt(1234), nil)
		signer := types.LatestSigner(params.AllDevChainProtocolChanges)
		signed, err := types.SignTx(tx, signer, key)
		if err != nil {
			b.Fatal(err)
		}
		if errs := pool.Add([]*types.Transaction{signed}, true, false); errs[0] != nil {
			b.Fatal(errs[0])
		}
	}
	newDB := initialDB.Copy()
	newDB.SetNonce(addr, uint64(b.N+1))
	bc.SetState(common.Hash{1}, newDB)
	b.ResetTimer()
	pool.Reset(nil, &types.Header{Root: common.Hash{1}})
}
