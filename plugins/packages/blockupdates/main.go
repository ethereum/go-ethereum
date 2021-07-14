package main

import (
	"bytes"
	"fmt"
	"context"
	"encoding/json"
	lru "github.com/hashicorp/golang-lru"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/plugins"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/plugins/interfaces"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"gopkg.in/urfave/cli.v1"
	"io"
)


var (
	pl *plugins.PluginLoader
	backend interfaces.Backend
	lastBlock common.Hash
	cache *lru.Cache
	blockEvents event.Feed
)

type stateUpdate struct {
	Destructs map[common.Hash]struct{}
	Accounts map[common.Hash][]byte
	Storage map[common.Hash]map[common.Hash][]byte
}

type kvpair struct {
	Key common.Hash
	Value []byte
}

type storage struct {
	Account common.Hash
	Data []kvpair
}

type storedStateUpdate struct {
	Destructs []common.Hash
	Accounts	[]kvpair
	Storage	 []storage
}

func (su *stateUpdate) MarshalJSON() ([]byte, error) {
	result := make(map[string]interface{})
	destructs := make([]common.Hash, 0, len(su.Destructs))
	for k := range su.Destructs {
		destructs = append(destructs, k)
	}
	result["destructs"] = destructs
	accounts := make(map[common.Hash]hexutil.Bytes)
	for k, v := range su.Accounts {
		accounts[k] = hexutil.Bytes(v)
	}
	result["accounts"] = accounts
	storage := make(map[common.Hash]map[common.Hash]hexutil.Bytes)
	for m, s := range su.Storage {
		storage[m] = make(map[common.Hash]hexutil.Bytes)
		for k, v := range s {
			storage[m][k] = hexutil.Bytes(v)
		}
	}
	result["storage"] = storage
	return json.Marshal(result)
}

func (su *stateUpdate) EncodeRLP(w io.Writer) error {
	destructs := make([]common.Hash, 0, len(su.Destructs))
	for k := range su.Destructs {
		destructs = append(destructs, k)
	}
	accounts := make([]kvpair, 0, len(su.Accounts))
	for k, v := range su.Accounts {
		accounts = append(accounts, kvpair{k, v})
	}
	s := make([]storage, 0, len(su.Storage))
	for a, m := range su.Storage {
		accountStorage := storage{a, make([]kvpair, 0, len(m))}
		for k, v := range m {
			accountStorage.Data = append(accountStorage.Data, kvpair{k, v})
		}
		s = append(s, accountStorage)
	}
	return rlp.Encode(w, storedStateUpdate{destructs, accounts, s})
}

func (su *stateUpdate) DecodeRLP(s *rlp.Stream) error {
	ssu := storedStateUpdate{}
	if err := s.Decode(&ssu); err != nil { return err }
	su.Destructs = make(map[common.Hash]struct{})
	for _, s := range ssu.Destructs {
		su.Destructs[s] = struct{}{}
	}
	su.Accounts = make(map[common.Hash][]byte)
	for _, kv := range ssu.Accounts {
		su.Accounts[kv.Key] = kv.Value
	}
	su.Storage = make(map[common.Hash]map[common.Hash][]byte)
	for _, s := range ssu.Storage {
		su.Storage[s.Account] = make(map[common.Hash][]byte)
		for _, kv := range s.Data {
			su.Storage[s.Account][kv.Key] = kv.Value
		}
	}
	return nil
}

func Initialize(ctx *cli.Context, loader *plugins.PluginLoader) {
	pl = loader
	cache, _ = lru.New(128) // TODO: Make size configurable
	if !ctx.GlobalBool(utils.SnapshotFlag.Name) {
		log.Warn("Snapshots are required for StateUpdate plugins, but are currently disabled. State Updates will be unavailable")
	}
	log.Info("Loaded block updater plugin")
}


// TODO:
// x Record StateUpdates in the database as they are written.
// x Keep an LRU cache of the most recent state updates.
// x Prune the StateUpdates in the database as corresponding blocks move to the freezer.
// * Add an RPC endpoint for getting the StateUpdates of a block
// * Invoke other plugins with each block and its respective StateUpdates.


func InitializeNode(stack *node.Node, b interfaces.Backend) {
	backend = b
}

func StateUpdate(blockRoot common.Hash, parentRoot common.Hash, destructs map[common.Hash]struct{}, accounts map[common.Hash][]byte, storage map[common.Hash]map[common.Hash][]byte) {
	su := &stateUpdate{
		Destructs: destructs,
		Accounts: accounts,
		Storage: storage,
	}
	cache.Add(blockRoot, su)
	data, _ := rlp.EncodeToBytes(su)
	backend.ChainDb().Put(append([]byte("su"), blockRoot.Bytes()...), data)
}

func AppendAncient(number uint64, hash, headerBytes, body, receipts, td []byte) {
	header := new(types.Header)
	if err := rlp.Decode(bytes.NewReader(headerBytes), header); err != nil {
		log.Warn("Could not decode ancient header", "block", number)
		return
	}
	backend.ChainDb().Delete(append([]byte("su"), header.Root.Bytes()...))
}


func NewHead(block *types.Block, hash common.Hash, logs []*types.Log) {
	if pl == nil {
		log.Warn("Attempting to emit NewHead, but default PluginLoader has not been initialized")
		return
	}
	result, err := blockUpdates(context.Background(), block)
	if err != nil {
		log.Error("Could not serialize block", "err", err, "hash", block.Hash())
		return
	}
	blockEvents.Send(result)

	receipts, err := backend.GetReceipts(context.Background(), block.Hash())
	var su *stateUpdate
	if v, ok := cache.Get(block.Root()); ok {
		su = v.(*stateUpdate)
	}
	data, err := backend.ChainDb().Get(append([]byte("su"), block.Root().Bytes()...))
	if err != nil {
		log.Error("StateUpdate unavailable for block", "hash", block.Hash())
		return
	}
	su = &stateUpdate{}
	if err := rlp.DecodeBytes(data, su); err != nil {
		log.Error("StateUpdate unavailable for block", "hash", block.Hash())
		return
	}
	fnList := pl.Lookup("BlockUpdates", func(item interface{}) bool {
    _, ok := item.(func(*types.Block, []*types.Log, types.Receipts, map[common.Hash]struct{}, map[common.Hash][]byte, map[common.Hash]map[common.Hash][]byte))
    return ok
  })
  for _, fni := range fnList {
    if fn, ok := fni.(func(*types.Block, []*types.Log, types.Receipts, map[common.Hash]struct{}, map[common.Hash][]byte, map[common.Hash]map[common.Hash][]byte)); ok {
      fn(block, logs, receipts, su.Destructs, su.Accounts, su.Storage)
    }
  }
	//TODO: Get plugins to invoke, invoke them
}



type BlockUpdates struct{
	backend interfaces.Backend
}

func blockUpdates(ctx context.Context, block *types.Block) (map[string]interface{}, error)	{
	result, err := ethapi.RPCMarshalBlock(block, true, true)
	if err != nil { return nil, err }
	result["receipts"], err = backend.GetReceipts(ctx, block.Hash())
	if err != nil { return nil, err }
	if v, ok := cache.Get(block.Root()); ok {
		result["stateUpdates"] = v
		return result, nil
	}
	data, err := backend.ChainDb().Get(append([]byte("su"), block.Root().Bytes()...))
	if err != nil { return nil, fmt.Errorf("State Updates unavailable for block %#x", block.Hash())}
	su := &stateUpdate{}
	if err := rlp.DecodeBytes(data, su); err != nil { return nil, fmt.Errorf("State updates unavailable for block %#x", block.Hash()) }
	result["stateUpdates"] = su
	return result, nil
}

func (b *BlockUpdates) BlockUpdatesByNumber(ctx context.Context, number rpc.BlockNumber) (map[string]interface{}, error) {
	block, err := b.backend.BlockByNumber(ctx, number)
	if err != nil { return nil, err }
	return blockUpdates(ctx, block)
}

func (b *BlockUpdates) BlockUpdatesByHash(ctx context.Context, hash common.Hash) (map[string]interface{}, error) {
	block, err := b.backend.BlockByHash(ctx, hash)
	if err != nil { return nil, err }
	return blockUpdates(ctx, block)
}

func (b *BlockUpdates) BlockUpdates(ctx context.Context) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	var (
		rpcSub = notifier.CreateSubscription()
		blockDataChan = make(chan map[string]interface{}, 1000)
	)

	sub := blockEvents.Subscribe(blockDataChan)
	go func() {

		for {
			select {
			case b := <-blockDataChan:
				notifier.Notify(rpcSub.ID, b)
			case <-rpcSub.Err(): // client send an unsubscribe request
				sub.Unsubscribe()
				return
			case <-notifier.Closed(): // connection dropped
				sub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}



func GetAPIs(stack *node.Node, backend interfaces.Backend) []rpc.API {
	return []rpc.API{
	 {
		 Namespace: "cardinal",
		 Version:	 "1.0",
		 Service:	 &BlockUpdates{backend},
		 Public:		true,
	 },
 }
}
