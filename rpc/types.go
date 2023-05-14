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

package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// API describes the set of methods offered over the RPC interface
type API struct {
	Namespace     string      // namespace under which the rpc methods of Service are exposed
	Version       string      // deprecated - this field is no longer used, but retained for compatibility
	Service       interface{} // receiver instance which holds the methods
	Public        bool        // deprecated - this field is no longer used, but retained for compatibility
	Authenticated bool        // whether the api should only be available behind authentication.
}

// ServerCodec implements reading, parsing and writing RPC messages for the server side of
// a RPC session. Implementations must be go-routine safe since the codec can be called in
// multiple go-routines concurrently.
type ServerCodec interface {
	peerInfo() PeerInfo
	readBatch() (msgs []*jsonrpcMessage, isBatch bool, err error)
	close()

	jsonWriter
}

// jsonWriter can write JSON messages to its underlying connection.
// Implementations must be safe for concurrent use.
type jsonWriter interface {
	// writeJSON writes a message to the connection.
	writeJSON(ctx context.Context, msg interface{}, isError bool) error

	// Closed returns a channel which is closed when the connection is closed.
	closed() <-chan interface{}
	// RemoteAddr returns the peer address of the connection.
	remoteAddr() string
}

type BlockNumber int64

const (
	SafeBlockNumber      = BlockNumber(-4)
	FinalizedBlockNumber = BlockNumber(-3)
	PendingBlockNumber   = BlockNumber(-2)
	LatestBlockNumber    = BlockNumber(-1)
	EarliestBlockNumber  = BlockNumber(0)
)

// UnmarshalJSON parses the given JSON fragment into a BlockNumber. It supports:
// - "safe", "finalized", "latest", "earliest" or "pending" as string arguments
// - the block number
// Returned errors:
// - an invalid block number error when the given argument isn't a known strings
// - an out of range error when the given block number is either too little or too large
func (bn *BlockNumber) UnmarshalJSON(data []byte) error {
	input := strings.TrimSpace(string(data))
	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		input = input[1 : len(input)-1]
	}

	switch input {
	case "earliest":
		*bn = EarliestBlockNumber
		return nil
	case "latest":
		*bn = LatestBlockNumber
		return nil
	case "pending":
		*bn = PendingBlockNumber
		return nil
	case "finalized":
		*bn = FinalizedBlockNumber
		return nil
	case "safe":
		*bn = SafeBlockNumber
		return nil
	}

	blckNum, err := hexutil.DecodeUint64(input)
	if err != nil {
		return err
	}
	if blckNum > math.MaxInt64 {
		return fmt.Errorf("block number larger than int64")
	}
	*bn = BlockNumber(blckNum)
	return nil
}

// MarshalText implements encoding.TextMarshaler. It marshals:
// - "safe", "finalized", "latest", "earliest" or "pending" as strings
// - other numbers as hex
func (bn BlockNumber) MarshalText() ([]byte, error) {
	switch bn {
	case EarliestBlockNumber:
		return []byte("earliest"), nil
	case LatestBlockNumber:
		return []byte("latest"), nil
	case PendingBlockNumber:
		return []byte("pending"), nil
	case FinalizedBlockNumber:
		return []byte("finalized"), nil
	case SafeBlockNumber:
		return []byte("safe"), nil
	default:
		return hexutil.Uint64(bn).MarshalText()
	}
}

func (bn BlockNumber) Int64() int64 {
	return (int64)(bn)
}

type BlockNumberOrHash struct {
	BlockNumber      *BlockNumber `json:"blockNumber,omitempty"`
	BlockHash        *common.Hash `json:"blockHash,omitempty"`
	RequireCanonical bool         `json:"requireCanonical,omitempty"`
}

func (bnh *BlockNumberOrHash) UnmarshalJSON(data []byte) error {
	type erased BlockNumberOrHash
	e := erased{}
	err := json.Unmarshal(data, &e)
	if err == nil {
		if e.BlockNumber != nil && e.BlockHash != nil {
			return fmt.Errorf("cannot specify both BlockHash and BlockNumber, choose one or the other")
		}
		bnh.BlockNumber = e.BlockNumber
		bnh.BlockHash = e.BlockHash
		bnh.RequireCanonical = e.RequireCanonical
		return nil
	}
	var input string
	err = json.Unmarshal(data, &input)
	if err != nil {
		return err
	}
	switch input {
	case "earliest":
		bn := EarliestBlockNumber
		bnh.BlockNumber = &bn
		return nil
	case "latest":
		bn := LatestBlockNumber
		bnh.BlockNumber = &bn
		return nil
	case "pending":
		bn := PendingBlockNumber
		bnh.BlockNumber = &bn
		return nil
	case "finalized":
		bn := FinalizedBlockNumber
		bnh.BlockNumber = &bn
		return nil
	case "safe":
		bn := SafeBlockNumber
		bnh.BlockNumber = &bn
		return nil
	default:
		if len(input) == 66 {
			hash := common.Hash{}
			err := hash.UnmarshalText([]byte(input))
			if err != nil {
				return err
			}
			bnh.BlockHash = &hash
			return nil
		} else {
			blckNum, err := hexutil.DecodeUint64(input)
			if err != nil {
				return err
			}
			if blckNum > math.MaxInt64 {
				return fmt.Errorf("blocknumber too high")
			}
			bn := BlockNumber(blckNum)
			bnh.BlockNumber = &bn
			return nil
		}
	}
}

func (bnh *BlockNumberOrHash) Number() (BlockNumber, bool) {
	if bnh.BlockNumber != nil {
		return *bnh.BlockNumber, true
	}
	return BlockNumber(0), false
}

func (bnh *BlockNumberOrHash) String() string {
	if bnh.BlockNumber != nil {
		return strconv.Itoa(int(*bnh.BlockNumber))
	}
	if bnh.BlockHash != nil {
		return bnh.BlockHash.String()
	}
	return "nil"
}

func (bnh *BlockNumberOrHash) Hash() (common.Hash, bool) {
	if bnh.BlockHash != nil {
		return *bnh.BlockHash, true
	}
	return common.Hash{}, false
}

func BlockNumberOrHashWithNumber(blockNr BlockNumber) BlockNumberOrHash {
	return BlockNumberOrHash{
		BlockNumber:      &blockNr,
		BlockHash:        nil,
		RequireCanonical: false,
	}
}

func BlockNumberOrHashWithHash(hash common.Hash, canonical bool) BlockNumberOrHash {
	return BlockNumberOrHash{
		BlockNumber:      nil,
		BlockHash:        &hash,
		RequireCanonical: canonical,
	}
}

// RPCBlock represents a block with header that will serialize to the RPC representation of a block
type RPCBlock struct {
	Number          *hexutil.Big      `json:"number,omitempty"`
	Hash            *common.Hash      `json:"hash,omitempty"`
	ParentHash      common.Hash       `json:"parentHash"`
	Nonce           *types.BlockNonce `json:"nonce,omitempty"`
	MixDigest       common.Hash       `json:"mixHash"`
	UncleHash       common.Hash       `json:"sha3Uncles"`
	Bloom           types.Bloom       `json:"logsBloom"`
	Root            common.Hash       `json:"stateRoot"`
	Coinbase        *common.Address   `json:"miner,omitempty"`
	Difficulty      *hexutil.Big      `json:"difficulty"`
	Extra           hexutil.Bytes     `json:"extraData"`
	Size            hexutil.Uint64    `json:"size,omitempty"`
	GasLimit        hexutil.Uint64    `json:"gasLimit"`
	GasUsed         hexutil.Uint64    `json:"gasUsed"`
	Time            hexutil.Uint64    `json:"timestamp"`
	TxHash          common.Hash       `json:"transactionsRoot"`
	ReceiptHash     common.Hash       `json:"receiptsRoot"`
	BaseFee         *hexutil.Big      `json:"baseFeePerGas,omitempty"`
	WithdrawalsHash *common.Hash      `json:"withdrawalsRoot,omitempty"`
	Transactions    []json.RawMessage `json:"transactions,omitempty"`
	TotalDifficulty *hexutil.Big      `json:"totalDifficulty,omitempty"`
	Uncles          []common.Hash     `json:"uncles,omitempty"`
	Withdrawals     types.Withdrawals `json:"withdrawls,omitempty"`
}

// RPCMarshalHeader converts the given header to the RPC output .
func RPCMarshalHeader(head *types.Header) *RPCBlock {
	headHash := head.Hash()
	block := &RPCBlock{
		Number:      (*hexutil.Big)(head.Number),
		Hash:        &headHash,
		ParentHash:  head.ParentHash,
		Nonce:       &head.Nonce,
		MixDigest:   head.MixDigest,
		UncleHash:   head.UncleHash,
		Bloom:       head.Bloom,
		Root:        head.Root,
		Coinbase:    &head.Coinbase,
		Difficulty:  (*hexutil.Big)(head.Difficulty),
		Extra:       hexutil.Bytes(head.Extra),
		Size:        hexutil.Uint64(head.Size()),
		GasLimit:    hexutil.Uint64(head.GasLimit),
		GasUsed:     hexutil.Uint64(head.GasUsed),
		Time:        hexutil.Uint64(head.Time),
		TxHash:      head.TxHash,
		ReceiptHash: head.ReceiptHash,
	}

	if head.BaseFee != nil {
		block.BaseFee = (*hexutil.Big)(head.BaseFee)
	}

	if head.WithdrawalsHash != nil {
		block.WithdrawalsHash = head.WithdrawalsHash
	}

	return block
}

// RPCMarshalBlock converts the given block to the RPC output which depends on fullTx. If inclTx is true transactions are
// returned. When fullTx is true the returned block contains full transaction details, otherwise it will only contain
// transaction hashes.
func RPCMarshalBlock(block *types.Block, inclTx bool, fullTx bool, config *params.ChainConfig) (*RPCBlock, error) {
	fields := RPCMarshalHeader(block.Header())
	fields.Size = hexutil.Uint64(block.Size())

	if inclTx {
		formatTx := func(tx *types.Transaction) (json.RawMessage, error) {
			return tx.Hash().Bytes(), nil
		}
		if fullTx {
			formatTx = func(tx *types.Transaction) (json.RawMessage, error) {
				return json.Marshal(newRPCTransactionFromBlockHash(block, tx.Hash(), config))
			}
		}
		txs := block.Transactions()
		transactions := make([]json.RawMessage, len(txs))
		var err error
		for i, tx := range txs {
			if transactions[i], err = formatTx(tx); err != nil {
				return nil, err
			}
		}
		fields.Transactions = transactions
	}
	uncles := block.Uncles()
	uncleHashes := make([]common.Hash, len(uncles))
	for i, uncle := range uncles {
		uncleHashes[i] = uncle.Hash()
	}
	fields.Uncles = uncleHashes
	if block.Header().WithdrawalsHash != nil {
		fields.Withdrawals = block.Withdrawals()
	}
	return fields, nil
}

// newRPCTransactionFromBlockHash returns a transaction that will serialize to the RPC representation.
func newRPCTransactionFromBlockHash(b *types.Block, hash common.Hash, config *params.ChainConfig) *RPCTransaction {
	for idx, tx := range b.Transactions() {
		if tx.Hash() == hash {
			return NewRPCTransactionFromBlockIndex(b, uint64(idx), config)
		}
	}
	return nil
}

// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction
type RPCTransaction struct {
	BlockHash        *common.Hash      `json:"blockHash"`
	BlockNumber      *hexutil.Big      `json:"blockNumber"`
	From             common.Address    `json:"from"`
	Gas              hexutil.Uint64    `json:"gas"`
	GasPrice         *hexutil.Big      `json:"gasPrice"`
	GasFeeCap        *hexutil.Big      `json:"maxFeePerGas,omitempty"`
	GasTipCap        *hexutil.Big      `json:"maxPriorityFeePerGas,omitempty"`
	Hash             common.Hash       `json:"hash"`
	Input            hexutil.Bytes     `json:"input"`
	Nonce            hexutil.Uint64    `json:"nonce"`
	To               *common.Address   `json:"to"`
	TransactionIndex *hexutil.Uint64   `json:"transactionIndex"`
	Value            *hexutil.Big      `json:"value"`
	Type             hexutil.Uint64    `json:"type"`
	Accesses         *types.AccessList `json:"accessList,omitempty"`
	ChainID          *hexutil.Big      `json:"chainId,omitempty"`
	V                *hexutil.Big      `json:"v"`
	R                *hexutil.Big      `json:"r"`
	S                *hexutil.Big      `json:"s"`
}

// NewRPCTransaction returns a transaction that will serialize to the RPC
// representation, with the given location metadata set (if available).
func NewRPCTransaction(tx *types.Transaction, blockHash common.Hash, blockNumber uint64, blockTime uint64, index uint64, baseFee *big.Int, config *params.ChainConfig) *RPCTransaction {
	signer := types.MakeSigner(config, new(big.Int).SetUint64(blockNumber), blockTime)
	from, _ := types.Sender(signer, tx)
	v, r, s := tx.RawSignatureValues()
	result := &RPCTransaction{
		Type:     hexutil.Uint64(tx.Type()),
		From:     from,
		Gas:      hexutil.Uint64(tx.Gas()),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		Hash:     tx.Hash(),
		Input:    hexutil.Bytes(tx.Data()),
		Nonce:    hexutil.Uint64(tx.Nonce()),
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
	}
	if blockHash != (common.Hash{}) {
		result.BlockHash = &blockHash
		result.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(blockNumber))
		result.TransactionIndex = (*hexutil.Uint64)(&index)
	}
	switch tx.Type() {
	case types.LegacyTxType:
		// if a legacy transaction has an EIP-155 chain id, include it explicitly
		if id := tx.ChainId(); id.Sign() != 0 {
			result.ChainID = (*hexutil.Big)(id)
		}
	case types.AccessListTxType:
		al := tx.AccessList()
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
	case types.DynamicFeeTxType:
		al := tx.AccessList()
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
		result.GasFeeCap = (*hexutil.Big)(tx.GasFeeCap())
		result.GasTipCap = (*hexutil.Big)(tx.GasTipCap())
		// if the transaction has been mined, compute the effective gas price
		if baseFee != nil && blockHash != (common.Hash{}) {
			// price = min(tip, gasFeeCap - baseFee) + baseFee
			price := math.BigMin(new(big.Int).Add(tx.GasTipCap(), baseFee), tx.GasFeeCap())
			result.GasPrice = (*hexutil.Big)(price)
		} else {
			result.GasPrice = (*hexutil.Big)(tx.GasFeeCap())
		}
	}
	return result
}

// NewRPCPendingTransaction returns a pending transaction that will serialize to the RPC representation
func NewRPCPendingTransaction(tx *types.Transaction, current *types.Header, config *params.ChainConfig) *RPCTransaction {
	var (
		baseFee     *big.Int
		blockNumber = uint64(0)
		blockTime   = uint64(0)
	)
	if current != nil {
		baseFee = misc.CalcBaseFee(config, current)
		blockNumber = current.Number.Uint64()
		blockTime = current.Time
	}
	return NewRPCTransaction(tx, common.Hash{}, blockNumber, blockTime, 0, baseFee, config)
}

// NewRPCTransactionFromBlockIndex returns a transaction that will serialize to the RPC representation.
func NewRPCTransactionFromBlockIndex(b *types.Block, index uint64, config *params.ChainConfig) *RPCTransaction {
	txs := b.Transactions()
	if index >= uint64(len(txs)) {
		return nil
	}
	return NewRPCTransaction(txs[index], b.Hash(), b.NumberU64(), b.Time(), index, b.BaseFee(), config)
}

// NewRPCRawTransactionFromBlockIndex returns the bytes of a transaction given a block and a transaction index.
func NewRPCRawTransactionFromBlockIndex(b *types.Block, index uint64) hexutil.Bytes {
	txs := b.Transactions()
	if index >= uint64(len(txs)) {
		return nil
	}
	blob, _ := txs[index].MarshalBinary()
	return blob
}
