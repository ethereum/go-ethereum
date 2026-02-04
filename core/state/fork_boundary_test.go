package state

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie/transitiontrie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

func TestForkBoundaryOpenTrie(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(db, nil)
	sdb := NewDatabase(tdb, nil)

	statedb, err := New(types.EmptyRootHash, sdb)
	if err != nil {
		t.Fatalf("failed to create statedb: %v", err)
	}

	addr := common.HexToAddress("0x1234")
	statedb.SetBalance(addr, uint256.NewInt(1000), tracing.BalanceIncreaseGenesisBalance)
	statedb.SetNonce(addr, 1, tracing.NonceChangeGenesis)

	root, err := statedb.Commit(0, false, false)
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}
	if err := tdb.Commit(root, false); err != nil {
		t.Fatalf("failed to commit triedb: %v", err)
	}

	sdb.SetForkBoundary(root)
	defer sdb.ClearForkBoundary()

	tr, err := sdb.OpenTrie(root)
	if err != nil {
		t.Fatalf("failed to open trie: %v", err)
	}

	if _, ok := tr.(*transitiontrie.TransitionTrie); !ok {
		t.Fatalf("expected TransitionTrie at fork boundary, got %T", tr)
	}

	acct, err := tr.GetAccount(addr)
	if err != nil {
		t.Fatalf("failed to get account: %v", err)
	}
	if acct == nil {
		t.Fatal("expected non-nil account from MPT base")
	}
	if acct.Balance.Cmp(uint256.NewInt(1000)) != 0 {
		t.Fatalf("unexpected balance: got %v, want 1000", acct.Balance)
	}
}

func TestForkBoundaryStateReader(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(db, nil)
	sdb := NewDatabase(tdb, nil)

	statedb, err := New(types.EmptyRootHash, sdb)
	if err != nil {
		t.Fatalf("failed to create statedb: %v", err)
	}

	addr := common.HexToAddress("0x5678")
	statedb.SetBalance(addr, uint256.NewInt(2000), tracing.BalanceIncreaseGenesisBalance)

	root, err := statedb.Commit(0, false, false)
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}
	if err := tdb.Commit(root, false); err != nil {
		t.Fatalf("failed to commit triedb: %v", err)
	}

	sdb.SetForkBoundary(root)
	defer sdb.ClearForkBoundary()

	reader, err := sdb.StateReader(root)
	if err != nil {
		t.Fatalf("failed to create state reader: %v", err)
	}

	acct, err := reader.Account(addr)
	if err != nil {
		t.Fatalf("failed to get account: %v", err)
	}
	if acct == nil {
		t.Fatal("expected non-nil account")
	}
	if acct.Balance.Cmp(uint256.NewInt(2000)) != 0 {
		t.Fatalf("unexpected balance: got %v, want 2000", acct.Balance)
	}
}

func TestForkBoundaryWrite(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(db, nil)
	sdb := NewDatabase(tdb, nil)

	statedb, err := New(types.EmptyRootHash, sdb)
	if err != nil {
		t.Fatalf("failed to create statedb: %v", err)
	}

	addr := common.HexToAddress("0xabcd")
	statedb.SetBalance(addr, uint256.NewInt(500), tracing.BalanceIncreaseGenesisBalance)

	root, err := statedb.Commit(0, false, false)
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}
	if err := tdb.Commit(root, false); err != nil {
		t.Fatalf("failed to commit triedb: %v", err)
	}

	sdb.SetForkBoundary(root)
	defer sdb.ClearForkBoundary()

	tr, err := sdb.OpenTrie(root)
	if err != nil {
		t.Fatalf("failed to open trie: %v", err)
	}

	newAddr := common.HexToAddress("0xef01")
	err = tr.UpdateAccount(newAddr, &types.StateAccount{
		Nonce:    1,
		Balance:  uint256.NewInt(100),
		Root:     types.EmptyRootHash,
		CodeHash: types.EmptyCodeHash.Bytes(),
	}, 0)
	if err != nil {
		t.Fatalf("failed to update account: %v", err)
	}

	acct, err := tr.GetAccount(newAddr)
	if err != nil {
		t.Fatalf("failed to get new account: %v", err)
	}
	if acct == nil {
		t.Fatal("expected non-nil new account")
	}
	if acct.Balance.Cmp(uint256.NewInt(100)) != 0 {
		t.Fatalf("unexpected balance for new account: got %v, want 100", acct.Balance)
	}

	origAcct, err := tr.GetAccount(addr)
	if err != nil {
		t.Fatalf("failed to get original account: %v", err)
	}
	if origAcct == nil {
		t.Fatal("expected non-nil original account from MPT base")
	}
}

func TestForkBoundaryNotActive(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(db, nil)
	sdb := NewDatabase(tdb, nil)

	statedb, err := New(types.EmptyRootHash, sdb)
	if err != nil {
		t.Fatalf("failed to create statedb: %v", err)
	}

	addr := common.HexToAddress("0x1111")
	statedb.SetBalance(addr, uint256.NewInt(300), tracing.BalanceIncreaseGenesisBalance)

	root, err := statedb.Commit(0, false, false)
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}
	if err := tdb.Commit(root, false); err != nil {
		t.Fatalf("failed to commit triedb: %v", err)
	}

	tr, err := sdb.OpenTrie(root)
	if err != nil {
		t.Fatalf("failed to open trie: %v", err)
	}

	if _, ok := tr.(*transitiontrie.TransitionTrie); ok {
		t.Fatal("should not get TransitionTrie without fork boundary set")
	}
}

func TestForkBoundaryWrongRoot(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(db, nil)
	sdb := NewDatabase(tdb, nil)

	statedb, err := New(types.EmptyRootHash, sdb)
	if err != nil {
		t.Fatalf("failed to create statedb: %v", err)
	}

	addr := common.HexToAddress("0x2222")
	statedb.SetBalance(addr, uint256.NewInt(400), tracing.BalanceIncreaseGenesisBalance)

	root, err := statedb.Commit(0, false, false)
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}
	if err := tdb.Commit(root, false); err != nil {
		t.Fatalf("failed to commit triedb: %v", err)
	}

	otherRoot := common.HexToHash("0xdeadbeef")
	sdb.SetForkBoundary(otherRoot)
	defer sdb.ClearForkBoundary()

	tr, err := sdb.OpenTrie(root)
	if err != nil {
		t.Fatalf("failed to open trie: %v", err)
	}

	if _, ok := tr.(*transitiontrie.TransitionTrie); ok {
		t.Fatal("should not get TransitionTrie for different root")
	}
}
