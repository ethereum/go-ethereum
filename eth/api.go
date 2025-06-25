// Copyright 2015 The go-ethereum Authors
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

package eth

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/stateless"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/internal/ethapi"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rlp"
	"github.com/scroll-tech/go-ethereum/rollup/ccc"
	"github.com/scroll-tech/go-ethereum/rollup/rcfg"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/scroll-tech/go-ethereum/trie"
)

// PublicEthereumAPI provides an API to access Ethereum full node-related
// information.
type PublicEthereumAPI struct {
	e *Ethereum
}

// NewPublicEthereumAPI creates a new Ethereum protocol API for full nodes.
func NewPublicEthereumAPI(e *Ethereum) *PublicEthereumAPI {
	return &PublicEthereumAPI{e}
}

// Etherbase is the address that mining rewards will be send to
func (api *PublicEthereumAPI) Etherbase() (common.Address, error) {
	return api.e.Etherbase()
}

// Coinbase is the address that mining rewards will be send to (alias for Etherbase)
func (api *PublicEthereumAPI) Coinbase() (common.Address, error) {
	return api.Etherbase()
}

// Hashrate returns the POW hashrate
func (api *PublicEthereumAPI) Hashrate() hexutil.Uint64 {
	return hexutil.Uint64(api.e.Miner().Hashrate())
}

// PublicMinerAPI provides an API to control the miner.
// It offers only methods that operate on data that pose no security risk when it is publicly accessible.
type PublicMinerAPI struct {
	e *Ethereum
}

// NewPublicMinerAPI create a new PublicMinerAPI instance.
func NewPublicMinerAPI(e *Ethereum) *PublicMinerAPI {
	return &PublicMinerAPI{e}
}

// Mining returns an indication if this node is currently mining.
func (api *PublicMinerAPI) Mining() bool {
	return api.e.IsMining()
}

// PrivateMinerAPI provides private RPC methods to control the miner.
// These methods can be abused by external users and must be considered insecure for use by untrusted users.
type PrivateMinerAPI struct {
	e *Ethereum
}

// NewPrivateMinerAPI create a new RPC service which controls the miner of this node.
func NewPrivateMinerAPI(e *Ethereum) *PrivateMinerAPI {
	return &PrivateMinerAPI{e: e}
}

// Start starts the miner with the given number of threads. If threads is nil,
// the number of workers started is equal to the number of logical CPUs that are
// usable by this process. If mining is already running, this method adjust the
// number of threads allowed to use and updates the minimum price required by the
// transaction pool.
func (api *PrivateMinerAPI) Start(threads *int) error {
	if threads == nil {
		return api.e.StartMining(runtime.NumCPU())
	}
	return api.e.StartMining(*threads)
}

// Stop terminates the miner, both at the consensus engine level as well as at
// the block creation level.
func (api *PrivateMinerAPI) Stop() {
	api.e.StopMining()
}

// SetExtra sets the extra data string that is included when this miner mines a block.
func (api *PrivateMinerAPI) SetExtra(extra string) (bool, error) {
	if err := api.e.Miner().SetExtra([]byte(extra)); err != nil {
		return false, err
	}
	return true, nil
}

// SetGasPrice sets the minimum accepted gas price for the miner.
func (api *PrivateMinerAPI) SetGasPrice(gasPrice hexutil.Big) bool {
	api.e.lock.Lock()
	api.e.gasPrice = (*big.Int)(&gasPrice)
	api.e.lock.Unlock()

	// This will override the min base fee configuration.
	// That is fine, it only happens if we explicitly set gas price via the console.
	api.e.txPool.SetGasPrice((*big.Int)(&gasPrice))
	return true
}

// SetGasLimit sets the gaslimit to target towards during mining.
func (api *PrivateMinerAPI) SetGasLimit(gasLimit hexutil.Uint64) bool {
	api.e.Miner().SetGasCeil(uint64(gasLimit))
	return true
}

// SetEtherbase sets the etherbase of the miner
func (api *PrivateMinerAPI) SetEtherbase(etherbase common.Address) bool {
	api.e.SetEtherbase(etherbase)
	return true
}

// PrivateAdminAPI is the collection of Ethereum full node-related APIs
// exposed over the private admin endpoint.
type PrivateAdminAPI struct {
	eth *Ethereum
}

// NewPrivateAdminAPI creates a new API definition for the full node private
// admin methods of the Ethereum service.
func NewPrivateAdminAPI(eth *Ethereum) *PrivateAdminAPI {
	return &PrivateAdminAPI{eth: eth}
}

// ExportChain exports the current blockchain into a local file,
// or a range of blocks if first and last are non-nil
func (api *PrivateAdminAPI) ExportChain(file string, first *uint64, last *uint64) (bool, error) {
	if first == nil && last != nil {
		return false, errors.New("last cannot be specified without first")
	}
	if first != nil && last == nil {
		head := api.eth.BlockChain().CurrentHeader().Number.Uint64()
		last = &head
	}
	if _, err := os.Stat(file); err == nil {
		// File already exists. Allowing overwrite could be a DoS vector,
		// since the 'file' may point to arbitrary paths on the drive
		return false, errors.New("location would overwrite an existing file")
	}
	// Make sure we can create the file to export into
	out, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return false, err
	}
	defer out.Close()

	var writer io.Writer = out
	if strings.HasSuffix(file, ".gz") {
		writer = gzip.NewWriter(writer)
		defer writer.(*gzip.Writer).Close()
	}

	// Export the blockchain
	if first != nil {
		if err := api.eth.BlockChain().ExportN(writer, *first, *last); err != nil {
			return false, err
		}
	} else if err := api.eth.BlockChain().Export(writer); err != nil {
		return false, err
	}
	return true, nil
}

func hasAllBlocks(chain *core.BlockChain, bs []*types.Block) bool {
	for _, b := range bs {
		if !chain.HasBlock(b.Hash(), b.NumberU64()) {
			return false
		}
	}

	return true
}

// ImportChain imports a blockchain from a local file.
func (api *PrivateAdminAPI) ImportChain(file string) (bool, error) {
	// Make sure the can access the file to import
	in, err := os.Open(file)
	if err != nil {
		return false, err
	}
	defer in.Close()

	var reader io.Reader = in
	if strings.HasSuffix(file, ".gz") {
		if reader, err = gzip.NewReader(reader); err != nil {
			return false, err
		}
	}

	// Run actual the import in pre-configured batches
	stream := rlp.NewStream(reader, 0)

	blocks, index := make([]*types.Block, 0, 2500), 0
	for batch := 0; ; batch++ {
		// Load a batch of blocks from the input file
		for len(blocks) < cap(blocks) {
			block := new(types.Block)
			if err := stream.Decode(block); err == io.EOF {
				break
			} else if err != nil {
				return false, fmt.Errorf("block %d: failed to parse: %v", index, err)
			}
			blocks = append(blocks, block)
			index++
		}
		if len(blocks) == 0 {
			break
		}

		if hasAllBlocks(api.eth.BlockChain(), blocks) {
			blocks = blocks[:0]
			continue
		}
		// Import the batch and reset the buffer
		if _, err := api.eth.BlockChain().InsertChain(blocks); err != nil {
			return false, fmt.Errorf("batch %d: failed to insert: %v", batch, err)
		}
		blocks = blocks[:0]
	}
	return true, nil
}

// SetRollupEventSyncedL1Height sets the synced L1 height for rollup event synchronization
func (api *PrivateAdminAPI) SetRollupEventSyncedL1Height(height uint64) error {
	rollupSyncService := api.eth.GetRollupSyncService()
	if rollupSyncService == nil {
		return errors.New("RollupSyncService is not available")
	}

	log.Info("Setting rollup event synced L1 height", "height", height)
	rollupSyncService.ResetStartSyncHeight(height)

	return nil
}

// SetL1MessageSyncedL1Height sets the synced L1 height for L1 message synchronization
func (api *PrivateAdminAPI) SetL1MessageSyncedL1Height(height uint64) error {
	syncService := api.eth.GetSyncService()
	if syncService == nil {
		return errors.New("SyncService is not available")
	}

	log.Info("Setting L1 message synced L1 height", "height", height)
	syncService.ResetStartSyncHeight(height)

	return nil
}

// PublicDebugAPI is the collection of Ethereum full node APIs exposed
// over the public debugging endpoint.
type PublicDebugAPI struct {
	eth *Ethereum
}

// NewPublicDebugAPI creates a new API definition for the full node-
// related public debug methods of the Ethereum service.
func NewPublicDebugAPI(eth *Ethereum) *PublicDebugAPI {
	return &PublicDebugAPI{eth: eth}
}

// DumpBlock retrieves the entire state of the database at a given block.
func (api *PublicDebugAPI) DumpBlock(blockNr rpc.BlockNumber) (state.Dump, error) {
	opts := &state.DumpConfig{
		OnlyWithAddresses: true,
		Max:               AccountRangeMaxResults, // Sanity limit over RPC
	}
	if blockNr == rpc.PendingBlockNumber {
		// If we're dumping the pending state, we need to request
		// both the pending block as well as the pending state from
		// the miner and operate on those
		_, stateDb := api.eth.miner.Pending()
		return stateDb.RawDump(opts), nil
	}
	var block *types.Block
	if blockNr == rpc.LatestBlockNumber {
		block = api.eth.blockchain.CurrentBlock()
	} else {
		block = api.eth.blockchain.GetBlockByNumber(uint64(blockNr))
	}
	if block == nil {
		return state.Dump{}, fmt.Errorf("block #%d not found", blockNr)
	}
	stateDb, err := api.eth.BlockChain().StateAt(block.Root())
	if err != nil {
		return state.Dump{}, err
	}
	return stateDb.RawDump(opts), nil
}

func (api *PublicDebugAPI) ExecutionWitness(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*ExecutionWitness, error) {
	block, err := api.eth.APIBackend.BlockByNumberOrHash(ctx, blockNrOrHash)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve block: %w", err)
	}
	if block == nil {
		return nil, fmt.Errorf("block not found: %s", blockNrOrHash.String())
	}

	witness, err := generateWitness(api.eth.blockchain, block)
	if err != nil {
		return nil, fmt.Errorf("failed to generate witness: %w", err)
	}

	return ToExecutionWitness(witness), nil
}

func generateWitness(blockchain *core.BlockChain, block *types.Block) (*stateless.Witness, error) {
	witness, err := stateless.NewWitness(block.Header(), blockchain)
	if err != nil {
		return nil, fmt.Errorf("failed to create witness: %w", err)
	}

	parentHeader := witness.Headers[0]
	// Avoid using snapshots to properly collect the witness data for all reads
	statedb, err := state.New(parentHeader.Root, blockchain.StateCache(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve parent state: %w", err)
	}
	statedb.WithWitness(witness)

	// Collect storage locations that prover needs but sequencer might not touch necessarily
	statedb.GetState(rcfg.L2MessageQueueAddress, rcfg.WithdrawTrieRootSlot)

	// Note: scroll-revm detects the Feynman transition block using this storage slot,
	// since it does not have access to the parent block timestamp. We need to make
	// sure that this is always present in the execution witness.
	statedb.GetState(rcfg.L1GasPriceOracleAddress, rcfg.IsFeynmanSlot)

	receipts, _, usedGas, err := blockchain.Processor().Process(block, statedb, *blockchain.GetVMConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to process block %d: %w", block.Number(), err)
	}

	if err := blockchain.Validator().ValidateState(block, statedb, receipts, usedGas); err != nil {
		return nil, fmt.Errorf("failed to validate block %d: %w", block.Number(), err)
	}

	if err = testWitness(blockchain, block, witness); err != nil {
		return nil, err
	}
	return witness, nil
}

func testWitness(blockchain *core.BlockChain, block *types.Block, witness *stateless.Witness) error {
	stateRoot := witness.Root()
	diskRoot, err := rawdb.ReadDiskStateRoot(blockchain.Database(), stateRoot)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return fmt.Errorf("failed to read disk state root for stateRoot %s: %w", stateRoot.Hex(), err)
	}
	if diskRoot != (common.Hash{}) {
		log.Debug("Using disk root for state root", "stateRoot", stateRoot.Hex(), "diskRoot", diskRoot.Hex())
		stateRoot = diskRoot
	}

	// Create and populate the state database to serve as the stateless backend
	statedb, err := state.New(stateRoot, state.NewDatabase(witness.MakeHashDB()), nil)
	if err != nil {
		return fmt.Errorf("failed to create state database with stateRoot %s: %w", stateRoot.Hex(), err)
	}

	receipts, _, usedGas, err := blockchain.Processor().Process(block, statedb, *blockchain.GetVMConfig())
	if err != nil {
		return fmt.Errorf("failed to process block %d (hash: %s): %w", block.Number(), block.Hash().Hex(), err)
	}

	if err := blockchain.Validator().ValidateState(block, statedb, receipts, usedGas); err != nil {
		return fmt.Errorf("failed to validate block %d (hash: %s): %w", block.Number(), block.Hash().Hex(), err)
	}

	postStateRoot := block.Root()
	diskRoot, err = rawdb.ReadDiskStateRoot(blockchain.Database(), postStateRoot)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return fmt.Errorf("failed to read disk state root for postStateRoot %s: %w", postStateRoot.Hex(), err)
	}
	if diskRoot != (common.Hash{}) {
		log.Debug("Using disk root for post state root", "postStateRoot", postStateRoot.Hex(), "diskRoot", diskRoot.Hex())
		postStateRoot = diskRoot
	}
	computedRoot := statedb.GetRootHash()
	if computedRoot != postStateRoot {
		log.Debug("State root mismatch", "block", block.Number(), "expected", postStateRoot.Hex(), "got", computedRoot)
		executionWitness := ToExecutionWitness(witness)
		jsonStr, err := json.Marshal(executionWitness)
		if err != nil {
			return fmt.Errorf("state root mismatch after processing block %d (hash: %s): expected %s, got %s, but failed to marshal witness: %w", block.Number(), block.Hash().Hex(), postStateRoot.Hex(), computedRoot, err)
		}
		return fmt.Errorf("state root mismatch after processing block %d (hash: %s): expected %s, got %s, witness: %s", block.Number(), block.Hash().Hex(), postStateRoot.Hex(), computedRoot, string(jsonStr))
	}
	return nil
}

// ExecutionWitness is a witness json encoding for transferring across the network.
// In the future, we'll probably consider using the extWitness format instead for less overhead if performance becomes an issue.
// Currently using this format for ease of reading, parsing and compatibility across clients.
type ExecutionWitness struct {
	Headers []*types.Header   `json:"headers"`
	Codes   map[string]string `json:"codes"`
	State   map[string]string `json:"state"`
}

func transformMap(in map[string]struct{}) map[string]string {
	out := make(map[string]string, len(in))
	for item := range in {
		bytes := []byte(item)
		key := crypto.Keccak256Hash(bytes).Hex()
		out[key] = hexutil.Encode(bytes)
	}
	return out
}

// ToExecutionWitness converts a witness to an execution witness format that is compatible with reth.
// keccak(node) => node
// keccak(bytecodes) => bytecodes
func ToExecutionWitness(w *stateless.Witness) *ExecutionWitness {
	return &ExecutionWitness{
		Headers: w.Headers,
		Codes:   transformMap(w.Codes),
		State:   transformMap(w.State),
	}
}

// PrivateDebugAPI is the collection of Ethereum full node APIs exposed over
// the private debugging endpoint.
type PrivateDebugAPI struct {
	eth *Ethereum
}

// NewPrivateDebugAPI creates a new API definition for the full node-related
// private debug methods of the Ethereum service.
func NewPrivateDebugAPI(eth *Ethereum) *PrivateDebugAPI {
	return &PrivateDebugAPI{eth: eth}
}

// Preimage is a debug API function that returns the preimage for a sha3 hash, if known.
func (api *PrivateDebugAPI) Preimage(ctx context.Context, hash common.Hash) (hexutil.Bytes, error) {
	if preimage := rawdb.ReadPreimage(api.eth.ChainDb(), hash); preimage != nil {
		return preimage, nil
	}
	return nil, errors.New("unknown preimage")
}

// BadBlockArgs represents the entries in the list returned when bad blocks are queried.
type BadBlockArgs struct {
	Hash  common.Hash            `json:"hash"`
	Block map[string]interface{} `json:"block"`
	RLP   string                 `json:"rlp"`
}

// GetBadBlocks returns a list of the last 'bad blocks' that the client has seen on the network
// and returns them as a JSON list of block-hashes
func (api *PrivateDebugAPI) GetBadBlocks(ctx context.Context) ([]*BadBlockArgs, error) {
	var (
		err     error
		blocks  = rawdb.ReadAllBadBlocks(api.eth.chainDb)
		results = make([]*BadBlockArgs, 0, len(blocks))
	)
	for _, block := range blocks {
		var (
			blockRlp  string
			blockJSON map[string]interface{}
		)
		if rlpBytes, err := rlp.EncodeToBytes(block); err != nil {
			blockRlp = err.Error() // Hacky, but hey, it works
		} else {
			blockRlp = fmt.Sprintf("0x%x", rlpBytes)
		}
		if blockJSON, err = ethapi.RPCMarshalBlock(block, true, true, api.eth.APIBackend.ChainConfig()); err != nil {
			blockJSON = map[string]interface{}{"error": err.Error()}
		}
		results = append(results, &BadBlockArgs{
			Hash:  block.Hash(),
			RLP:   blockRlp,
			Block: blockJSON,
		})
	}
	return results, nil
}

// AccountRangeMaxResults is the maximum number of results to be returned per call
const AccountRangeMaxResults = 256

// AccountRange enumerates all accounts in the given block and start point in paging request
func (api *PublicDebugAPI) AccountRange(blockNrOrHash rpc.BlockNumberOrHash, start []byte, maxResults int, nocode, nostorage, incompletes bool) (state.IteratorDump, error) {
	var stateDb *state.StateDB
	var err error

	if number, ok := blockNrOrHash.Number(); ok {
		if number == rpc.PendingBlockNumber {
			// If we're dumping the pending state, we need to request
			// both the pending block as well as the pending state from
			// the miner and operate on those
			_, stateDb = api.eth.miner.Pending()
		} else {
			var block *types.Block
			if number == rpc.LatestBlockNumber {
				block = api.eth.blockchain.CurrentBlock()
			} else {
				block = api.eth.blockchain.GetBlockByNumber(uint64(number))
			}
			if block == nil {
				return state.IteratorDump{}, fmt.Errorf("block #%d not found", number)
			}
			stateDb, err = api.eth.BlockChain().StateAt(block.Root())
			if err != nil {
				return state.IteratorDump{}, err
			}
		}
	} else if hash, ok := blockNrOrHash.Hash(); ok {
		block := api.eth.blockchain.GetBlockByHash(hash)
		if block == nil {
			return state.IteratorDump{}, fmt.Errorf("block %s not found", hash.Hex())
		}
		stateDb, err = api.eth.BlockChain().StateAt(block.Root())
		if err != nil {
			return state.IteratorDump{}, err
		}
	} else {
		return state.IteratorDump{}, errors.New("either block number or block hash must be specified")
	}

	opts := &state.DumpConfig{
		SkipCode:          nocode,
		SkipStorage:       nostorage,
		OnlyWithAddresses: !incompletes,
		Start:             start,
		Max:               uint64(maxResults),
	}
	if maxResults > AccountRangeMaxResults || maxResults <= 0 {
		opts.Max = AccountRangeMaxResults
	}
	return stateDb.IteratorDump(opts), nil
}

// StorageRangeResult is the result of a debug_storageRangeAt API call.
type StorageRangeResult struct {
	Storage storageMap   `json:"storage"`
	NextKey *common.Hash `json:"nextKey"` // nil if Storage includes the last key in the trie.
}

type storageMap map[common.Hash]storageEntry

type storageEntry struct {
	Key   *common.Hash `json:"key"`
	Value common.Hash  `json:"value"`
}

// StorageRangeAt returns the storage at the given block height and transaction index.
func (api *PrivateDebugAPI) StorageRangeAt(blockHash common.Hash, txIndex int, contractAddress common.Address, keyStart hexutil.Bytes, maxResult int) (StorageRangeResult, error) {
	// Retrieve the block
	block := api.eth.blockchain.GetBlockByHash(blockHash)
	if block == nil {
		return StorageRangeResult{}, fmt.Errorf("block %#x not found", blockHash)
	}
	_, _, statedb, err := api.eth.stateAtTransaction(block, txIndex, 0)
	if err != nil {
		return StorageRangeResult{}, err
	}
	st := statedb.StorageTrie(contractAddress)
	if st == nil {
		return StorageRangeResult{}, fmt.Errorf("account %x doesn't exist", contractAddress)
	}
	return storageRangeAt(st, keyStart, maxResult)
}

func storageRangeAt(st state.Trie, start []byte, maxResult int) (StorageRangeResult, error) {
	it := trie.NewIterator(st.NodeIterator(start))
	result := StorageRangeResult{Storage: storageMap{}}
	for i := 0; i < maxResult && it.Next(); i++ {
		_, content, _, err := rlp.Split(it.Value)
		if err != nil {
			return StorageRangeResult{}, err
		}
		e := storageEntry{Value: common.BytesToHash(content)}
		if preimage := st.GetKey(it.Key); preimage != nil {
			preimage := common.BytesToHash(preimage)
			e.Key = &preimage
		}
		result.Storage[common.BytesToHash(it.Key)] = e
	}
	// Add the 'next key' so clients can continue downloading.
	if it.Next() {
		next := common.BytesToHash(it.Key)
		result.NextKey = &next
	}
	return result, nil
}

// GetModifiedAccountsByNumber returns all accounts that have changed between the
// two blocks specified. A change is defined as a difference in nonce, balance,
// code hash, or storage hash.
//
// With one parameter, returns the list of accounts modified in the specified block.
func (api *PrivateDebugAPI) GetModifiedAccountsByNumber(startNum uint64, endNum *uint64) ([]common.Address, error) {
	var startBlock, endBlock *types.Block

	startBlock = api.eth.blockchain.GetBlockByNumber(startNum)
	if startBlock == nil {
		return nil, fmt.Errorf("start block %x not found", startNum)
	}

	if endNum == nil {
		endBlock = startBlock
		startBlock = api.eth.blockchain.GetBlockByHash(startBlock.ParentHash())
		if startBlock == nil {
			return nil, fmt.Errorf("block %x has no parent", endBlock.Number())
		}
	} else {
		endBlock = api.eth.blockchain.GetBlockByNumber(*endNum)
		if endBlock == nil {
			return nil, fmt.Errorf("end block %d not found", *endNum)
		}
	}
	return api.getModifiedAccounts(startBlock, endBlock)
}

// GetModifiedAccountsByHash returns all accounts that have changed between the
// two blocks specified. A change is defined as a difference in nonce, balance,
// code hash, or storage hash.
//
// With one parameter, returns the list of accounts modified in the specified block.
func (api *PrivateDebugAPI) GetModifiedAccountsByHash(startHash common.Hash, endHash *common.Hash) ([]common.Address, error) {
	var startBlock, endBlock *types.Block
	startBlock = api.eth.blockchain.GetBlockByHash(startHash)
	if startBlock == nil {
		return nil, fmt.Errorf("start block %x not found", startHash)
	}

	if endHash == nil {
		endBlock = startBlock
		startBlock = api.eth.blockchain.GetBlockByHash(startBlock.ParentHash())
		if startBlock == nil {
			return nil, fmt.Errorf("block %x has no parent", endBlock.Number())
		}
	} else {
		endBlock = api.eth.blockchain.GetBlockByHash(*endHash)
		if endBlock == nil {
			return nil, fmt.Errorf("end block %x not found", *endHash)
		}
	}
	return api.getModifiedAccounts(startBlock, endBlock)
}

func (api *PrivateDebugAPI) getModifiedAccounts(startBlock, endBlock *types.Block) ([]common.Address, error) {
	if startBlock.Number().Uint64() >= endBlock.Number().Uint64() {
		return nil, fmt.Errorf("start block height (%d) must be less than end block height (%d)", startBlock.Number().Uint64(), endBlock.Number().Uint64())
	}
	triedb := api.eth.BlockChain().StateCache().TrieDB()

	oldTrie, err := trie.NewSecure(startBlock.Root(), triedb)
	if err != nil {
		return nil, err
	}
	newTrie, err := trie.NewSecure(endBlock.Root(), triedb)
	if err != nil {
		return nil, err
	}
	diff, _ := trie.NewDifferenceIterator(oldTrie.NodeIterator([]byte{}), newTrie.NodeIterator([]byte{}))
	iter := trie.NewIterator(diff)

	var dirty []common.Address
	for iter.Next() {
		key := newTrie.GetKey(iter.Key)
		if key == nil {
			return nil, fmt.Errorf("no preimage found for hash %x", iter.Key)
		}
		dirty = append(dirty, common.BytesToAddress(key))
	}
	return dirty, nil
}

// GetAccessibleState returns the first number where the node has accessible
// state on disk. Note this being the post-state of that block and the pre-state
// of the next block.
// The (from, to) parameters are the sequence of blocks to search, which can go
// either forwards or backwards
func (api *PrivateDebugAPI) GetAccessibleState(from, to rpc.BlockNumber) (uint64, error) {
	db := api.eth.ChainDb()
	var pivot uint64
	if p := rawdb.ReadLastPivotNumber(db); p != nil {
		pivot = *p
		log.Info("Found fast-sync pivot marker", "number", pivot)
	}
	var resolveNum = func(num rpc.BlockNumber) (uint64, error) {
		// We don't have state for pending (-2), so treat it as latest
		if num.Int64() < 0 {
			block := api.eth.blockchain.CurrentBlock()
			if block == nil {
				return 0, fmt.Errorf("current block missing")
			}
			return block.NumberU64(), nil
		}
		return uint64(num.Int64()), nil
	}
	var (
		start   uint64
		end     uint64
		delta   = int64(1)
		lastLog time.Time
		err     error
	)
	if start, err = resolveNum(from); err != nil {
		return 0, err
	}
	if end, err = resolveNum(to); err != nil {
		return 0, err
	}
	if start == end {
		return 0, fmt.Errorf("from and to needs to be different")
	}
	if start > end {
		delta = -1
	}
	for i := int64(start); i != int64(end); i += delta {
		if time.Since(lastLog) > 8*time.Second {
			log.Info("Finding roots", "from", start, "to", end, "at", i)
			lastLog = time.Now()
		}
		if i < int64(pivot) {
			continue
		}
		h := api.eth.BlockChain().GetHeaderByNumber(uint64(i))
		if h == nil {
			return 0, fmt.Errorf("missing header %d", i)
		}
		if ok, _ := api.eth.ChainDb().Has(h.Root[:]); ok {
			return uint64(i), nil
		}
	}
	return 0, fmt.Errorf("No state found")
}

// ScrollAPI provides private RPC methods to query the L1 message database.
type ScrollAPI struct {
	eth *Ethereum
}

// l1MessageTxRPC is the RPC-layer representation of an L1 message.
type l1MessageTxRPC struct {
	QueueIndex uint64          `json:"queueIndex"`
	Gas        uint64          `json:"gas"`
	To         *common.Address `json:"to"`
	Value      *hexutil.Big    `json:"value"`
	Data       hexutil.Bytes   `json:"data"`
	Sender     common.Address  `json:"sender"`
	Hash       common.Hash     `json:"hash"`
}

// NewScrollAPI creates a new RPC service to query the L1 message database.
func NewScrollAPI(eth *Ethereum) *ScrollAPI {
	return &ScrollAPI{eth: eth}
}

// GetL1SyncHeight returns the latest synced L1 block height from the local database.
func (api *ScrollAPI) GetL1SyncHeight(ctx context.Context) (height *uint64, err error) {
	return rawdb.ReadSyncedL1BlockNumber(api.eth.ChainDb()), nil
}

// GetL1MessageByIndex queries an L1 message by its index in the local database.
func (api *ScrollAPI) GetL1MessageByIndex(ctx context.Context, queueIndex uint64) (height *l1MessageTxRPC, err error) {
	msg := rawdb.ReadL1Message(api.eth.ChainDb(), queueIndex)
	if msg == nil {
		return nil, nil
	}
	rpcMsg := l1MessageTxRPC{
		QueueIndex: msg.QueueIndex,
		Gas:        msg.Gas,
		To:         msg.To,
		Value:      (*hexutil.Big)(msg.Value),
		Data:       msg.Data,
		Sender:     msg.Sender,
		Hash:       types.NewTx(msg).Hash(),
	}
	return &rpcMsg, nil
}

// GetFirstQueueIndexNotInL2Block returns the first L1 message queue index that is
// not included in the chain up to and including the provided block.
func (api *ScrollAPI) GetFirstQueueIndexNotInL2Block(ctx context.Context, hash common.Hash) (queueIndex *uint64, err error) {
	return rawdb.ReadFirstQueueIndexNotInL2Block(api.eth.ChainDb(), hash), nil
}

// GetLatestRelayedQueueIndex returns the highest L1 message queue index included in the canonical chain.
func (api *ScrollAPI) GetLatestRelayedQueueIndex(ctx context.Context) (queueIndex *uint64, err error) {
	block := api.eth.blockchain.CurrentBlock()
	queueIndex, err = api.GetFirstQueueIndexNotInL2Block(ctx, block.Hash())
	if queueIndex == nil || err != nil {
		return queueIndex, err
	}
	if *queueIndex == 0 {
		return nil, nil
	}
	lastIncluded := *queueIndex - 1
	return &lastIncluded, nil
}

// rpcMarshalBlock uses the generalized output filler, then adds the total difficulty field, which requires
// a `ScrollAPI`.
func (api *ScrollAPI) rpcMarshalBlock(ctx context.Context, b *types.Block, fullTx bool) (map[string]interface{}, error) {
	fields, err := ethapi.RPCMarshalBlock(b, true, fullTx, api.eth.APIBackend.ChainConfig())
	if err != nil {
		return nil, err
	}
	fields["totalDifficulty"] = (*hexutil.Big)(api.eth.APIBackend.GetTd(ctx, b.Hash()))
	rc := rawdb.ReadBlockRowConsumption(api.eth.ChainDb(), b.Hash())
	if rc != nil {
		fields["rowConsumption"] = rc
	} else {
		fields["rowConsumption"] = nil
	}
	return fields, err
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned.
func (api *ScrollAPI) GetBlockByHash(ctx context.Context, hash common.Hash, fullTx bool) (map[string]interface{}, error) {
	block, err := api.eth.APIBackend.BlockByHash(ctx, hash)
	if block != nil {
		return api.rpcMarshalBlock(ctx, block, fullTx)
	}
	return nil, err
}

// GetBlockByNumber returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned.
func (api *ScrollAPI) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	block, err := api.eth.APIBackend.BlockByNumber(ctx, number)
	if block != nil {
		return api.rpcMarshalBlock(ctx, block, fullTx)
	}
	return nil, err
}

// GetNumSkippedTransactions returns the number of skipped transactions.
func (api *ScrollAPI) GetNumSkippedTransactions(ctx context.Context) (uint64, error) {
	return rawdb.ReadNumSkippedTransactions(api.eth.ChainDb()), nil
}

// SyncStatus includes L2 block sync height, L1 rollup sync height,
// L1 message sync height, and L2 finalized block height.
type SyncStatus struct {
	L2BlockSyncHeight      uint64 `json:"l2BlockSyncHeight,omitempty"`
	L1RollupSyncHeight     uint64 `json:"l1RollupSyncHeight,omitempty"`
	L1MessageSyncHeight    uint64 `json:"l1MessageSyncHeight,omitempty"`
	L2FinalizedBlockHeight uint64 `json:"l2FinalizedBlockHeight,omitempty"`
}

// SyncStatus returns the overall rollup status including L2 block sync height, L1 rollup sync height,
// L1 message sync height, and L2 finalized block height.
func (api *ScrollAPI) SyncStatus(_ context.Context) *SyncStatus {
	status := &SyncStatus{}

	l2BlockHeader := api.eth.blockchain.CurrentHeader()
	if l2BlockHeader != nil {
		status.L2BlockSyncHeight = l2BlockHeader.Number.Uint64()
	}

	l1RollupSyncHeightPtr := rawdb.ReadRollupEventSyncedL1BlockNumber(api.eth.ChainDb())
	if l1RollupSyncHeightPtr != nil {
		status.L1RollupSyncHeight = *l1RollupSyncHeightPtr
	}

	l1MessageSyncHeightPtr := rawdb.ReadSyncedL1BlockNumber(api.eth.ChainDb())
	if l1MessageSyncHeightPtr != nil {
		status.L1MessageSyncHeight = *l1MessageSyncHeightPtr
	}

	l2FinalizedBlockHeightPtr := rawdb.ReadFinalizedL2BlockNumber(api.eth.ChainDb())
	if l2FinalizedBlockHeightPtr != nil {
		status.L2FinalizedBlockHeight = *l2FinalizedBlockHeightPtr
	}

	return status
}

// EstimateL1DataFee returns an estimate of the L1 data fee required to
// process the given transaction against the current pending block.
func (api *ScrollAPI) EstimateL1DataFee(ctx context.Context, args ethapi.TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash) (*hexutil.Uint64, error) {
	bNrOrHash := rpc.BlockNumberOrHashWithNumber(rpc.PendingBlockNumber)
	if blockNrOrHash != nil {
		bNrOrHash = *blockNrOrHash
	}

	l1DataFee, err := ethapi.EstimateL1MsgFee(ctx, api.eth.APIBackend, args, bNrOrHash, nil, 0, api.eth.APIBackend.RPCGasCap(), api.eth.APIBackend.ChainConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to estimate L1 data fee: %w", err)
	}

	result := hexutil.Uint64(l1DataFee.Uint64())
	return &result, nil
}

// RPCTransaction is the standard RPC transaction return type with some additional skip-related fields.
type RPCTransaction struct {
	ethapi.RPCTransaction
	SkipReason      string       `json:"skipReason"`
	SkipBlockNumber *hexutil.Big `json:"skipBlockNumber"`
	SkipBlockHash   *common.Hash `json:"skipBlockHash,omitempty"`

	// wrapped traces, currently only available for `scroll_getSkippedTransaction` API, when `MinerStoreSkippedTxTracesFlag` is set
	Traces *types.BlockTrace `json:"traces,omitempty"`
}

// GetSkippedTransaction returns a skipped transaction by its hash.
func (api *ScrollAPI) GetSkippedTransaction(ctx context.Context, hash common.Hash) (*RPCTransaction, error) {
	stx := rawdb.ReadSkippedTransaction(api.eth.ChainDb(), hash)
	if stx == nil {
		return nil, nil
	}
	var rpcTx RPCTransaction
	rpcTx.RPCTransaction = *ethapi.NewRPCTransaction(stx.Tx, common.Hash{}, 0, 0, 0, nil, api.eth.blockchain.Config())
	rpcTx.SkipReason = stx.Reason
	rpcTx.SkipBlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(stx.BlockNumber))
	rpcTx.SkipBlockHash = stx.BlockHash
	if len(stx.TracesBytes) != 0 {
		traces := &types.BlockTrace{}
		if err := json.Unmarshal(stx.TracesBytes, traces); err != nil {
			return nil, fmt.Errorf("fail to Unmarshal traces for skipped tx, hash: %s, err: %w", hash.String(), err)
		}
		rpcTx.Traces = traces
	}
	return &rpcTx, nil
}

// GetSkippedTransactionHashes returns a list of skipped transaction hashes between the two indices provided (inclusive).
func (api *ScrollAPI) GetSkippedTransactionHashes(ctx context.Context, from uint64, to uint64) ([]common.Hash, error) {
	it := rawdb.IterateSkippedTransactionsFrom(api.eth.ChainDb(), from)
	defer it.Release()

	var hashes []common.Hash

	for it.Next() {
		if it.Index() > to {
			break
		}
		hashes = append(hashes, it.TransactionHash())
	}

	return hashes, nil
}

// CalculateRowConsumptionByBlockNumber
func (api *ScrollAPI) CalculateRowConsumptionByBlockNumber(ctx context.Context, number rpc.BlockNumber) (*types.RowConsumption, error) {
	block := api.eth.blockchain.GetBlockByNumber(uint64(number.Int64()))
	if block == nil {
		return nil, errors.New("block not found")
	}

	// todo: fix temp AsyncChecker leaking the internal Checker instances
	var checkErr error
	asyncChecker := ccc.NewAsyncChecker(api.eth.blockchain, 1, false).WithOnFailingBlock(func(b *types.Block, err error) {
		log.Error("failed to calculate row consumption on demand", "number", number, "hash", b.Hash().Hex(), "err", err)
		checkErr = err
	})
	if err := asyncChecker.Check(block); err != nil {
		return nil, err
	}
	asyncChecker.Wait()
	return rawdb.ReadBlockRowConsumption(api.eth.ChainDb(), block.Hash()), checkErr
}

type DiskAndHeaderRoot struct {
	DiskRoot   common.Hash `json:"diskRoot"`
	HeaderRoot common.Hash `json:"headerRoot"`
}

// DiskRoot
func (api *ScrollAPI) DiskRoot(ctx context.Context, blockNrOrHash *rpc.BlockNumberOrHash) (DiskAndHeaderRoot, error) {
	block, err := api.eth.APIBackend.BlockByNumberOrHash(ctx, *blockNrOrHash)
	if err != nil {
		return DiskAndHeaderRoot{}, fmt.Errorf("failed to retrieve block: %w", err)
	}
	if block == nil {
		return DiskAndHeaderRoot{}, fmt.Errorf("block not found: %s", blockNrOrHash.String())
	}

	if diskRoot, _ := rawdb.ReadDiskStateRoot(api.eth.ChainDb(), block.Root()); diskRoot != (common.Hash{}) {
		return DiskAndHeaderRoot{
			DiskRoot:   diskRoot,
			HeaderRoot: block.Root(),
		}, nil
	}
	return DiskAndHeaderRoot{
		DiskRoot:   block.Root(),
		HeaderRoot: block.Root(),
	}, nil
}
