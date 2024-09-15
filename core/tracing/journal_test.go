package tracing

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

type testTracer struct {
	bal   *big.Int
	nonce uint64
}

func (t *testTracer) OnBalanceChange(addr common.Address, prev *big.Int, new *big.Int, reason BalanceChangeReason) {
	t.bal = new
}

func (t *testTracer) OnNonceChange(addr common.Address, prev uint64, new uint64) {
	t.nonce = new
}

func TestJournalIntegration(t *testing.T) {
	tr := &testTracer{}
	wr, err := WrapWithJournal(&Hooks{OnBalanceChange: tr.OnBalanceChange, OnNonceChange: tr.OnNonceChange})
	if err != nil {
		t.Fatalf("failed to wrap test tracer: %v", err)
	}
	addr := common.HexToAddress("0x1234")
	wr.OnEnter(0, 0, addr, addr, nil, 1000, big.NewInt(0))
	wr.OnBalanceChange(addr, nil, big.NewInt(100), BalanceChangeUnspecified)
	wr.OnEnter(1, 0, addr, addr, nil, 1000, big.NewInt(0))
	wr.OnNonceChange(addr, 0, 1)
	wr.OnBalanceChange(addr, big.NewInt(100), big.NewInt(200), BalanceChangeUnspecified)
	wr.OnBalanceChange(addr, big.NewInt(200), big.NewInt(250), BalanceChangeUnspecified)
	wr.OnExit(0, nil, 100, errors.New("revert"), true)
	wr.OnExit(0, nil, 150, nil, false)
	if tr.bal.Cmp(big.NewInt(100)) != 0 {
		t.Fatalf("unexpected balance: %v", tr.bal)
	}
	if tr.nonce != 0 {
		t.Fatalf("unexpected nonce: %v", tr.nonce)
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
