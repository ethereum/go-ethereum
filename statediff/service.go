// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package statediff

import (
	"bytes"
	"math/big"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"

	ind "github.com/ethereum/go-ethereum/statediff/indexer"
	nodeinfo "github.com/ethereum/go-ethereum/statediff/indexer/node"
	"github.com/ethereum/go-ethereum/statediff/indexer/postgres"
	"github.com/ethereum/go-ethereum/statediff/indexer/prom"
	. "github.com/ethereum/go-ethereum/statediff/types"
)

const chainEventChanSize = 20000

var writeLoopParams = Params{
	IntermediateStateNodes:   true,
	IntermediateStorageNodes: true,
	IncludeBlock:             true,
	IncludeReceipts:          true,
	IncludeTD:                true,
	IncludeCode:              true,
}

type blockChain interface {
	SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription
	GetBlockByHash(hash common.Hash) *types.Block
	GetBlockByNumber(number uint64) *types.Block
	GetReceiptsByHash(hash common.Hash) types.Receipts
	GetTdByHash(hash common.Hash) *big.Int
	UnlockTrie(root common.Hash)
	StateCache() state.Database
}

// IService is the state-diffing service interface
type IService interface {
	// Start() and Stop()
	node.Lifecycle
	// Method to getting API(s) for this service
	APIs() []rpc.API
	// Main event loop for processing state diffs
	Loop(chainEventCh chan core.ChainEvent)
	// Method to subscribe to receive state diff processing output
	Subscribe(id rpc.ID, sub chan<- Payload, quitChanogr chan<- bool, params Params)
	// Method to unsubscribe from state diff processing
	Unsubscribe(id rpc.ID) error
	// Method to get state diff object at specific block
	StateDiffAt(blockNumber uint64, params Params) (*Payload, error)
	// Method to get state trie object at specific block
	StateTrieAt(blockNumber uint64, params Params) (*Payload, error)
	// Method to stream out all code and codehash pairs
	StreamCodeAndCodeHash(blockNumber uint64, outChan chan<- CodeAndCodeHash, quitChan chan<- bool)
	// Method to write state diff object directly to DB
	WriteStateDiffAt(blockNumber uint64, params Params) error
	// Event loop for progressively processing and writing diffs directly to DB
	WriteLoop(chainEventCh chan core.ChainEvent)
}

// Service is the underlying struct for the state diffing service
type Service struct {
	// Used to sync access to the Subscriptions
	sync.Mutex
	// Used to build the state diff objects
	Builder Builder
	// Used to subscribe to chain events (blocks)
	BlockChain blockChain
	// Used to signal shutdown of the service
	QuitChan chan bool
	// A mapping of rpc.IDs to their subscription channels, mapped to their subscription type (hash of the Params rlp)
	Subscriptions map[common.Hash]map[rpc.ID]Subscription
	// A mapping of subscription params rlp hash to the corresponding subscription params
	SubscriptionTypes map[common.Hash]Params
	// Cache the last block so that we can avoid having to lookup the next block's parent
	lastBlock lastBlockCache
	// Whether or not we have any subscribers; only if we do, do we processes state diffs
	subscribers int32
	// Interface for publishing statediffs as PG-IPLD objects
	indexer ind.Indexer
	// Whether to enable writing state diffs directly to track blochain head
	enableWriteLoop bool
}

// Wrap the cached last block for safe access from different service loops
type lastBlockCache struct {
	sync.Mutex
	block *types.Block
}

// New creates a new statediff.Service
func New(stack *node.Node, ethServ *eth.Ethereum, dbParams *DBParams, enableWriteLoop bool) error {
	blockChain := ethServ.BlockChain()
	var indexer ind.Indexer
	if dbParams != nil {
		info := nodeinfo.Info{
			GenesisBlock: blockChain.Genesis().Hash().Hex(),
			NetworkID:    strconv.FormatUint(ethServ.NetVersion(), 10),
			ChainID:      blockChain.Config().ChainID.Uint64(),
			ID:           dbParams.ID,
			ClientName:   dbParams.ClientName,
		}

		// TODO: pass max idle, open, lifetime?
		db, err := postgres.NewDB(dbParams.ConnectionURL, postgres.ConnectionConfig{}, info)
		if err != nil {
			return err
		}
		indexer = ind.NewStateDiffIndexer(blockChain.Config(), db)
	}
	prom.Init()
	sds := &Service{
		Mutex:             sync.Mutex{},
		BlockChain:        blockChain,
		Builder:           NewBuilder(blockChain.StateCache()),
		QuitChan:          make(chan bool),
		Subscriptions:     make(map[common.Hash]map[rpc.ID]Subscription),
		SubscriptionTypes: make(map[common.Hash]Params),
		indexer:           indexer,
		enableWriteLoop:   enableWriteLoop,
	}
	stack.RegisterLifecycle(sds)
	stack.RegisterAPIs(sds.APIs())
	return nil
}

// Protocols exports the services p2p protocols, this service has none
func (sds *Service) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns the RPC descriptors the statediff.Service offers
func (sds *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: APIName,
			Version:   APIVersion,
			Service:   NewPublicStateDiffAPI(sds),
			Public:    true,
		},
	}
}

func (lbc *lastBlockCache) replace(currentBlock *types.Block, bc blockChain) *types.Block {
	lbc.Lock()
	parentHash := currentBlock.ParentHash()
	var parentBlock *types.Block
	if lbc.block != nil && bytes.Equal(lbc.block.Hash().Bytes(), parentHash.Bytes()) {
		parentBlock = lbc.block
	} else {
		parentBlock = bc.GetBlockByHash(parentHash)
	}
	lbc.block = currentBlock
	lbc.Unlock()
	return parentBlock
}

func (sds *Service) WriteLoop(chainEventCh chan core.ChainEvent) {
	chainEventSub := sds.BlockChain.SubscribeChainEvent(chainEventCh)
	defer chainEventSub.Unsubscribe()
	errCh := chainEventSub.Err()
	for {
		select {
		//Notify chain event channel of events
		case chainEvent := <-chainEventCh:
			log.Debug("(WriteLoop) Event received from chainEventCh", "event", chainEvent)
			currentBlock := chainEvent.Block
			parentBlock := sds.lastBlock.replace(currentBlock, sds.BlockChain)
			if parentBlock == nil {
				log.Error("Parent block is nil, skipping this block", "block height", currentBlock.Number())
				continue
			}
			err := sds.writeStateDiff(currentBlock, parentBlock.Root(), writeLoopParams)
			if err != nil {
				log.Error("statediff (DB write) processing error", "block height", currentBlock.Number().Uint64(), "error", err.Error())
				continue
			}
		case err := <-errCh:
			log.Warn("Error from chain event subscription", "error", err)
			sds.close()
			return
		case <-sds.QuitChan:
			log.Info("Quitting the statediff writing process")
			sds.close()
			return
		}
	}
}

// Loop is the main processing method
func (sds *Service) Loop(chainEventCh chan core.ChainEvent) {
	chainEventSub := sds.BlockChain.SubscribeChainEvent(chainEventCh)
	defer chainEventSub.Unsubscribe()
	errCh := chainEventSub.Err()
	for {
		select {
		//Notify chain event channel of events
		case chainEvent := <-chainEventCh:
			log.Debug("Event received from chainEventCh", "event", chainEvent)
			// if we don't have any subscribers, do not process a statediff
			if atomic.LoadInt32(&sds.subscribers) == 0 {
				log.Debug("Currently no subscribers to the statediffing service; processing is halted")
				continue
			}
			currentBlock := chainEvent.Block
			parentBlock := sds.lastBlock.replace(currentBlock, sds.BlockChain)
			if parentBlock == nil {
				log.Error("Parent block is nil, skipping this block", "number", currentBlock.Number())
				continue
			}
			sds.streamStateDiff(currentBlock, parentBlock.Root())
		case err := <-errCh:
			log.Warn("Error from chain event subscription", "error", err)
			sds.close()
			return
		case <-sds.QuitChan:
			log.Info("Quitting the statediffing process")
			sds.close()
			return
		}
	}
}

// streamStateDiff method builds the state diff payload for each subscription according to their subscription type and sends them the result
func (sds *Service) streamStateDiff(currentBlock *types.Block, parentRoot common.Hash) {
	sds.Lock()
	for ty, subs := range sds.Subscriptions {
		params, ok := sds.SubscriptionTypes[ty]
		if !ok {
			log.Error("no parameter set associated with this subscription", "subscription type", ty.Hex())
			sds.closeType(ty)
			continue
		}
		// create payload for this subscription type
		payload, err := sds.processStateDiff(currentBlock, parentRoot, params)
		if err != nil {
			log.Error("statediff processing error", "block height", currentBlock.Number().Uint64(), "parameters", params, "error", err.Error())
			continue
		}
		for id, sub := range subs {
			select {
			case sub.PayloadChan <- *payload:
				log.Debug("sending statediff payload at head", "height", currentBlock.Number(), "subscription id", id)
			default:
				log.Info("unable to send statediff payload; channel has no receiver", "subscription id", id)
			}
		}
	}
	sds.Unlock()
}

// StateDiffAt returns a state diff object payload at the specific blockheight
// This operation cannot be performed back past the point of db pruning; it requires an archival node for historical data
func (sds *Service) StateDiffAt(blockNumber uint64, params Params) (*Payload, error) {
	currentBlock := sds.BlockChain.GetBlockByNumber(blockNumber)
	log.Info("sending state diff", "block height", blockNumber)
	if blockNumber == 0 {
		return sds.processStateDiff(currentBlock, common.Hash{}, params)
	}
	parentBlock := sds.BlockChain.GetBlockByHash(currentBlock.ParentHash())
	return sds.processStateDiff(currentBlock, parentBlock.Root(), params)
}

// processStateDiff method builds the state diff payload from the current block, parent state root, and provided params
func (sds *Service) processStateDiff(currentBlock *types.Block, parentRoot common.Hash, params Params) (*Payload, error) {
	stateDiff, err := sds.Builder.BuildStateDiffObject(Args{
		NewStateRoot: currentBlock.Root(),
		OldStateRoot: parentRoot,
		BlockHash:    currentBlock.Hash(),
		BlockNumber:  currentBlock.Number(),
	}, params)
	// allow dereferencing of parent, keep current locked as it should be the next parent
	sds.BlockChain.UnlockTrie(parentRoot)
	if err != nil {
		return nil, err
	}
	stateDiffRlp, err := rlp.EncodeToBytes(stateDiff)
	if err != nil {
		return nil, err
	}
	log.Info("state diff size", "at block height", currentBlock.Number().Uint64(), "rlp byte size", len(stateDiffRlp))
	return sds.newPayload(stateDiffRlp, currentBlock, params)
}

func (sds *Service) newPayload(stateObject []byte, block *types.Block, params Params) (*Payload, error) {
	payload := &Payload{
		StateObjectRlp: stateObject,
	}
	if params.IncludeBlock {
		blockBuff := new(bytes.Buffer)
		if err := block.EncodeRLP(blockBuff); err != nil {
			return nil, err
		}
		payload.BlockRlp = blockBuff.Bytes()
	}
	if params.IncludeTD {
		payload.TotalDifficulty = sds.BlockChain.GetTdByHash(block.Hash())
	}
	if params.IncludeReceipts {
		receiptBuff := new(bytes.Buffer)
		receipts := sds.BlockChain.GetReceiptsByHash(block.Hash())
		if err := rlp.Encode(receiptBuff, receipts); err != nil {
			return nil, err
		}
		payload.ReceiptsRlp = receiptBuff.Bytes()
	}
	return payload, nil
}

// StateTrieAt returns a state trie object payload at the specified blockheight
// This operation cannot be performed back past the point of db pruning; it requires an archival node for historical data
func (sds *Service) StateTrieAt(blockNumber uint64, params Params) (*Payload, error) {
	currentBlock := sds.BlockChain.GetBlockByNumber(blockNumber)
	log.Info("sending state trie", "block height", blockNumber)
	return sds.processStateTrie(currentBlock, params)
}

func (sds *Service) processStateTrie(block *types.Block, params Params) (*Payload, error) {
	stateNodes, err := sds.Builder.BuildStateTrieObject(block)
	if err != nil {
		return nil, err
	}
	stateTrieRlp, err := rlp.EncodeToBytes(stateNodes)
	if err != nil {
		return nil, err
	}
	log.Info("state trie size", "at block height", block.Number().Uint64(), "rlp byte size", len(stateTrieRlp))
	return sds.newPayload(stateTrieRlp, block, params)
}

// Subscribe is used by the API to subscribe to the service loop
func (sds *Service) Subscribe(id rpc.ID, sub chan<- Payload, quitChan chan<- bool, params Params) {
	log.Info("Subscribing to the statediff service")
	if atomic.CompareAndSwapInt32(&sds.subscribers, 0, 1) {
		log.Info("State diffing subscription received; beginning statediff processing")
	}
	// Subscription type is defined as the hash of the rlp-serialized subscription params
	by, err := rlp.EncodeToBytes(params)
	if err != nil {
		log.Error("State diffing params need to be rlp-serializable")
		return
	}
	subscriptionType := crypto.Keccak256Hash(by)
	// Add subscriber
	sds.Lock()
	if sds.Subscriptions[subscriptionType] == nil {
		sds.Subscriptions[subscriptionType] = make(map[rpc.ID]Subscription)
	}
	sds.Subscriptions[subscriptionType][id] = Subscription{
		PayloadChan: sub,
		QuitChan:    quitChan,
	}
	sds.SubscriptionTypes[subscriptionType] = params
	sds.Unlock()
}

// Unsubscribe is used to unsubscribe from the service loop
func (sds *Service) Unsubscribe(id rpc.ID) error {
	log.Info("Unsubscribing from the statediff service", "subscription id", id)
	sds.Lock()
	for ty := range sds.Subscriptions {
		delete(sds.Subscriptions[ty], id)
		if len(sds.Subscriptions[ty]) == 0 {
			// If we removed the last subscription of this type, remove the subscription type outright
			delete(sds.Subscriptions, ty)
			delete(sds.SubscriptionTypes, ty)
		}
	}
	if len(sds.Subscriptions) == 0 {
		if atomic.CompareAndSwapInt32(&sds.subscribers, 1, 0) {
			log.Info("No more subscriptions; halting statediff processing")
		}
	}
	sds.Unlock()
	return nil
}

// Start is used to begin the service
func (sds *Service) Start() error {
	log.Info("Starting statediff service")

	chainEventCh := make(chan core.ChainEvent, chainEventChanSize)
	go sds.Loop(chainEventCh)

	if sds.enableWriteLoop {
		log.Info("Starting statediff DB write loop", "params", writeLoopParams)
		go sds.WriteLoop(make(chan core.ChainEvent, chainEventChanSize))
	}

	return nil
}

// Stop is used to close down the service
func (sds *Service) Stop() error {
	log.Info("Stopping statediff service")
	close(sds.QuitChan)
	return nil
}

// close is used to close all listening subscriptions
func (sds *Service) close() {
	sds.Lock()
	for ty, subs := range sds.Subscriptions {
		for id, sub := range subs {
			select {
			case sub.QuitChan <- true:
				log.Info("closing subscription", "id", id)
			default:
				log.Info("unable to close subscription; channel has no receiver", "subscription id", id)
			}
			delete(sds.Subscriptions[ty], id)
		}
		delete(sds.Subscriptions, ty)
		delete(sds.SubscriptionTypes, ty)
	}
	sds.Unlock()
}

// closeType is used to close all subscriptions of given type
// closeType needs to be called with subscription access locked
func (sds *Service) closeType(subType common.Hash) {
	subs := sds.Subscriptions[subType]
	for id, sub := range subs {
		sendNonBlockingQuit(id, sub)
	}
	delete(sds.Subscriptions, subType)
	delete(sds.SubscriptionTypes, subType)
}

func sendNonBlockingQuit(id rpc.ID, sub Subscription) {
	select {
	case sub.QuitChan <- true:
		log.Info("closing subscription", "id", id)
	default:
		log.Info("unable to close subscription; channel has no receiver", "subscription id", id)
	}
}

// StreamCodeAndCodeHash subscription method for extracting all the codehash=>code mappings that exist in the trie at the provided height
func (sds *Service) StreamCodeAndCodeHash(blockNumber uint64, outChan chan<- CodeAndCodeHash, quitChan chan<- bool) {
	current := sds.BlockChain.GetBlockByNumber(blockNumber)
	log.Info("sending code and codehash", "block height", blockNumber)
	currentTrie, err := sds.BlockChain.StateCache().OpenTrie(current.Root())
	if err != nil {
		log.Error("error creating trie for block", "number", current.Number(), "err", err)
		close(quitChan)
		return
	}
	it := currentTrie.NodeIterator([]byte{})
	leafIt := trie.NewIterator(it)
	go func() {
		defer close(quitChan)
		for leafIt.Next() {
			select {
			case <-sds.QuitChan:
				return
			default:
			}
			account := new(state.Account)
			if err := rlp.DecodeBytes(leafIt.Value, account); err != nil {
				log.Error("error decoding state account", "err", err)
				return
			}
			codeHash := common.BytesToHash(account.CodeHash)
			code, err := sds.BlockChain.StateCache().ContractCode(common.Hash{}, codeHash)
			if err != nil {
				log.Error("error collecting contract code", "err", err)
				return
			}
			outChan <- CodeAndCodeHash{
				Hash: codeHash,
				Code: code,
			}
		}
	}()
}

// WriteStateDiffAt writes a state diff at the specific blockheight directly to the database
// This operation cannot be performed back past the point of db pruning; it requires an archival node
// for historical data
func (sds *Service) WriteStateDiffAt(blockNumber uint64, params Params) error {
	currentBlock := sds.BlockChain.GetBlockByNumber(blockNumber)
	parentRoot := common.Hash{}
	if blockNumber != 0 {
		parentBlock := sds.BlockChain.GetBlockByHash(currentBlock.ParentHash())
		parentRoot = parentBlock.Root()
	}
	return sds.writeStateDiff(currentBlock, parentRoot, params)
}

// Writes a state diff from the current block, parent state root, and provided params
func (sds *Service) writeStateDiff(block *types.Block, parentRoot common.Hash, params Params) error {
	log.Info("writing state diff", "block height", block.Number().Uint64())
	var totalDifficulty *big.Int
	var receipts types.Receipts
	if params.IncludeTD {
		totalDifficulty = sds.BlockChain.GetTdByHash(block.Hash())
	}
	if params.IncludeReceipts {
		receipts = sds.BlockChain.GetReceiptsByHash(block.Hash())
	}
	tx, err := sds.indexer.PushBlock(block, receipts, totalDifficulty)
	if err != nil {
		return err
	}
	// defer handling of commit/rollback for any return case
	defer tx.Close()
	output := func(node StateNode) error {
		return sds.indexer.PushStateNode(tx, node)
	}
	codeOutput := func(c CodeAndCodeHash) error {
		return sds.indexer.PushCodeAndCodeHash(tx, c)
	}
	err = sds.Builder.WriteStateDiffObject(StateRoots{
		NewStateRoot: block.Root(),
		OldStateRoot: parentRoot,
	}, params, output, codeOutput)

	// allow dereferencing of parent, keep current locked as it should be the next parent
	sds.BlockChain.UnlockTrie(parentRoot)
	if err != nil {
		return err
	}
	return nil
}
