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


// stateUpdate will be used to track state updates
type stateUpdate struct {
	Destructs map[common.Hash]struct{}
	Accounts map[common.Hash][]byte
	Storage map[common.Hash]map[common.Hash][]byte
}


// kvpair is used for RLP encoding of maps, as maps cannot be RLP encoded directly
type kvpair struct {
	Key common.Hash
	Value []byte
}

// storage is used for RLP encoding two layers of maps, as maps cannot be RLP encoded directly
type storage struct {
	Account common.Hash
	Data []kvpair
}

// storedStateUpdate is an RLP encodable version of stateUpdate
type storedStateUpdate struct {
	Destructs []common.Hash
	Accounts	[]kvpair
	Storage	 []storage
}


// MarshalJSON represents the stateUpdate as JSON for RPC calls
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

// EncodeRLP converts the stateUpdate to a storedStateUpdate, and RLP encodes the result for storage
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

// DecodeRLP takes a byte stream, decodes it to a storedStateUpdate, the n converts that into a stateUpdate object
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


// Initialize does initial setup of variables as the plugin is loaded.
func Initialize(ctx *cli.Context, loader *plugins.PluginLoader) {
	pl = loader
	cache, _ = lru.New(128) // TODO: Make size configurable
	if !ctx.GlobalBool(utils.SnapshotFlag.Name) {
		log.Warn("Snapshots are required for StateUpdate plugins, but are currently disabled. State Updates will be unavailable")
	}
	log.Info("Loaded block updater plugin")
}


// InitializeNode is invoked by the plugin loader when the node and Backend are
// ready. We will track the backend to provide access to blocks and other
// useful information.
func InitializeNode(stack *node.Node, b interfaces.Backend) {
	backend = b
}


// StateUpdate gives us updates about state changes made in each block. We
// cache them for short term use, and write them to disk for the longer term.
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

// AppendAncient removes our state update records from leveldb as the
// corresponding blocks are moved from leveldb to the ancients database. At
// some point in the future, we may want to look at a way to move the state
// updates to an ancients table of their own for longer term retention.
func AppendAncient(number uint64, hash, headerBytes, body, receipts, td []byte) {
	header := new(types.Header)
	if err := rlp.Decode(bytes.NewReader(headerBytes), header); err != nil {
		log.Warn("Could not decode ancient header", "block", number)
		return
	}
	backend.ChainDb().Delete(append([]byte("su"), header.Root.Bytes()...))
}


// NewHead is invoked when a new block becomes the latest recognized block. We
// use this to notify the blockEvents channel of new blocks, as well as invoke
// the BlockUpdates hook on downstream plugins.
// TODO: We're not necessarily handling reorgs properly, which may result in
// some blocks not being emitted through this hook.
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
	} else {
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
}


// BlockUpdates is a service that lets clients query for block updates for a
// given block by hash or number, or subscribe to new block upates.
type BlockUpdates struct{
	backend interfaces.Backend
}

// blockUpdate handles the serialization of a block
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

// BlockUpdatesByNumber retrieves a block by number, gets receipts and state
// updates, and serializes the response.
func (b *BlockUpdates) BlockUpdatesByNumber(ctx context.Context, number rpc.BlockNumber) (map[string]interface{}, error) {
	block, err := b.backend.BlockByNumber(ctx, number)
	if err != nil { return nil, err }
	return blockUpdates(ctx, block)
}

// BlockUpdatesByHash retrieves a block by hash, gets receipts and state
// updates, and serializes the response.
func (b *BlockUpdates) BlockUpdatesByHash(ctx context.Context, hash common.Hash) (map[string]interface{}, error) {
	block, err := b.backend.BlockByHash(ctx, hash)
	if err != nil { return nil, err }
	return blockUpdates(ctx, block)
}

// BlockUpdates allows clients to subscribe to notifications of new blocks
// along with receipts and state updates.
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


// GetAPIs exposes the BlockUpdates service under the cardinal namespace.
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
