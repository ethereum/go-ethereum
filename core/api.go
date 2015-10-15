// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
	rpc "github.com/ethereum/go-ethereum/rpc/v2"
	"gopkg.in/fatih/set.v0"
)

const (
	FilterTimeout = 300 * time.Second // Remove filter after FilterTimeout

	defaultGasPrice = uint64(10000000000000)
	defaultGas      = uint64(90000)
)

// PublicBlockChainApi provides an API to access the Ethereum blockchain.
// It offers only methods that operate on public data that is freely available to anyone.
type PublicBlockChainApi struct {
	bc *BlockChain
	am *accounts.Manager
}

// NewPublicBlockChainApi creates a new Etheruem blockchain API.
func NewPublicBlockChainApi(bc *BlockChain, am *accounts.Manager) *PublicBlockChainApi {
	return &PublicBlockChainApi{bc: bc, am: am}
}

// BlockNumber returns the block number of the chain head.
func (s *PublicBlockChainApi) BlockNumber() *big.Int {
	return s.bc.CurrentHeader().Number
}

// GetBalance returns the amount of wei for the given address in the state of the given block number.
// When block number equals rpc.LatestBlockNumber the current block is used.
func (s *PublicBlockChainApi) GetBalance(address common.Address, blockNr rpc.BlockNumber) (*big.Int, error) {
	block := blockByNumber(s.bc, blockNr)
	if block == nil {
		return nil, nil
	}

	state, err := state.New(block.Root(), s.bc.chainDb)
	if err != nil {
		return nil, err
	}
	return state.GetBalance(address), nil
}

// blockByNumber is a commonly used helper function which retrieves and returns the block for the given block number. It
// returns nil when no block could be found.
func blockByNumber(bc *BlockChain, blockNr rpc.BlockNumber) *types.Block {
	if blockNr == rpc.LatestBlockNumber {
		return bc.CurrentBlock()
	}

	return bc.GetBlockByNumber(uint64(blockNr))
}

// GetBlockByNumber returns the requested block. When blockNr is -1 the chain head is returned. When fullTx is true all
// transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainApi) GetBlockByNumber(blockNr rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	if block := blockByNumber(s.bc, blockNr); block != nil {
		return s.rpcOutputBlock(block, true, fullTx)
	}
	return nil, nil
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainApi) GetBlockByHash(blockHash common.Hash, fullTx bool) (map[string]interface{}, error) {
	if block := s.bc.GetBlock(blockHash); block != nil {
		return s.rpcOutputBlock(block, true, fullTx)
	}
	return nil, nil
}

// GetUncleByBlockNumberAndIndex returns the uncle block for the given block hash and index. When fullTx is true
// all transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainApi) GetUncleByBlockNumberAndIndex(blockNr rpc.BlockNumber, index rpc.HexNumber) (map[string]interface{}, error) {
	if blockNr == rpc.PendingBlockNumber {
		return nil, nil
	}

	if block := blockByNumber(s.bc, blockNr); block != nil {
		uncles := block.Uncles()
		if index.Int() < 0 || index.Int() >= len(uncles) {
			glog.V(logger.Debug).Infof("uncle block on index %d not found for block #%d", index.Int(), blockNr)
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index.Int()])
		return s.rpcOutputBlock(block, false, false)
	}
	return nil, nil
}

// GetUncleByBlockHashAndIndex returns the uncle block for the given block hash and index. When fullTx is true
// all transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainApi) GetUncleByBlockHashAndIndex(blockHash common.Hash, index rpc.HexNumber) (map[string]interface{}, error) {
	if block := s.bc.GetBlock(blockHash); block != nil {
		uncles := block.Uncles()
		if index.Int() < 0 || index.Int() >= len(uncles) {
			glog.V(logger.Debug).Infof("uncle block on index %d not found for block %s", index.Int(), blockHash.Hex())
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index.Int()])
		return s.rpcOutputBlock(block, false, false)
	}
	return nil, nil
}

// GetUncleCountByBlockNumber returns number of uncles in the block for the given block number
func (s *PublicBlockChainApi) GetUncleCountByBlockNumber(blockNr rpc.BlockNumber) *rpc.HexNumber {
	if blockNr == rpc.PendingBlockNumber {
		return rpc.NewHexNumber(0)
	}

	if block := blockByNumber(s.bc, blockNr); block != nil {
		return rpc.NewHexNumber(len(block.Uncles()))
	}
	return nil
}

// GetUncleCountByBlockHash returns number of uncles in the block for the given block hash
func (s *PublicBlockChainApi) GetUncleCountByBlockHash(blockHash common.Hash) *rpc.HexNumber {
	if block := s.bc.GetBlock(blockHash); block != nil {
		return rpc.NewHexNumber(len(block.Uncles()))
	}
	return nil
}

// NewBlocksArgs allows the user to specify if the returned block should include transactions and in which format.
type NewBlocksArgs struct {
	IncludeTransactions bool `json:"includeTransactions"`
	TransactionDetails  bool `json:"transactionDetails"`
}

// NewBlocks triggers a new block event each time a block is appended to the chain. It accepts an argument which allows
// the caller to specify whether the output should contain transactions and in what format.
func (s *PublicBlockChainApi) NewBlocks(args NewBlocksArgs) (rpc.Subscription, error) {
	sub := s.bc.eventMux.Subscribe(ChainEvent{})

	output := func(rawBlock interface{}) interface{} {
		if event, ok := rawBlock.(ChainEvent); ok {
			notification, err := s.rpcOutputBlock(event.Block, args.IncludeTransactions, args.TransactionDetails)
			if err == nil {
				return notification
			}
		}
		return rawBlock
	}

	return rpc.NewSubscriptionWithOutputFormat(sub, output), nil
}

// GetCode returns the code stored at the given address in the state for the given block number.
func (s *PublicBlockChainApi) GetCode(address common.Address, blockNr rpc.BlockNumber) (string, error) {
	return s.GetData(address, blockNr)
}

// GetData returns the data stored at the given address in the state for the given block number.
func (s *PublicBlockChainApi) GetData(address common.Address, blockNr rpc.BlockNumber) (string, error) {
	if block := blockByNumber(s.bc, blockNr); block != nil {
		state, err := state.New(block.Root(), s.bc.chainDb)
		if err != nil {
			return "", err
		}
		res := state.GetCode(address)
		if len(res) == 0 { // backwards compatibility
			return "0x", nil
		}
		return common.ToHex(res), nil
	}

	return "0x", nil
}

// GetStorageAt returns the storage from the state at the given address, key and block number.
func (s *PublicBlockChainApi) GetStorageAt(address common.Address, key string, blockNr rpc.BlockNumber) (string, error) {
	if block := blockByNumber(s.bc, blockNr); block != nil {
		state, err := state.New(block.Root(), s.bc.chainDb)
		if err != nil {
			return "", err
		}

		return state.GetState(address, common.HexToHash(key)).Hex(), nil
	}

	return "0x", nil
}

// callmsg is the message type used for call transations.
type callmsg struct {
	from          *state.StateObject
	to            *common.Address
	gas, gasPrice *big.Int
	value         *big.Int
	data          []byte
}

// accessor boilerplate to implement core.Message
func (m callmsg) From() (common.Address, error) { return m.from.Address(), nil }
func (m callmsg) Nonce() uint64                 { return m.from.Nonce() }
func (m callmsg) To() *common.Address           { return m.to }
func (m callmsg) GasPrice() *big.Int            { return m.gasPrice }
func (m callmsg) Gas() *big.Int                 { return m.gas }
func (m callmsg) Value() *big.Int               { return m.value }
func (m callmsg) Data() []byte                  { return m.data }

type CallArgs struct {
	From     common.Address `json:"from"`
	To       common.Address `json:"to"`
	Gas      rpc.HexNumber  `json:"gas"`
	GasPrice rpc.HexNumber  `json:"gasPrice"`
	Value    rpc.HexNumber  `json:"value"`
	Data     string         `json:"data"`
}

func (s *PublicBlockChainApi) doCall(args CallArgs, blockNr rpc.BlockNumber) (string, *big.Int, error) {
	if block := blockByNumber(s.bc, blockNr); block != nil {
		stateDb, err := state.New(block.Root(), s.bc.chainDb)
		if err != nil {
			return "0x", nil, err
		}

		stateDb = stateDb.Copy()
		var from *state.StateObject
		if args.From == (common.Address{}) {
			accounts, err := s.am.Accounts()
			if err != nil || len(accounts) == 0 {
				from = stateDb.GetOrNewStateObject(common.Address{})
			} else {
				from = stateDb.GetOrNewStateObject(accounts[0].Address)
			}
		} else {
			from = stateDb.GetOrNewStateObject(args.From)
		}

		from.SetBalance(common.MaxBig)

		msg := callmsg{
			from:     from,
			to:       &args.To,
			gas:      args.Gas.BigInt(),
			gasPrice: args.GasPrice.BigInt(),
			value:    args.Value.BigInt(),
			data:     common.FromHex(args.Data),
		}

		if msg.gas.Cmp(common.Big0) == 0 {
			msg.gas = big.NewInt(50000000)
		}

		if msg.gasPrice.Cmp(common.Big0) == 0 {
			msg.gasPrice = new(big.Int).Mul(big.NewInt(50), common.Shannon)
		}

		header := s.bc.CurrentBlock().Header()
		vmenv := NewEnv(stateDb, s.bc, msg, header)
		gp := new(GasPool).AddGas(common.MaxBig)
		res, gas, err := ApplyMessage(vmenv, msg, gp)
		if len(res) == 0 { // backwards compatability
			return "0x", gas, err
		}
		return common.ToHex(res), gas, err
	}

	return "0x", common.Big0, nil
}

func (s *PublicBlockChainApi) Call(args CallArgs, blockNr rpc.BlockNumber) (string, error) {
	result, _, err := s.doCall(args, blockNr)
	return result, err
}

func (s *PublicBlockChainApi) EstimateGas(args CallArgs) (*rpc.HexNumber, error) {
	_, gas, err := s.doCall(args, rpc.LatestBlockNumber)
	return rpc.NewHexNumber(gas), err
}

// rpcOutputBlock converts the given block to the RPC output which depends on fullTx. If inclTx is true transactions are
// returned. When fullTx is true the returned block contains full transaction details, otherwise it will only contain
// transaction hashes.
func (s *PublicBlockChainApi) rpcOutputBlock(b *types.Block, inclTx bool, fullTx bool) (map[string]interface{}, error) {
	fields := map[string]interface{}{
		"number":           rpc.NewHexNumber(b.Number()),
		"hash":             b.Hash(),
		"parentHash":       b.ParentHash(),
		"nonce":            b.Header().Nonce,
		"sha3Uncles":       b.UncleHash(),
		"logsBloom":        b.Bloom(),
		"stateRoot":        b.Root(),
		"miner":            b.Coinbase(),
		"difficulty":       rpc.NewHexNumber(b.Difficulty()),
		"totalDifficulty":  rpc.NewHexNumber(s.bc.GetTd(b.Hash())),
		"extraData":        fmt.Sprintf("0x%x", b.Extra()),
		"size":             rpc.NewHexNumber(b.Size().Int64()),
		"gasLimit":         rpc.NewHexNumber(b.GasLimit()),
		"gasUsed":          rpc.NewHexNumber(b.GasUsed()),
		"timestamp":        rpc.NewHexNumber(b.Time()),
		"transactionsRoot": b.TxHash(),
		"receiptRoot":      b.ReceiptHash(),
	}

	if inclTx {
		formatTx := func(tx *types.Transaction) (interface{}, error) {
			return tx.Hash(), nil
		}

		if fullTx {
			formatTx = func(tx *types.Transaction) (interface{}, error) {
				return newRPCTransaction(b, tx.Hash())
			}
		}

		txs := b.Transactions()
		transactions := make([]interface{}, len(txs))
		var err error
		for i, tx := range b.Transactions() {
			if transactions[i], err = formatTx(tx); err != nil {
				return nil, err
			}
		}
		fields["transactions"] = transactions
	}

	uncles := b.Uncles()
	uncleHashes := make([]common.Hash, len(uncles))
	for i, uncle := range uncles {
		uncleHashes[i] = uncle.Hash()
	}
	fields["uncles"] = uncleHashes

	return fields, nil
}

// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction
type RPCTransaction struct {
	BlockHash        common.Hash     `json:"blockHash"`
	BlockNumber      *rpc.HexNumber  `json:"blockNumber"`
	From             common.Address  `json:"from"`
	Gas              *rpc.HexNumber  `json:"gas"`
	GasPrice         *rpc.HexNumber  `json:"gasPrice"`
	Hash             common.Hash     `json:"hash"`
	Input            string          `json:"input"`
	Nonce            *rpc.HexNumber  `json:"nonce"`
	To               *common.Address `json:"to"`
	TransactionIndex *rpc.HexNumber  `json:"transactionIndex"`
	Value            *rpc.HexNumber  `json:"value"`
}

// newRPCPendingTransaction returns a pending transaction that will serialize to the RPC representation
func newRPCPendingTransaction(tx *types.Transaction) *RPCTransaction {
	from, _ := tx.From()

	return &RPCTransaction{
		From:     from,
		Gas:      rpc.NewHexNumber(tx.Gas()),
		GasPrice: rpc.NewHexNumber(tx.GasPrice()),
		Hash:     tx.Hash(),
		Input:    fmt.Sprintf("0x%x", tx.Data()),
		Nonce:    rpc.NewHexNumber(tx.Nonce()),
		To:       tx.To(),
		Value:    rpc.NewHexNumber(tx.Value()),
	}
}

// newRPCTransaction returns a transaction that will serialize to the RPC representation.
func newRPCTransactionFromBlockIndex(b *types.Block, txIndex int) (*RPCTransaction, error) {
	if txIndex >= 0 && txIndex < len(b.Transactions()) {
		tx := b.Transactions()[txIndex]
		from, err := tx.From()
		if err != nil {
			return nil, err
		}

		return &RPCTransaction{
			BlockHash:        b.Hash(),
			BlockNumber:      rpc.NewHexNumber(b.Number()),
			From:             from,
			Gas:              rpc.NewHexNumber(tx.Gas()),
			GasPrice:         rpc.NewHexNumber(tx.GasPrice()),
			Hash:             tx.Hash(),
			Input:            fmt.Sprintf("0x%x", tx.Data()),
			Nonce:            rpc.NewHexNumber(tx.Nonce()),
			To:               tx.To(),
			TransactionIndex: rpc.NewHexNumber(txIndex),
			Value:            rpc.NewHexNumber(tx.Value()),
		}, nil
	}

	return nil, nil
}

// newRPCTransaction returns a transaction that will serialize to the RPC representation.
func newRPCTransaction(b *types.Block, txHash common.Hash) (*RPCTransaction, error) {
	for idx, tx := range b.Transactions() {
		if tx.Hash() == txHash {
			return newRPCTransactionFromBlockIndex(b, idx)
		}
	}

	return nil, nil
}

// PublicTransactionPoolApi exposes methods for the RPC interface
type PublicTransactionPoolApi struct {
	eventMux *event.TypeMux
	chainDb  ethdb.Database
	bc       *BlockChain
	am       *accounts.Manager
	txPool   *TxPool
	txMu     sync.Mutex
}

// NewPublicTransactionPoolApi creates a new RPC service with methods specific for the transaction pool.
func NewPublicTransactionPoolApi(txPool *TxPool, chainDb ethdb.Database, bc *BlockChain, am *accounts.Manager) *PublicTransactionPoolApi {
	return &PublicTransactionPoolApi{
		eventMux: txPool.eventMux,
		chainDb:  chainDb,
		bc:       bc,
		am:       am,
		txPool:   txPool,
	}
}

func getTransaction(chainDb ethdb.Database, txPool *TxPool, txHash common.Hash) (*types.Transaction, bool, error) {
	txData, err := chainDb.Get(txHash.Bytes())
	isPending := false
	tx := new(types.Transaction)

	if err == nil && len(txData) > 0 {
		if err := rlp.DecodeBytes(txData, tx); err != nil {
			return nil, isPending, err
		}
	} else {
		// pending transaction?
		tx = txPool.GetTransaction(txHash)
		isPending = true
	}

	return tx, isPending, nil
}

// GetBlockTransactionCountByNumber returns the number of transactions in the block with the given block number.
func (s *PublicTransactionPoolApi) GetBlockTransactionCountByNumber(blockNr rpc.BlockNumber) *rpc.HexNumber {
	if blockNr == rpc.PendingBlockNumber {
		return rpc.NewHexNumber(0)
	}

	if block := blockByNumber(s.bc, blockNr); block != nil {
		return rpc.NewHexNumber(len(block.Transactions()))
	}

	return nil
}

// GetBlockTransactionCountByHash returns the number of transactions in the block with the given hash.
func (s *PublicTransactionPoolApi) GetBlockTransactionCountByHash(blockHash common.Hash) *rpc.HexNumber {
	if block := s.bc.GetBlock(blockHash); block != nil {
		return rpc.NewHexNumber(len(block.Transactions()))
	}
	return nil
}

// GetTransactionByBlockNumberAndIndex returns the transaction for the given block number and index.
func (s *PublicTransactionPoolApi) GetTransactionByBlockNumberAndIndex(blockNr rpc.BlockNumber, index rpc.HexNumber) (*RPCTransaction, error) {
	if block := blockByNumber(s.bc, blockNr); block != nil {
		return newRPCTransactionFromBlockIndex(block, index.Int())
	}
	return nil, nil
}

// GetTransactionByBlockHashAndIndex returns the transaction for the given block hash and index.
func (s *PublicTransactionPoolApi) GetTransactionByBlockHashAndIndex(blockHash common.Hash, index rpc.HexNumber) (*RPCTransaction, error) {
	if block := s.bc.GetBlock(blockHash); block != nil {
		return newRPCTransactionFromBlockIndex(block, index.Int())
	}
	return nil, nil
}

// GetTransactionCount returns the number of transactions the given address has sent for the given block number
func (s *PublicTransactionPoolApi) GetTransactionCount(address common.Address, blockNr rpc.BlockNumber) (*rpc.HexNumber, error) {
	block := blockByNumber(s.bc, blockNr)
	if block == nil {
		return nil, nil
	}

	state, err := state.New(block.Root(), s.chainDb)
	if err != nil {
		return nil, err
	}
	return rpc.NewHexNumber(state.GetNonce(address)), nil
}

// getTransactionBlockData fetches the meta data for the given transaction from the chain database. This is useful to
// retrieve block information for a hash. It returns the block hash, block index and transaction index.
func getTransactionBlockData(chainDb ethdb.Database, txHash common.Hash) (common.Hash, uint64, uint64, error) {
	var txBlock struct {
		BlockHash  common.Hash
		BlockIndex uint64
		Index      uint64
	}

	blockData, err := chainDb.Get(append(txHash.Bytes(), 0x0001))
	if err != nil {
		return common.Hash{}, uint64(0), uint64(0), err
	}

	reader := bytes.NewReader(blockData)
	if err = rlp.Decode(reader, &txBlock); err != nil {
		return common.Hash{}, uint64(0), uint64(0), err
	}

	return txBlock.BlockHash, txBlock.BlockIndex, txBlock.Index, nil
}

// GetTransactionByHash returns the transaction for the given hash
func (s *PublicTransactionPoolApi) GetTransactionByHash(txHash common.Hash) (*RPCTransaction, error) {
	var tx *types.Transaction
	var isPending bool
	var err error

	if tx, isPending, err = getTransaction(s.chainDb, s.txPool, txHash); err != nil {
		glog.V(logger.Debug).Infof("%v\n", err)
		return nil, nil
	} else if tx == nil {
		return nil, nil
	}

	if isPending {
		return newRPCPendingTransaction(tx), nil
	}

	blockHash, _, _, err := getTransactionBlockData(s.chainDb, txHash)
	if err != nil {
		glog.V(logger.Debug).Infof("%v\n", err)
		return nil, nil
	}

	if block := s.bc.GetBlock(blockHash); block != nil {
		return newRPCTransaction(block, txHash)
	}

	return nil, nil
}

// GetTransactionReceipt returns the transaction receipt for the given transaction hash.
func (s *PublicTransactionPoolApi) GetTransactionReceipt(txHash common.Hash) (map[string]interface{}, error) {
	receipt := GetReceipt(s.chainDb, txHash)
	if receipt == nil {
		glog.V(logger.Debug).Infof("receipt not found for transaction %s", txHash.Hex())
		return nil, nil
	}

	tx, _, err := getTransaction(s.chainDb, s.txPool, txHash)
	if err != nil {
		glog.V(logger.Debug).Infof("%v\n", err)
		return nil, nil
	}

	txBlock, blockIndex, index, err := getTransactionBlockData(s.chainDb, txHash)
	if err != nil {
		glog.V(logger.Debug).Infof("%v\n", err)
		return nil, nil
	}

	from, err := tx.From()
	if err != nil {
		glog.V(logger.Debug).Infof("%v\n", err)
		return nil, nil
	}

	fields := map[string]interface{}{
		"blockHash":         txBlock,
		"blockNumber":       rpc.NewHexNumber(blockIndex),
		"transactionHash":   txHash,
		"transactionIndex":  rpc.NewHexNumber(index),
		"from":              from,
		"to":                tx.To(),
		"gasUsed":           rpc.NewHexNumber(receipt.GasUsed),
		"cumulativeGasUsed": rpc.NewHexNumber(receipt.CumulativeGasUsed),
		"contractAddress":   nil,
		"logs":              receipt.Logs,
	}

	if receipt.Logs == nil {
		fields["logs"] = []vm.Logs{}
	}

	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if bytes.Compare(receipt.ContractAddress.Bytes(), bytes.Repeat([]byte{0}, 20)) != 0 {
		fields["contractAddress"] = receipt.ContractAddress
	}

	return fields, nil
}

// sign is a helper function that signs a transaction with the private key of the given address.
func (s *PublicTransactionPoolApi) sign(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
	acc := accounts.Account{address}
	signature, err := s.am.Sign(acc, tx.SigHash().Bytes())
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(signature)
}

type SendTxArgs struct {
	From     common.Address `json:"from"`
	To       common.Address `json:"to"`
	Gas      *rpc.HexNumber `json:"gas"`
	GasPrice *rpc.HexNumber `json:"gasPrice"`
	Value    *rpc.HexNumber `json:"value"`
	Data     string         `json:"data"`
	Nonce    *rpc.HexNumber `json:"nonce"`
}

// SendTransaction will create a transaction for the given transaction argument, sign it and submit it to the
// transaction pool.
func (s *PublicTransactionPoolApi) SendTransaction(args SendTxArgs) (common.Hash, error) {
	if args.Gas == nil {
		args.Gas = rpc.NewHexNumber(defaultGas)
	}
	if args.GasPrice == nil {
		args.GasPrice = rpc.NewHexNumber(defaultGasPrice)
	}
	if args.Value == nil {
		args.Value = rpc.NewHexNumber(0)
	}

	s.txMu.Lock()
	defer s.txMu.Unlock()

	if args.Nonce == nil {
		args.Nonce = rpc.NewHexNumber(s.txPool.State().GetNonce(args.From))
	}

	var tx *types.Transaction
	contractCreation := (args.To == common.Address{})

	if contractCreation {
		tx = types.NewContractCreation(args.Nonce.Uint64(), args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	} else {
		tx = types.NewTransaction(args.Nonce.Uint64(), args.To, args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	}

	signedTx, err := s.sign(args.From, tx)
	if err != nil {
		return common.Hash{}, err
	}

	if err := s.txPool.Add(signedTx); err != nil {
		return common.Hash{}, nil
	}

	if contractCreation {
		addr := crypto.CreateAddress(args.From, args.Nonce.Uint64())
		glog.V(logger.Info).Infof("Tx(%s) created: %s\n", signedTx.Hash().Hex(), addr.Hex())
	} else {
		glog.V(logger.Info).Infof("Tx(%s) to: %s\n", signedTx.Hash().Hex(), tx.To().Hex())
	}

	return signedTx.Hash(), nil
}

// SendRawTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (s *PublicTransactionPoolApi) SendRawTransaction(encodedTx string) (string, error) {
	tx := new(types.Transaction)
	if err := rlp.DecodeBytes(common.FromHex(encodedTx), tx); err != nil {
		return "", err
	}

	if err := s.txPool.Add(tx); err != nil {
		return "", err
	}

	if tx.To() == nil {
		from, err := tx.From()
		if err != nil {
			return "", err
		}
		addr := crypto.CreateAddress(from, tx.Nonce())
		glog.V(logger.Info).Infof("Tx(%x) created: %x\n", tx.Hash(), addr)
	} else {
		glog.V(logger.Info).Infof("Tx(%x) to: %x\n", tx.Hash(), tx.To())
	}

	return tx.Hash().Hex(), nil
}

// Sign will sign the given data string with the given address. The account corresponding with the address needs to
// be unlocked.
func (s *PublicTransactionPoolApi) Sign(address common.Address, data string) (string, error) {
	signature, error := s.am.Sign(accounts.Account{Address: address}, common.HexToHash(data).Bytes())
	return common.ToHex(signature), error
}

type SignTransactionArgs struct {
	From     common.Address
	To       common.Address
	Nonce    *rpc.HexNumber
	Value    *rpc.HexNumber
	Gas      *rpc.HexNumber
	GasPrice *rpc.HexNumber
	Data     string

	BlockNumber int64
}

// Tx is a helper object for argument and return values
type Tx struct {
	tx *types.Transaction

	To       *common.Address `json:"to"`
	From     common.Address  `json:"from"`
	Nonce    *rpc.HexNumber  `json:"nonce"`
	Value    *rpc.HexNumber  `json:"value"`
	Data     string          `json:"data"`
	GasLimit *rpc.HexNumber  `json:"gas"`
	GasPrice *rpc.HexNumber  `json:"gasPrice"`
	Hash     common.Hash     `json:"hash"`
}

func (tx *Tx) UnmarshalJSON(b []byte) (err error) {
	req := struct {
		To       common.Address `json:"to"`
		From     common.Address `json:"from"`
		Nonce    *rpc.HexNumber `json:"nonce"`
		Value    *rpc.HexNumber `json:"value"`
		Data     string         `json:"data"`
		GasLimit *rpc.HexNumber `json:"gas"`
		GasPrice *rpc.HexNumber `json:"gasPrice"`
		Hash     common.Hash    `json:"hash"`
	}{}

	if err := json.Unmarshal(b, &req); err != nil {
		return err
	}

	contractCreation := (req.To == (common.Address{}))

	tx.To = &req.To
	tx.From = req.From
	tx.Nonce = req.Nonce
	tx.Value = req.Value
	tx.Data = req.Data
	tx.GasLimit = req.GasLimit
	tx.GasPrice = req.GasPrice
	tx.Hash = req.Hash

	data := common.Hex2Bytes(tx.Data)

	if tx.Nonce == nil {
		return fmt.Errorf("need nonce")
	}
	if tx.Value == nil {
		tx.Value = rpc.NewHexNumber(0)
	}
	if tx.GasLimit == nil {
		tx.GasLimit = rpc.NewHexNumber(0)
	}
	if tx.GasPrice == nil {
		tx.GasPrice = rpc.NewHexNumber(defaultGasPrice)
	}

	if contractCreation {
		tx.tx = types.NewContractCreation(tx.Nonce.Uint64(), tx.Value.BigInt(), tx.GasLimit.BigInt(), tx.GasPrice.BigInt(), data)
	} else {
		if tx.To == nil {
			return fmt.Errorf("need to address")
		}
		tx.tx = types.NewTransaction(tx.Nonce.Uint64(), *tx.To, tx.Value.BigInt(), tx.GasLimit.BigInt(), tx.GasPrice.BigInt(), data)
	}

	return nil
}

type SignTransactionResult struct {
	Raw string `json:"raw"`
	Tx  *Tx    `json:"tx"`
}

func newTx(t *types.Transaction) *Tx {
	from, _ := t.From()
	return &Tx{
		tx:       t,
		To:       t.To(),
		From:     from,
		Value:    rpc.NewHexNumber(t.Value()),
		Nonce:    rpc.NewHexNumber(t.Nonce()),
		Data:     "0x" + common.Bytes2Hex(t.Data()),
		GasLimit: rpc.NewHexNumber(t.Gas()),
		GasPrice: rpc.NewHexNumber(t.GasPrice()),
		Hash:     t.Hash(),
	}
}

// SignTransaction will sign the given transaction with the from account.
// The node needs to have the private key of the account corresponding with
// the given from address and it needs to be unlocked.
func (s *PublicTransactionPoolApi) SignTransaction(args *SignTransactionArgs) (*SignTransactionResult, error) {
	if args.Gas == nil {
		args.Gas = rpc.NewHexNumber(defaultGas)
	}
	if args.GasPrice == nil {
		args.GasPrice = rpc.NewHexNumber(defaultGasPrice)
	}
	if args.Value == nil {
		args.Value = rpc.NewHexNumber(0)
	}

	s.txMu.Lock()
	defer s.txMu.Unlock()

	if args.Nonce == nil {
		args.Nonce = rpc.NewHexNumber(s.txPool.State().GetNonce(args.From))
	}

	var tx *types.Transaction
	contractCreation := (args.To == common.Address{})

	if contractCreation {
		tx = types.NewContractCreation(args.Nonce.Uint64(), args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	} else {
		tx = types.NewTransaction(args.Nonce.Uint64(), args.To, args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	}

	signedTx, err := s.sign(args.From, tx)
	if err != nil {
		return nil, err
	}

	data, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		return nil, err
	}

	return &SignTransactionResult{"0x" + common.Bytes2Hex(data), newTx(tx)}, nil
}

// PendingTransactions returns the transactions that are in the transaction pool and have a from address that is one of
// the accounts this node manages.
func (s *PublicTransactionPoolApi) PendingTransactions() ([]*RPCTransaction, error) {
	accounts, err := s.am.Accounts()
	if err != nil {
		return nil, err
	}

	accountSet := set.New()
	for _, account := range accounts {
		accountSet.Add(account.Address)
	}

	pending := s.txPool.GetTransactions()
	transactions := make([]*RPCTransaction, 0)
	for _, tx := range pending {
		if from, _ := tx.From(); accountSet.Has(from) {
			transactions = append(transactions, newRPCPendingTransaction(tx))
		}
	}

	return transactions, nil
}

// NewPendingTransaction creates a subscription that is triggered each time a transaction enters the transaction pool
// and is send from one of the transactions this nodes manages.
func (s *PublicTransactionPoolApi) NewPendingTransactions() (rpc.Subscription, error) {
	sub := s.eventMux.Subscribe(TxPreEvent{})

	accounts, err := s.am.Accounts()
	if err != nil {
		return rpc.Subscription{}, err
	}
	accountSet := set.New()
	for _, account := range accounts {
		accountSet.Add(account.Address)
	}
	accountSetLastUpdates := time.Now()

	output := func(transaction interface{}) interface{} {
		if time.Since(accountSetLastUpdates) > (time.Duration(2) * time.Second) {
			if accounts, err = s.am.Accounts(); err != nil {
				accountSet.Clear()
				for _, account := range accounts {
					accountSet.Add(account.Address)
				}
				accountSetLastUpdates = time.Now()
			}
		}

		tx := transaction.(TxPreEvent)
		if from, err := tx.Tx.From(); err == nil {
			if accountSet.Has(from) {
				return tx.Tx.Hash()
			}
		}
		return nil
	}

	return rpc.NewSubscriptionWithOutputFormat(sub, output), nil
}

// Resend accepts an existing transaction and a new gas price and limit. It will remove the given transaction from the
// pool and reinsert it with the new gas price and limit.
func (s *PublicTransactionPoolApi) Resend(tx *Tx, gasPrice, gasLimit *rpc.HexNumber) (common.Hash, error) {

	pending := s.txPool.GetTransactions()
	for _, p := range pending {
		if pFrom, err := p.From(); err == nil && pFrom == tx.From && p.SigHash() == tx.tx.SigHash() {
			if gasPrice == nil {
				gasPrice = rpc.NewHexNumber(tx.tx.GasPrice())
			}
			if gasLimit == nil {
				gasLimit = rpc.NewHexNumber(tx.tx.Gas())
			}

			var newTx *types.Transaction
			contractCreation := (*tx.tx.To() == common.Address{})
			if contractCreation {
				newTx = types.NewContractCreation(tx.tx.Nonce(), tx.tx.Value(), gasPrice.BigInt(), gasLimit.BigInt(), tx.tx.Data())
			} else {
				newTx = types.NewTransaction(tx.tx.Nonce(), *tx.tx.To(), tx.tx.Value(), gasPrice.BigInt(), gasLimit.BigInt(), tx.tx.Data())
			}

			signedTx, err := s.sign(tx.From, newTx)
			if err != nil {
				return common.Hash{}, err
			}

			s.txPool.RemoveTx(tx.Hash)
			if err = s.txPool.Add(signedTx); err != nil {
				return common.Hash{}, err
			}

			return signedTx.Hash(), nil
		}
	}

	return common.Hash{}, fmt.Errorf("Transaction %#x not found", tx.Hash)
}
