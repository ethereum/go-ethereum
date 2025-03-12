package l1

import (
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

var (
	// ScrollChainABI holds information about ScrollChain's context and available invokable methods.
	ScrollChainABI *abi.ABI
	// L1MessageQueueABIManual holds information about L1MessageQueue's context and available invokable methods.
	L1MessageQueueABIManual *abi.ABI
)

func init() {
	ScrollChainABI, _ = ScrollChainMetaData.GetAbi()
	L1MessageQueueABIManual, _ = L1MessageQueueMetaDataManual.GetAbi()
}

// ScrollChainMetaData contains ABI of the ScrollChain contract.
var ScrollChainMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"commitAndFinalizeBatch\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"parentBatchHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"finalizeStruct\",\"type\":\"tuple\",\"internalType\":\"struct ScrollChainInterface.FinalizeStruct\",\"components\":[{\"name\":\"batchHeader\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"totalL1MessagesPoppedOverall\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"postStateRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"withdrawRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"zkProof\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"commitBatch\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"parentBatchHeader\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"chunks\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"},{\"name\":\"skippedL1MessageBitmap\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"commitBatchWithBlobProof\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"parentBatchHeader\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"chunks\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"},{\"name\":\"skippedL1MessageBitmap\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"blobDataProof\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"commitBatches\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"parentBatchHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"lastBatchHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"committedBatches\",\"inputs\":[{\"name\":\"batchIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"finalizeBatch\",\"inputs\":[{\"name\":\"_batchHeader\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"_prevStateRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_postStateRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_withdrawRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"finalizeBatch4844\",\"inputs\":[{\"name\":\"_batchHeader\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_postStateRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_withdrawRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_blobDataProof\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"finalizeBatchWithProof\",\"inputs\":[{\"name\":\"batchHeader\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"prevStateRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"postStateRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"withdrawRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"aggrProof\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"finalizeBatchWithProof4844\",\"inputs\":[{\"name\":\"batchHeader\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"prevStateRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"postStateRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"withdrawRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"blobDataProof\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"aggrProof\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"finalizeBundle\",\"inputs\":[{\"name\":\"batchHeader\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"postStateRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"withdrawRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"finalizeBundlePostEuclidV2\",\"inputs\":[{\"name\":\"batchHeader\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"totalL1MessagesPoppedOverall\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"postStateRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"withdrawRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"aggrProof\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"finalizeBundlePostEuclidV2NoProof\",\"inputs\":[{\"name\":\"batchHeader\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"totalL1MessagesPoppedOverall\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"postStateRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"withdrawRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"finalizeBundleWithProof\",\"inputs\":[{\"name\":\"batchHeader\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"postStateRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"withdrawRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"aggrProof\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"finalizeEuclidInitialBatch\",\"inputs\":[{\"name\":\"postStateRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"finalizedStateRoots\",\"inputs\":[{\"name\":\"batchIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"importGenesisBatch\",\"inputs\":[{\"name\":\"_batchHeader\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"_stateRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"isBatchFinalized\",\"inputs\":[{\"name\":\"batchIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"lastFinalizedBatchIndex\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"revertBatch\",\"inputs\":[{\"name\":\"batchHeader\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"count\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"revertBatch\",\"inputs\":[{\"name\":\"batchHeader\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"withdrawRoots\",\"inputs\":[{\"name\":\"batchIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"CommitBatch\",\"inputs\":[{\"name\":\"batchIndex\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"},{\"name\":\"batchHash\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"FinalizeBatch\",\"inputs\":[{\"name\":\"batchIndex\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"},{\"name\":\"batchHash\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"stateRoot\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"withdrawRoot\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RevertBatch\",\"inputs\":[{\"name\":\"batchIndex\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"},{\"name\":\"batchHash\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RevertBatch\",\"inputs\":[{\"name\":\"startBatchIndex\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"},{\"name\":\"finishBatchIndex\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"UpdateEnforcedBatchMode\",\"inputs\":[{\"name\":\"enabled\",\"type\":\"bool\",\"indexed\":false,\"internalType\":\"bool\"},{\"name\":\"lastCommittedBatchIndex\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false}]",
}

// L1MessageQueueMetaDataManual contains all meta data concerning the L1MessageQueue contract.
var L1MessageQueueMetaDataManual = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_messenger\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_scrollChain\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_enforcedTxGateway\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"appendCrossDomainMessage\",\"inputs\":[{\"name\":\"_target\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_gasLimit\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"appendEnforcedTransaction\",\"inputs\":[{\"name\":\"_sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_target\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_gasLimit\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"calculateIntrinsicGasFee\",\"inputs\":[{\"name\":\"_calldata\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"computeTransactionHash\",\"inputs\":[{\"name\":\"_sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_queueIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_value\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_target\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_gasLimit\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"dropCrossDomainMessage\",\"inputs\":[{\"name\":\"_index\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"enforcedTxGateway\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"estimateCrossDomainMessageFee\",\"inputs\":[{\"name\":\"_gasLimit\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"finalizePoppedCrossDomainMessage\",\"inputs\":[{\"name\":\"_newFinalizedQueueIndexPlusOne\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"gasOracle\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getCrossDomainMessage\",\"inputs\":[{\"name\":\"_queueIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_messenger\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_scrollChain\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_enforcedTxGateway\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_gasOracle\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_maxGasLimit\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"isMessageDropped\",\"inputs\":[{\"name\":\"_queueIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"isMessageSkipped\",\"inputs\":[{\"name\":\"_queueIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"maxGasLimit\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"messageQueue\",\"inputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"messenger\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"nextCrossDomainMessageIndex\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"nextUnfinalizedQueueIndex\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"pendingQueueIndex\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"popCrossDomainMessage\",\"inputs\":[{\"name\":\"_startIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_count\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_skippedBitmap\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"resetPoppedCrossDomainMessage\",\"inputs\":[{\"name\":\"_startIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"scrollChain\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"updateGasOracle\",\"inputs\":[{\"name\":\"_newGasOracle\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"updateMaxGasLimit\",\"inputs\":[{\"name\":\"_newMaxGasLimit\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"DequeueTransaction\",\"inputs\":[{\"name\":\"startIndex\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"count\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"skippedBitmap\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DropTransaction\",\"inputs\":[{\"name\":\"index\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"FinalizedDequeuedTransaction\",\"inputs\":[{\"name\":\"finalizedIndex\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"uint8\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"QueueTransaction\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"target\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"queueIndex\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"},{\"name\":\"gasLimit\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ResetDequeuedTransaction\",\"inputs\":[{\"name\":\"startIndex\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"UpdateGasOracle\",\"inputs\":[{\"name\":\"_oldGasOracle\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"_newGasOracle\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"UpdateMaxGasLimit\",\"inputs\":[{\"name\":\"_oldMaxGasLimit\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"_newMaxGasLimit\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"ErrorZeroAddress\",\"inputs\":[]}]",
}

const (
	// CommitEventType contains data of event of commit batch
	CommitEventType int = iota
	// RevertEventV0Type contains data of event of revert batch from V0 to V6
	RevertEventV0Type
	// RevertEventV7Type contains data of event of revert batch after V7 (EuclidV2)
	RevertEventV7Type
	// FinalizeEventType contains data of event of finalize batch
	FinalizeEventType

	commitBatchMethodName              = "commitBatch"
	commitBatchWithBlobProofMethodName = "commitBatchWithBlobProof"
	commitBatchesV7MethodName          = "commitBatches"
	commitAndFinalizeBatch             = "commitAndFinalizeBatch"

	finalizeBundlePostEuclidV2MethodName = "finalizeBundlePostEuclidV2"

	// the length of method ID at the beginning of transaction data
	methodIDLength = 4
)

// RollupEvent represents a single rollup event (commit, revert, finalize)
type RollupEvent interface {
	Type() int
	BatchIndex() *big.Int
	BatchHash() common.Hash
	TxHash() common.Hash
	BlockHash() common.Hash
	BlockNumber() uint64
}

type RollupEvents []RollupEvent

// CommitBatchEventUnpacked represents a CommitBatch event raised by the ScrollChain contract.
type CommitBatchEventUnpacked struct {
	BatchIndex *big.Int
	BatchHash  common.Hash
}

// CommitBatchEvent represents a CommitBatch event raised by the ScrollChain contract with additional fields.
type CommitBatchEvent struct {
	batchIndex  *big.Int
	batchHash   common.Hash
	txHash      common.Hash
	blockHash   common.Hash
	blockNumber uint64
}

func (c *CommitBatchEvent) Type() int {
	return CommitEventType
}

func (c *CommitBatchEvent) BatchIndex() *big.Int {
	return c.batchIndex
}

func (c *CommitBatchEvent) BatchHash() common.Hash {
	return c.batchHash
}

func (c *CommitBatchEvent) TxHash() common.Hash {
	return c.txHash
}

func (c *CommitBatchEvent) BlockHash() common.Hash {
	return c.blockHash
}

func (c *CommitBatchEvent) BlockNumber() uint64 {
	return c.blockNumber
}

func (c *CommitBatchEvent) CompareTo(other *CommitBatchEvent) int {
	return c.batchIndex.Cmp(other.batchIndex)
}

type RevertBatchEventV0Unpacked struct {
	BatchIndex *big.Int
	BatchHash  common.Hash
}

// RevertBatchEventV0 represents a RevertBatch event raised by the ScrollChain contract from V0 to V6.
type RevertBatchEventV0 struct {
	batchIndex  *big.Int
	batchHash   common.Hash
	txHash      common.Hash
	blockHash   common.Hash
	blockNumber uint64
}

func (r *RevertBatchEventV0) BlockNumber() uint64 {
	return r.blockNumber
}

func (r *RevertBatchEventV0) BlockHash() common.Hash {
	return r.blockHash
}

func (r *RevertBatchEventV0) TxHash() common.Hash {
	return r.txHash
}

func (r *RevertBatchEventV0) Type() int {
	return RevertEventV0Type
}

func (r *RevertBatchEventV0) BatchIndex() *big.Int {
	return r.batchIndex
}

func (r *RevertBatchEventV0) BatchHash() common.Hash {
	return r.batchHash
}

type RevertBatchEventV7Unpacked struct {
	StartBatchIndex  *big.Int
	FinishBatchIndex *big.Int
}

// RevertBatchEventV7 represents a RevertBatch event raised by the ScrollChain contract after V7 (EuclidV2).
// It reverts a range of batches from startBatchIndex (inclusive) to finishBatchIndex (inclusive).
type RevertBatchEventV7 struct {
	startBatchIndex  *big.Int
	finishBatchIndex *big.Int

	txHash      common.Hash
	blockHash   common.Hash
	blockNumber uint64
}

func (r *RevertBatchEventV7) BlockNumber() uint64 {
	return r.blockNumber
}

func (r *RevertBatchEventV7) BlockHash() common.Hash {
	return r.blockHash
}

func (r *RevertBatchEventV7) TxHash() common.Hash {
	return r.txHash
}

func (r *RevertBatchEventV7) Type() int {
	return RevertEventV7Type
}

// BatchIndex returns the start batch index of the reverted range.
func (r *RevertBatchEventV7) BatchIndex() *big.Int {
	return r.startBatchIndex
}

// BatchHash is not applicable for RevertBatchEventV7.
func (r *RevertBatchEventV7) BatchHash() common.Hash {
	return common.Hash{}
}

// StartBatchIndex returns the start batch index of the reverted range.
func (r *RevertBatchEventV7) StartBatchIndex() *big.Int {
	return r.startBatchIndex
}

// FinishBatchIndex returns the finish batch index of the reverted range.
func (r *RevertBatchEventV7) FinishBatchIndex() *big.Int {
	return r.finishBatchIndex
}

type FinalizeBatchEventUnpacked struct {
	BatchIndex   *big.Int
	BatchHash    common.Hash
	StateRoot    common.Hash
	WithdrawRoot common.Hash
}

// FinalizeBatchEvent represents a FinalizeBatch event raised by the ScrollChain contract.
type FinalizeBatchEvent struct {
	batchIndex   *big.Int
	batchHash    common.Hash
	stateRoot    common.Hash
	withdrawRoot common.Hash
	txHash       common.Hash
	blockHash    common.Hash
	blockNumber  uint64
}

func NewFinalizeBatchEvent(
	batchIndex *big.Int,
	batchHash common.Hash,
	stateRoot common.Hash,
	withdrawRoot common.Hash,
	txHash common.Hash,
	blockHash common.Hash,
	blockNumber uint64,
) *FinalizeBatchEvent {
	return &FinalizeBatchEvent{
		batchIndex:   batchIndex,
		batchHash:    batchHash,
		stateRoot:    stateRoot,
		withdrawRoot: withdrawRoot,
		txHash:       txHash,
		blockHash:    blockHash,
		blockNumber:  blockNumber,
	}
}

func (f *FinalizeBatchEvent) TxHash() common.Hash {
	return f.txHash
}

func (f *FinalizeBatchEvent) BlockHash() common.Hash {
	return f.blockHash
}

func (f *FinalizeBatchEvent) BlockNumber() uint64 {
	return f.blockNumber
}

func (f *FinalizeBatchEvent) Type() int {
	return FinalizeEventType
}

func (f *FinalizeBatchEvent) BatchIndex() *big.Int {
	return f.batchIndex
}

func (f *FinalizeBatchEvent) BatchHash() common.Hash {
	return f.batchHash
}

func (f *FinalizeBatchEvent) StateRoot() common.Hash {
	return f.stateRoot
}

func (f *FinalizeBatchEvent) WithdrawRoot() common.Hash {
	return f.withdrawRoot
}

// UnpackLog unpacks a retrieved log into the provided output structure.
func UnpackLog(c *abi.ABI, out interface{}, event string, log types.Log) error {
	if log.Topics[0] != c.Events[event].ID {
		return fmt.Errorf("event signature mismatch")
	}
	if len(log.Data) > 0 {
		if err := c.UnpackIntoInterface(out, event, log.Data); err != nil {
			return err
		}
	}
	var indexed abi.Arguments
	for _, arg := range c.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	return abi.ParseTopics(out, indexed, log.Topics[1:])
}

type CommitBatchArgs struct {
	Version                uint8
	ParentBatchHeader      []byte
	Chunks                 [][]byte
	SkippedL1MessageBitmap []byte
	BlobHashes             []common.Hash

	// added in CodecV7
	ParentBatchHash common.Hash
	LastBatchHash   common.Hash
}

func newCommitBatchArgs(method *abi.Method, values []interface{}) (*CommitBatchArgs, error) {
	var args CommitBatchArgs
	err := method.Inputs.Copy(&args, values)
	return &args, err
}

func newCommitBatchArgsFromCommitBatchWithProof(method *abi.Method, values []interface{}) (*CommitBatchArgs, error) {
	var args commitBatchWithBlobProofArgs
	err := method.Inputs.Copy(&args, values)
	if err != nil {
		return nil, err
	}
	return &CommitBatchArgs{
		Version:                args.Version,
		ParentBatchHeader:      args.ParentBatchHeader,
		Chunks:                 args.Chunks,
		SkippedL1MessageBitmap: args.SkippedL1MessageBitmap,
	}, nil
}

func newCommitBatchArgsFromCommitBatchesV7(method *abi.Method, values []any) (*CommitBatchArgs, error) {
	var args commitBatchesV7Args
	err := method.Inputs.Copy(&args, values)
	if err != nil {
		return nil, err
	}

	return &CommitBatchArgs{
		Version:         args.Version,
		ParentBatchHash: args.ParentBatchHash,
		LastBatchHash:   args.LastBatchHash,
	}, nil
}

type commitBatchWithBlobProofArgs struct {
	Version                uint8
	ParentBatchHeader      []byte
	Chunks                 [][]byte
	SkippedL1MessageBitmap []byte
	BlobDataProof          []byte
}

type commitBatchesV7Args struct {
	Version         uint8
	ParentBatchHash common.Hash
	LastBatchHash   common.Hash
}

type FinalizeBatchArgs struct {
	BatchHeader                  []byte
	TotalL1MessagesPoppedOverall *big.Int
	PostStateRoot                common.Hash
	WithdrawRoot                 common.Hash
	AggrProof                    []byte
}

func newFinalizeBatchArgs(method *abi.Method, values []any) (*FinalizeBatchArgs, error) {
	var args FinalizeBatchArgs
	err := method.Inputs.Copy(&args, values)
	if err != nil {
		return nil, err
	}
	return &args, nil
}

type commitAndFinalizeBatchArgs struct {
	Version         uint8
	ParentBatchHash common.Hash
	FinalizeStruct  FinalizeBatchArgs
}

func newCommitAndFinalizeBatchArgs(method *abi.Method, values []any) (*commitAndFinalizeBatchArgs, error) {
	var args commitAndFinalizeBatchArgs
	err := method.Inputs.Copy(&args, values)
	if err != nil {
		return nil, err
	}

	return &args, nil
}
