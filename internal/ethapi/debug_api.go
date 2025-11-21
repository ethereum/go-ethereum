package ethapi

import (
    "context"
    "sync"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/rpc"
)

type StorageBatchRequest struct {
    Address common.Address `json:"address"`
    Slots   []common.Hash  `json:"slots"`
}

type StorageBatchResponse map[string]string

type DebugAPI struct {
    b Backend
}

type Backend interface {
    BlockByNumber(ctx context.Context, number rpc.BlockNumber) (Block, error)
    StateAt(root common.Hash) (StateDB, error)
}

type Block interface {
    Root() common.Hash
}

type StateDB interface {
    GetState(addr common.Address, slot common.Hash) common.Hash
}

func (api *DebugAPI) BatchGetStorageAt(ctx context.Context, reqs []StorageBatchRequest, blockNr rpc.BlockNumber) (map[string]StorageBatchResponse, error) {
    res := make(map[string]StorageBatchResponse)
    block, err := api.b.BlockByNumber(ctx, blockNr)
    if err != nil { return nil, err }

    statedb, err := api.b.StateAt(block.Root())
    if err != nil { return nil, err }

    var wg sync.WaitGroup
    var mu sync.Mutex

    for _, req := range reqs {
        wg.Add(1)
        go func(req StorageBatchRequest) {
            defer wg.Done()
            batchRes := make(StorageBatchResponse)
            for _, slot := range req.Slots {
                value := statedb.GetState(req.Address, slot)
                batchRes[slot.Hex()] = value.Hex()
            }
            mu.Lock()
            res[req.Address.Hex()] = batchRes
            mu.Unlock()
        }(req)
    }

    wg.Wait()
    return res, nil
}
