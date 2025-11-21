package ethapi

import (
    "context"
    "testing"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/rpc"
)

type dummyBackend struct{}
func (b *dummyBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (Block, error) { return &dummyBlock{}, nil }
func (b *dummyBackend) StateAt(root common.Hash) (StateDB, error) { return &dummyState{}, nil }

type dummyBlock struct{}
func (b *dummyBlock) Root() common.Hash { return common.Hash{} }

type dummyState struct{}
func (s *dummyState) GetState(addr common.Address, slot common.Hash) common.Hash {
    return common.HexToHash("0x1234")
}

func TestBatchGetStorageAt(t *testing.T) {
    api := &DebugAPI{b: &dummyBackend{}}
    reqs := []StorageBatchRequest{
        {Address: common.HexToAddress("0xabc"), Slots: []common.Hash{common.HexToHash("0x0"), common.HexToHash("0x1")}},
        {Address: common.HexToAddress("0xdef"), Slots: []common.Hash{common.HexToHash("0x0")}},
    }

    res, err := api.BatchGetStorageAt(context.Background(), reqs, "latest")
    if err != nil { t.Fatal(err) }

    if res["0xabc"]["0x0"] != "0x0000000000000000000000000000000000001234" {
        t.Errorf("unexpected value for 0xabc 0x0: %s", res["0xabc"]["0x0"])
    }
}
