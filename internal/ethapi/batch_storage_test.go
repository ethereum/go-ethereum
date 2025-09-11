package ethapi

import (
    "context"
    "testing"

    "github.com/ethereum/go-ethereum/accounts"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/common/hexutil"
    "github.com/ethereum/go-ethereum/core"
    "github.com/ethereum/go-ethereum/core/state"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/core/vm"
    "github.com/ethereum/go-ethereum/ethdb"
    "github.com/ethereum/go-ethereum/event"
    "github.com/ethereum/go-ethereum/params"
    "github.com/ethereum/go-ethereum/rpc"
)

// storageBackend composes backendMock and overrides StateAndHeaderByNumberOrHash
// to return a prepared in-memory StateDB for testing.
type storageBackend struct {
    *backendMock
    state *state.StateDB
}

func (b *storageBackend) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
    return b.state, nil, nil
}

// Ensure storageBackend still satisfies the Backend interface by forwarding
// the remaining methods via the embedded backendMock (compile-time check).
var _ Backend = (*storageBackend)(nil)

func TestDebugBatchGetStorage(t *testing.T) {
    t.Parallel()

    // Prepare an in-memory state database and set some storage values.
    sdb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())

    addr1 := common.HexToAddress("0x1000000000000000000000000000000000000001")
    addr2 := common.HexToAddress("0x2000000000000000000000000000000000000002")

    keyA := common.HexToHash("0x01")
    keyB := common.HexToHash("0x02")
    keyC := common.HexToHash("0x03")

    valA := common.HexToHash("0xaaa")
    valB := common.HexToHash("0xbbb")
    valC := common.HexToHash("0xccc")

    sdb.SetState(addr1, keyA, valA)
    sdb.SetState(addr1, keyB, valB)
    sdb.SetState(addr2, keyC, valC)

    // Wire the API with a backend that returns our state.
    base := newBackendMock()
    b := &storageBackend{backendMock: base, state: sdb}
    api := NewDebugAPI(b)

    // Build request: addr1 asks [keyB, keyA] (order test), addr2 asks [keyC, missing]
    req := map[common.Address][]string{
        addr1: {"0x02", "0x01"},
        addr2: {"0x03", "0x04"},
    }

    got, err := api.BatchGetStorage(context.Background(), req, rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber))
    if err != nil {
        t.Fatalf("BatchGetStorage returned error: %v", err)
    }

    // Validate addr1 order and values
    if len(got[addr1]) != 2 {
        t.Fatalf("addr1 length mismatch: got %d want 2", len(got[addr1]))
    }
    if hexutil.Bytes(got[addr1][0]).String() != hexutil.Bytes(valB[:]).String() {
        t.Fatalf("addr1[0] mismatch: got %s want %s", hexutil.Bytes(got[addr1][0]).String(), hexutil.Bytes(valB[:]).String())
    }
    if hexutil.Bytes(got[addr1][1]).String() != hexutil.Bytes(valA[:]).String() {
        t.Fatalf("addr1[1] mismatch: got %s want %s", hexutil.Bytes(got[addr1][1]).String(), hexutil.Bytes(valA[:]).String())
    }

    // Validate addr2 values (existing and zero for missing)
    if len(got[addr2]) != 2 {
        t.Fatalf("addr2 length mismatch: got %d want 2", len(got[addr2]))
    }
    if hexutil.Bytes(got[addr2][0]).String() != hexutil.Bytes(valC[:]).String() {
        t.Fatalf("addr2[0] mismatch: got %s want %s", hexutil.Bytes(got[addr2][0]).String(), hexutil.Bytes(valC[:]).String())
    }
    if hexutil.Bytes(got[addr2][1]).String() != hexutil.Bytes(common.Hash{}[:]).String() {
        t.Fatalf("addr2[1] mismatch: got %s want %s (zero)", hexutil.Bytes(got[addr2][1]).String(), hexutil.Bytes(common.Hash{}[:]).String())
    }
}

// Ensure backendMock compiles in this file by referencing needed imports
// (no-ops; prevents unused import errors if interface evolves).
var (
    _ = accounts.Account{}
    _ = core.ChainEvent{}
    _ vm.Config
    _ ethdb.Database
    _ event.Subscription
    _ params.ChainConfig
)

