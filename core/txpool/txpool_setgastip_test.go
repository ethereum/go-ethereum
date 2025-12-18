package txpool

import (
    "math/big"
    "sync"
    "testing"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/event"
)

// mockSubPool is a lightweight test double implementing the SubPool interface
// used to observe SetGasTip invocations.
type mockSubPool struct {
    mu        sync.Mutex
    callCount int
    lastTip   *big.Int
}

func (m *mockSubPool) Filter(tx *types.Transaction) bool                       { return false }
func (m *mockSubPool) FilterType(kind byte) bool                              { return false }
func (m *mockSubPool) Init(gasTip uint64, head *types.Header, reserver Reserver) error {
    return nil
}
func (m *mockSubPool) Close() error                                            { return nil }
func (m *mockSubPool) Reset(oldHead, newHead *types.Header)                   {}
func (m *mockSubPool) SetGasTip(tip *big.Int) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.callCount++
    if tip == nil {
        m.lastTip = nil
        return
    }
    // copy value
    m.lastTip = new(big.Int).Set(tip)
}
func (m *mockSubPool) Has(hash common.Hash) bool                               { return false }
func (m *mockSubPool) Get(hash common.Hash) *types.Transaction                { return nil }
func (m *mockSubPool) GetRLP(hash common.Hash) []byte                         { return nil }
func (m *mockSubPool) GetMetadata(hash common.Hash) *TxMetadata               { return nil }
func (m *mockSubPool) ValidateTxBasics(tx *types.Transaction) error           { return nil }
func (m *mockSubPool) Add(txs []*types.Transaction, sync bool) []error        { return nil }
func (m *mockSubPool) Pending(filter PendingFilter) map[common.Address][]*LazyTransaction {
    return nil
}
func (m *mockSubPool) SubscribeTransactions(ch chan<- core.NewTxsEvent, reorgs bool) event.Subscription {
    return nil
}
func (m *mockSubPool) Nonce(addr common.Address) uint64                       { return 0 }
func (m *mockSubPool) Stats() (int, int)                                      { return 0, 0 }
func (m *mockSubPool) Content() (map[common.Address][]*types.Transaction, map[common.Address][]*types.Transaction) {
    return nil, nil
}
func (m *mockSubPool) ContentFrom(addr common.Address) ([]*types.Transaction, []*types.Transaction) {
    return nil, nil
}
func (m *mockSubPool) Status(hash common.Hash) TxStatus                       { return TxStatusUnknown }
func (m *mockSubPool) Clear()                                                  {}

func TestSetGasTip(t *testing.T) {
    tests := []struct {
        name          string
        tip           *big.Int
        expectInvokes int
    }{
        {"nil tip", nil, 0},
        {"zero tip", big.NewInt(0), 0},
        {"negative tip", big.NewInt(-1), 0},
        {"very high tip", big.NewInt(1_000_000_000_000_001), 2},
        {"valid tip", big.NewInt(5), 2},
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            m1 := &mockSubPool{}
            m2 := &mockSubPool{}
            p := &TxPool{
                subpools: []SubPool{m1, m2},
            }
            // call SetGasTip under test
            p.SetGasTip(tc.tip)

            // verify invocation counts
            m1.mu.Lock()
            got1 := m1.callCount
            m1.mu.Unlock()
            m2.mu.Lock()
            got2 := m2.callCount
            m2.mu.Unlock()

            if got1+got2 != tc.expectInvokes {
                t.Fatalf("unexpected total SetGasTip invocations: got %d want %d", got1+got2, tc.expectInvokes)
            }

            // when invoked, ensure the lastTip equals the passed tip (for non-nil)
            if tc.tip != nil && tc.expectInvokes > 0 {
                // pick one mock to assert
                m1.mu.Lock()
                if m1.lastTip == nil {
                    t.Fatalf("expected lastTip set on subpool but was nil")
                }
                if m1.lastTip.Cmp(tc.tip) != 0 {
                    t.Fatalf("lastTip mismatch: got %v want %v", m1.lastTip, tc.tip)
                }
                m1.mu.Unlock()
            }
        })
    }
}
