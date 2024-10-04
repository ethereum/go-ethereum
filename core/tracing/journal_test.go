package tracing

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

type testTracer struct {
	bal     *big.Int
	nonce   uint64
	code    []byte
	storage map[common.Hash]common.Hash
}

func (t *testTracer) OnBalanceChange(addr common.Address, prev *big.Int, new *big.Int, reason BalanceChangeReason) {
	t.bal = new
}

func (t *testTracer) OnNonceChange(addr common.Address, prev uint64, new uint64) {
	t.nonce = new
}

func (t *testTracer) OnCodeChange(addr common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte) {
	t.code = code
}

func (t *testTracer) OnStorageChange(addr common.Address, slot common.Hash, prev common.Hash, new common.Hash) {
	if t.storage == nil {
		t.storage = make(map[common.Hash]common.Hash)
	}
	if new == (common.Hash{}) {
		delete(t.storage, slot)
	} else {
		t.storage[slot] = new
	}
}

func TestJournalIntegration(t *testing.T) {
	tr := &testTracer{}
	wr, err := WrapWithJournal(&Hooks{OnBalanceChange: tr.OnBalanceChange, OnNonceChange: tr.OnNonceChange, OnCodeChange: tr.OnCodeChange, OnStorageChange: tr.OnStorageChange})
	if err != nil {
		t.Fatalf("failed to wrap test tracer: %v", err)
	}
	addr := common.HexToAddress("0x1234")
	wr.OnEnter(0, 0, addr, addr, nil, 1000, big.NewInt(0))
	wr.OnBalanceChange(addr, nil, big.NewInt(100), BalanceChangeUnspecified)
	wr.OnCodeChange(addr, common.Hash{}, nil, common.Hash{}, []byte{1, 2, 3})
	wr.OnStorageChange(addr, common.Hash{1}, common.Hash{}, common.Hash{2})
	wr.OnEnter(1, 0, addr, addr, nil, 1000, big.NewInt(0))
	wr.OnNonceChange(addr, 0, 1)
	wr.OnBalanceChange(addr, big.NewInt(100), big.NewInt(200), BalanceChangeUnspecified)
	wr.OnBalanceChange(addr, big.NewInt(200), big.NewInt(250), BalanceChangeUnspecified)
	wr.OnStorageChange(addr, common.Hash{1}, common.Hash{2}, common.Hash{3})
	wr.OnStorageChange(addr, common.Hash{2}, common.Hash{}, common.Hash{4})
	wr.OnExit(1, nil, 100, errors.New("revert"), true)
	wr.OnExit(0, nil, 150, nil, false)
	if tr.bal.Cmp(big.NewInt(100)) != 0 {
		t.Fatalf("unexpected balance: %v", tr.bal)
	}
	if tr.nonce != 0 {
		t.Fatalf("unexpected nonce: %v", tr.nonce)
	}
	if len(tr.code) != 3 {
		t.Fatalf("unexpected code: %v", tr.code)
	}
	if len(tr.storage) != 1 {
		t.Fatalf("unexpected storage len. want %d, have %d", 1, len(tr.storage))
	}
	if tr.storage[common.Hash{1}] != (common.Hash{2}) {
		t.Fatalf("unexpected storage. want %v, have %v", common.Hash{2}, tr.storage[common.Hash{1}])
	}
}

func TestJournalTopRevert(t *testing.T) {
	tr := &testTracer{}
	wr, err := WrapWithJournal(&Hooks{OnBalanceChange: tr.OnBalanceChange, OnNonceChange: tr.OnNonceChange})
	if err != nil {
		t.Fatalf("failed to wrap test tracer: %v", err)
	}
	addr := common.HexToAddress("0x1234")
	wr.OnEnter(0, 0, addr, addr, nil, 1000, big.NewInt(0))
	wr.OnBalanceChange(addr, big.NewInt(0), big.NewInt(100), BalanceChangeUnspecified)
	wr.OnEnter(1, 0, addr, addr, nil, 1000, big.NewInt(0))
	wr.OnNonceChange(addr, 0, 1)
	wr.OnBalanceChange(addr, big.NewInt(100), big.NewInt(200), BalanceChangeUnspecified)
	wr.OnBalanceChange(addr, big.NewInt(200), big.NewInt(250), BalanceChangeUnspecified)
	wr.OnExit(0, nil, 100, errors.New("revert"), true)
	wr.OnExit(0, nil, 150, errors.New("revert"), true)
	if tr.bal.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("unexpected balance: %v", tr.bal)
	}
	if tr.nonce != 0 {
		t.Fatalf("unexpected nonce: %v", tr.nonce)
	}
}
