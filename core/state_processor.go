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

package core

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	tutils "github.com/ethereum/go-ethereum/trie/utils"
	"github.com/ethereum/go-verkle"
	"github.com/holiman/uint256"
)

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain         // Canonical block chain
	engine consensus.Engine    // Consensus engine used for block rewards
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *params.ChainConfig, bc *BlockChain, engine consensus.Engine) *StateProcessor {
	return &StateProcessor{
		config: config,
		bc:     bc,
		engine: engine,
	}
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the statedb and applying any rewards to both
// the processor (coinbase) and any included uncles.
//
// Process returns the receipts and logs accumulated during the process and
// returns the amount of gas that was used in the process. If any of the
// transactions failed to execute due to insufficient gas it will return an error.
func (p *StateProcessor) Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (types.Receipts, []*types.Log, uint64, error) {
	var (
		receipts    types.Receipts
		usedGas     = new(uint64)
		header      = block.Header()
		blockHash   = block.Hash()
		blockNumber = block.Number()
		allLogs     []*types.Log
		gp          = new(GasPool).AddGas(block.GasLimit())
	)
	// Mutate the block and state according to any hard-fork specs
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	var (
		context = NewEVMBlockContext(header, p.bc, nil)
		vmenv   = vm.NewEVM(context, vm.TxContext{}, statedb, p.config, cfg)
		signer  = types.MakeSigner(p.config, header.Number, header.Time)
	)
	// Iterate over and process the individual transactions
	for i, tx := range block.Transactions() {
		msg, err := TransactionToMessage(tx, signer, header.BaseFee)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}
		statedb.SetTxContext(tx.Hash(), i)
		receipt, err := applyTransaction(msg, p.config, gp, statedb, blockNumber, blockHash, tx, usedGas, vmenv)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}
	// Fail if Shanghai not enabled and len(withdrawals) is non-zero.
	withdrawals := block.Withdrawals()
	if len(withdrawals) > 0 && !p.config.IsShanghai(block.Number(), block.Time()) {
		return nil, nil, 0, errors.New("withdrawals before shanghai")
	}

	// Perform the overlay transition, if relevant
	if err := OverlayVerkleTransition(statedb); err != nil {
		return nil, nil, 0, fmt.Errorf("error performing verkle overlay transition: %w", err)
	}

	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	p.engine.Finalize(p.bc, header, statedb, block.Transactions(), block.Uncles(), withdrawals)

	if block.NumberU64()%100 == 0 {
		stateRoot := statedb.GetTrie().Hash()
		log.Info("State root", "number", block.NumberU64(), "hash", stateRoot)
	}

	return receipts, allLogs, *usedGas, nil
}

func applyTransaction(msg *Message, config *params.ChainConfig, gp *GasPool, statedb *state.StateDB, blockNumber *big.Int, blockHash common.Hash, tx *types.Transaction, usedGas *uint64, evm *vm.EVM) (*types.Receipt, error) {
	// Create a new context to be used in the EVM environment.
	txContext := NewEVMTxContext(msg)
	txContext.Accesses = statedb.NewAccessWitness()
	evm.Reset(txContext, statedb)

	// Apply the transaction to the current state (included in the env).
	result, err := ApplyMessage(evm, msg, gp)
	if err != nil {
		return nil, err
	}

	// Update the state with pending changes.
	var root []byte
	if config.IsByzantium(blockNumber) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(blockNumber)).Bytes()
	}
	*usedGas += result.UsedGas

	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt := &types.Receipt{Type: tx.Type(), PostState: root, CumulativeGasUsed: *usedGas}
	if result.Failed() {
		receipt.Status = types.ReceiptStatusFailed
	} else {
		receipt.Status = types.ReceiptStatusSuccessful
	}
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = result.UsedGas

	// If the transaction created a contract, store the creation address in the receipt.
	if msg.To == nil {
		receipt.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, tx.Nonce())
	}

	statedb.Witness().Merge(txContext.Accesses)

	// Set the receipt logs and create the bloom filter.
	receipt.Logs = statedb.GetLogs(tx.Hash(), blockNumber.Uint64(), blockHash)
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	receipt.BlockHash = blockHash
	receipt.BlockNumber = blockNumber
	receipt.TransactionIndex = uint(statedb.TxIndex())
	return receipt, err
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
func ApplyTransaction(config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, error) {
	msg, err := TransactionToMessage(tx, types.MakeSigner(config, header.Number, header.Time), header.BaseFee)
	if err != nil {
		return nil, err
	}
	// Create a new context to be used in the EVM environment
	blockContext := NewEVMBlockContext(header, bc, author)
	vmenv := vm.NewEVM(blockContext, vm.TxContext{BlobHashes: tx.BlobHashes()}, statedb, config, cfg)
	return applyTransaction(msg, config, gp, statedb, header.Number, header.Hash(), tx, usedGas, vmenv)
}

var zeroTreeIndex uint256.Int

// keyValueMigrator is a helper module that collects key-values from the overlay-tree migration for Verkle Trees.
// It assumes that the walk of the base tree is done in address-order, so it exploit that fact to
// collect the key-values in a way that is efficient.
type keyValueMigrator struct {
	// leafData contains the values for the future leaf for a particular VKT branch.
	leafData []migratedKeyValue

	// When prepare() is called, it will start a background routine that will process the leafData
	// saving the result in newLeaves to be used by migrateCollectedKeyValues(). The background
	// routine signals that it is done by closing processingReady.
	processingReady chan struct{}
	newLeaves       []verkle.LeafNode
	prepareErr      error
}

func newKeyValueMigrator() *keyValueMigrator {
	// We do initialize the VKT config since prepare() might indirectly make multiple GetConfig() calls
	// in different goroutines when we never called GetConfig() before, causing a race considering the way
	// that `config` is designed in go-verkle.
	// TODO: jsign as a fix for this in the PR where we move to a file-less precomp, since it allows safe
	//       concurrent calls to GetConfig(). When that gets merged, we can remove this line.
	_ = verkle.GetConfig()
	return &keyValueMigrator{
		processingReady: make(chan struct{}),
		leafData:        make([]migratedKeyValue, 0, 10_000),
	}
}

type migratedKeyValue struct {
	branchKey    branchKey
	leafNodeData verkle.BatchNewLeafNodeData
}
type branchKey struct {
	addr      common.Address
	treeIndex uint256.Int
}

func newBranchKey(addr []byte, treeIndex *uint256.Int) branchKey {
	var sk branchKey
	copy(sk.addr[:], addr)
	sk.treeIndex = *treeIndex
	return sk
}

func (kvm *keyValueMigrator) addStorageSlot(addr []byte, slotNumber []byte, slotValue []byte) {
	treeIndex, subIndex := tutils.GetTreeKeyStorageSlotTreeIndexes(slotNumber)
	leafNodeData := kvm.getOrInitLeafNodeData(newBranchKey(addr, treeIndex))
	leafNodeData.Values[subIndex] = slotValue
}

func (kvm *keyValueMigrator) addAccount(addr []byte, acc *types.StateAccount) {
	leafNodeData := kvm.getOrInitLeafNodeData(newBranchKey(addr, &zeroTreeIndex))

	var version [verkle.LeafValueSize]byte
	leafNodeData.Values[tutils.VersionLeafKey] = version[:]

	var balance [verkle.LeafValueSize]byte
	for i, b := range acc.Balance.Bytes() {
		balance[len(acc.Balance.Bytes())-1-i] = b
	}
	leafNodeData.Values[tutils.BalanceLeafKey] = balance[:]

	var nonce [verkle.LeafValueSize]byte
	binary.LittleEndian.PutUint64(nonce[:8], acc.Nonce)
	leafNodeData.Values[tutils.NonceLeafKey] = nonce[:]

	leafNodeData.Values[tutils.CodeKeccakLeafKey] = acc.CodeHash[:]
}

func (kvm *keyValueMigrator) addAccountCode(addr []byte, codeSize uint64, chunks []byte) {
	leafNodeData := kvm.getOrInitLeafNodeData(newBranchKey(addr, &zeroTreeIndex))

	// Save the code size.
	var codeSizeBytes [verkle.LeafValueSize]byte
	binary.LittleEndian.PutUint64(codeSizeBytes[:8], codeSize)
	leafNodeData.Values[tutils.CodeSizeLeafKey] = codeSizeBytes[:]

	// The first 128 chunks are stored in the account header leaf.
	for i := 0; i < 128 && i < len(chunks)/32; i++ {
		leafNodeData.Values[byte(128+i)] = chunks[32*i : 32*(i+1)]
	}

	// Potential further chunks, have their own leaf nodes.
	for i := 128; i < len(chunks)/32; {
		treeIndex, _ := tutils.GetTreeKeyCodeChunkIndices(uint256.NewInt(uint64(i)))
		leafNodeData := kvm.getOrInitLeafNodeData(newBranchKey(addr, treeIndex))

		j := i
		for ; (j-i) < 256 && j < len(chunks)/32; j++ {
			leafNodeData.Values[byte((j-128)%256)] = chunks[32*j : 32*(j+1)]
		}
		i = j
	}
}

func (kvm *keyValueMigrator) getOrInitLeafNodeData(bk branchKey) *verkle.BatchNewLeafNodeData {
	// Remember that keyValueMigration receives actions ordered by (address, subtreeIndex).
	// This means that we can assume that the last element of leafData is the one that we
	// are looking for, or that we need to create a new one.
	if len(kvm.leafData) == 0 || kvm.leafData[len(kvm.leafData)-1].branchKey != bk {
		kvm.leafData = append(kvm.leafData, migratedKeyValue{
			branchKey: bk,
			leafNodeData: verkle.BatchNewLeafNodeData{
				Stem:   nil, // It will be calculated in the prepare() phase, since it's CPU heavy.
				Values: make(map[byte][]byte),
			},
		})
	}
	return &kvm.leafData[len(kvm.leafData)-1].leafNodeData
}

func (kvm *keyValueMigrator) prepare() {
	// We fire a background routine to process the leafData and save the result in newLeaves.
	// The background routine signals that it is done by closing processingReady.
	go func() {
		// Step 1: We split kvm.leafData in numBatches batches, and we process each batch in a separate goroutine.
		//         This fills each leafNodeData.Stem with the correct value.
		var wg sync.WaitGroup
		batchNum := runtime.NumCPU()
		batchSize := (len(kvm.leafData) + batchNum - 1) / batchNum
		for i := 0; i < len(kvm.leafData); i += batchSize {
			start := i
			end := i + batchSize
			if end > len(kvm.leafData) {
				end = len(kvm.leafData)
			}
			wg.Add(1)

			batch := kvm.leafData[start:end]
			go func() {
				defer wg.Done()
				var currAddr common.Address
				var currPoint *verkle.Point
				for i := range batch {
					if batch[i].branchKey.addr != currAddr {
						currAddr = batch[i].branchKey.addr
						currPoint = tutils.EvaluateAddressPoint(currAddr[:])
					}
					stem := tutils.GetTreeKeyWithEvaluatedAddess(currPoint, &batch[i].branchKey.treeIndex, 0)
					stem = stem[:verkle.StemSize]
					batch[i].leafNodeData.Stem = stem
				}
			}()
		}
		wg.Wait()

		// Step 2: Now that we have all stems (i.e: tree keys) calculated, we can create the new leaves.
		nodeValues := make([]verkle.BatchNewLeafNodeData, len(kvm.leafData))
		for i := range kvm.leafData {
			nodeValues[i] = kvm.leafData[i].leafNodeData
		}

		// Create all leaves in batch mode so we can optimize cryptography operations.
		kvm.newLeaves, kvm.prepareErr = verkle.BatchNewLeafNode(nodeValues)
		close(kvm.processingReady)
	}()
}

func (kvm *keyValueMigrator) migrateCollectedKeyValues(tree *trie.VerkleTrie) error {
	now := time.Now()
	<-kvm.processingReady
	if kvm.prepareErr != nil {
		return fmt.Errorf("failed to prepare key values: %w", kvm.prepareErr)
	}
	log.Info("Prepared key values from base tree", "duration", time.Since(now))

	// Insert into the tree.
	if err := tree.InsertMigratedLeaves(kvm.newLeaves); err != nil {
		return fmt.Errorf("failed to insert migrated leaves: %w", err)
	}

	return nil
}
