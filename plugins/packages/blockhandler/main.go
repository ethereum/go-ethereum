package main

import (
  "encoding/json"
  lru "github.com/hashicorp/golang-lru"
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/common/hexutil"
  "github.com/ethereum/go-ethereum/core/types"
  "github.com/ethereum/go-ethereum/events"
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
  blockEvents events.Feed
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
  Accounts  []kvpair
  Storage   []storage
}

func (su *stateUpdate) MarshalJSON() ([]byte, error) {
  result := make(map[string]interface{}, 0, len(su.Destructs))
  result["destructs"] = []common.Hash{}
  for k := range su.Destructs {
    result["destructs"] = append(result["destructs"], k)
  }
  accounts = make(map[common.Hash]hexutil.Bytes)
  for k, v := range su.Accounts {
    result["accounts"][k] = hexutil.Bytes(v)
  }
  result["accounts"] = accounts
  storage = make(map[common.Hash]map[common.Hash]hexutil.Bytes)
  for m, s := range su.Storage {
    storage[m] = make(map[common.Hash]hexutil.Bytes)
    for k, v := range s {
      storage[m][s] = hexutil.Bytes(v)
    }
  }
  result["storage"] = storage
  return json.Marshal(result)
}

func (su *stateUpdate) EncodeRLP(w io.Writer) error {
  destructs = make([]common.Hash, 0, len(su.Destructs))
  for k := range su.Destructs {
    destructs = append(destructs, k)
  }
  accounts := make([]kvpair, 0, len(accounts))
  for k, v := range su.Accounts {
    accounts = append(accounts, kvpair{k, v})
  }
  s := make([]storage, 0, len(storage))
  for a, m := range su.Storage {
    accountStorage = storage{a, make([]kvpair, 0, len(m))}
    for k, v := range m {
      accountStorage.Data = append(accountStorage.Data, []kvpair{k, v})
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

func Initialize(ctx *cli.Context, loader *PluginLoader) {
  pl = loader
  cache, _ = lru.New(128) // TODO: Make size configurable
  if !ctx.GlobalBool(utils.SnapshotFlag.Name) {
    log.Warn("Snapshots are required for StateUpdate plugins, but are currently disabled. State Updates will be unavailable")
  }
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
  su := stateUpdate{
    destructs: destructs,
    accounts: accounts,
    storage: storage,
  }
  cache.Add(blockRoot, su)
  data, _ := rlp.EncodeToBytes(su)
  backend.ChainDb().Put(append([]byte("su"), blockRoot...), data)
}

func AppendAncient(number uint64, hash, header, body, receipts, td []byte) {
  header := new(types.Header)
	if err := rlp.Decode(bytes.NewReader(data), header); err != nil {
    log.Warn("Could not decode ancient header", "block", number)
    return
  }
  backend.Chaindb().Delete(append([]byte("su"), header.Root()...))
}


func NewHead(block *types.Block, hash common.Hash, logs []*types.Log) {
  if pl == nil {
		log.Warn("Attempting to emit NewHead, but default PluginLoader has not been initialized")
    return
  }
  result, err := blockUpdates(block)
  if err != nil {
    log.Error("Could not serialize block", "err", err, "hash", block.Hash())
    return
  }
  blockEvents.Send(result)



  receipts, err = backend.GetReceipts(ctx, block.Hash()) (types.Receipts, error)
  var su *stateUpdate
  if v, ok := cache.Get(block.Root()); ok {
    su = v.(stateUpdate)
  }
  data, err := backend.ChainDb().Get(append([]byte("su"), block.Root()...))
  if err != nil {
    log.Error("StateUpdate unavailable for block", "hash", block.Hash())
    return
  }
  su = &stateUpdate{}
  if err := rlp.DecodeBytes(data, su); err != nil {
    log.Error("StateUpdate unavailable for block", "hash", block.Hash())
    return
  }
  //TODO: Get plugins to invoke, invoke them
  return result
}



type BlockUpdates struct{
  backend plugin.Backend
}

func blockUpdates(block *types.Block) (map[string]interface{}, error) {
  result := ethapi.RPCMarshalBlock(block, true, true)
  var (
    err error
  )
  result["receipts"], err = backend.GetReceipts(ctx, block.Hash()) (types.Receipts, error)
  if err != nil { return nil, err }
  if v, ok := cache.Get(block.Root); ok {
    result["stateUpdates"] = v
    return result
  }
  data, err := backend.ChainDb().Get(append([]byte("su"), block.Root()...))
  if err != nil { return nil, fmt.Errorf("State Updates unavailable for block %#x", block.Hash())}
  su := &stateUpdate{}
  if err := rlp.DecodeBytes(data, su); err != nil { return nil, fmt.Errorf("State updates unavailable for block %#x", block.Hash()) }
  result["stateUpdates"] = su
  return result
}

func (b *BlockUpdates) BlockUpdatesByNumber(ctx context.Context, number rpc.BlockNumber) (map[string]interface{}, error) {
  block := b.backend.BlockByNumber(ctx, number)
  return blockUpdates(block)
}

func (b *BlockUpdates) BlockUpdatesByHash(ctx context.Context, hash common.Hash) (map[string]interface{}, error) {
  block := b.backend.BlockByHash(ctx, hash)
  return blockUpdates(block)
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

  sub := blockEvents.Send(blockDataChan)
	if err != nil {
		return nil, err
	}

  go func() {

		for {
			select {
			case hash := <-hashCh:
        b.blockUpdates()
        notifier.Notify(rpcSub.ID,)
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



func GetAPIs(stack *node.Node, backend plugins.Backend) []rpc.API {
  return []rpc.API{
   {
     Namespace: "cardinal",
     Version:   "1.0",
     Service:   &BlockUpdates{backend},
     Public:    true,
   },
 }
}
