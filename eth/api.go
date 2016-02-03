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

package eth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"sync"
	"time"

	"gopkg.in/fatih/set.v0"

	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	defaultGasPrice = uint64(10000000000000)
	defaultGas      = uint64(90000)
)

// blockByNumber is a commonly used helper function which retrieves and returns the block for the given block number. It
// returns nil when no block could be found.
func blockByNumber(m *miner.Miner, bc *core.BlockChain, blockNr rpc.BlockNumber) *types.Block {
	if blockNr == rpc.PendingBlockNumber {
		return m.PendingBlock()
	}
	if blockNr == rpc.LatestBlockNumber {
		return bc.CurrentBlock()
	}

	return bc.GetBlockByNumber(uint64(blockNr))
}

// PublicEthereumAPI provides an API to access Ethereum related information.
// It offers only methods that operate on public data that is freely available to anyone.
type PublicEthereumAPI struct {
	e   *Ethereum
	gpo *GasPriceOracle
}

// NewPublicEthereumAPI creates a new Etheruem protocol API.
func NewPublicEthereumAPI(e *Ethereum) *PublicEthereumAPI {
	return &PublicEthereumAPI{e, NewGasPriceOracle(e)}
}

// GasPrice returns a suggestion for a gas price.
func (s *PublicEthereumAPI) GasPrice() *big.Int {
	return s.gpo.SuggestPrice()
}

// GetCompilers returns the collection of available smart contract compilers
func (s *PublicEthereumAPI) GetCompilers() ([]string, error) {
	solc, err := s.e.Solc()
	if err != nil {
		return nil, err
	}

	if solc != nil {
		return []string{"Solidity"}, nil
	}

	return []string{}, nil
}

// CompileSolidity compiles the given solidity source
func (s *PublicEthereumAPI) CompileSolidity(source string) (map[string]*compiler.Contract, error) {
	solc, err := s.e.Solc()
	if err != nil {
		return nil, err
	}

	if solc == nil {
		return nil, errors.New("solc (solidity compiler) not found")
	}

	return solc.Compile(source)
}

// Etherbase is the address that mining rewards will be send to
func (s *PublicEthereumAPI) Etherbase() (common.Address, error) {
	return s.e.Etherbase()
}

// see Etherbase
func (s *PublicEthereumAPI) Coinbase() (common.Address, error) {
	return s.Etherbase()
}

// ProtocolVersion returns the current Ethereum protocol version this node supports
func (s *PublicEthereumAPI) ProtocolVersion() *rpc.HexNumber {
	return rpc.NewHexNumber(s.e.EthVersion())
}

// Hashrate returns the POW hashrate
func (s *PublicEthereumAPI) Hashrate() *rpc.HexNumber {
	return rpc.NewHexNumber(s.e.Miner().HashRate())
}

// Syncing returns false in case the node is currently not synching with the network. It can be up to date or has not
// yet received the latest block headers from its pears. In case it is synchronizing an object with 3 properties is
// returned:
// - startingBlock: block number this node started to synchronise from
// - currentBlock: block number this node is currently importing
// - highestBlock: block number of the highest block header this node has received from peers
func (s *PublicEthereumAPI) Syncing() (interface{}, error) {
	origin, current, height := s.e.Downloader().Progress()
	if current < height {
		return map[string]interface{}{
			"startingBlock": rpc.NewHexNumber(origin),
			"currentBlock":  rpc.NewHexNumber(current),
			"highestBlock":  rpc.NewHexNumber(height),
		}, nil
	}
	return false, nil
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

// Start the miner with the given number of threads
func (s *PrivateMinerAPI) Start(threads rpc.HexNumber) (bool, error) {
	s.e.StartAutoDAG()
	err := s.e.StartMining(threads.Int(), "")
	if err == nil {
		return true, nil
	}
	return false, err
}

// Stop the miner
func (s *PrivateMinerAPI) Stop() bool {
	s.e.StopMining()
	return true
}

// SetExtra sets the extra data string that is included when this miner mines a block.
func (s *PrivateMinerAPI) SetExtra(extra string) (bool, error) {
	if err := s.e.Miner().SetExtra([]byte(extra)); err != nil {
		return false, err
	}
	return true, nil
}

// SetGasPrice sets the minimum accepted gas price for the miner.
func (s *PrivateMinerAPI) SetGasPrice(gasPrice rpc.Number) bool {
	s.e.Miner().SetGasPrice(gasPrice.BigInt())
	return true
}

// SetEtherbase sets the etherbase of the miner
func (s *PrivateMinerAPI) SetEtherbase(etherbase common.Address) bool {
	s.e.SetEtherbase(etherbase)
	return true
}

// StartAutoDAG starts auto DAG generation. This will prevent the DAG generating on epoch change
// which will cause the node to stop mining during the generation process.
func (s *PrivateMinerAPI) StartAutoDAG() bool {
	s.e.StartAutoDAG()
	return true
}

// StopAutoDAG stops auto DAG generation
func (s *PrivateMinerAPI) StopAutoDAG() bool {
	s.e.StopAutoDAG()
	return true
}

// MakeDAG creates the new DAG for the given block number
func (s *PrivateMinerAPI) MakeDAG(blockNr rpc.BlockNumber) (bool, error) {
	if err := ethash.MakeDAG(uint64(blockNr.Int64()), ""); err != nil {
		return false, err
	}
	return true, nil
}

// PublicTxPoolAPI offers and API for the transaction pool. It only operates on data that is non confidential.
type PublicTxPoolAPI struct {
	e *Ethereum
}

// NewPublicTxPoolAPI creates a new tx pool service that gives information about the transaction pool.
func NewPublicTxPoolAPI(e *Ethereum) *PublicTxPoolAPI {
	return &PublicTxPoolAPI{e}
}

// Content returns the transactions contained within the transaction pool.
func (s *PublicTxPoolAPI) Content() map[string]map[string]map[string][]*RPCTransaction {
	content := map[string]map[string]map[string][]*RPCTransaction{
		"pending": make(map[string]map[string][]*RPCTransaction),
		"queued":  make(map[string]map[string][]*RPCTransaction),
	}
	pending, queue := s.e.TxPool().Content()

	// Flatten the pending transactions
	for account, batches := range pending {
		dump := make(map[string][]*RPCTransaction)
		for nonce, txs := range batches {
			nonce := fmt.Sprintf("%d", nonce)
			for _, tx := range txs {
				dump[nonce] = append(dump[nonce], newRPCPendingTransaction(tx))
			}
		}
		content["pending"][account.Hex()] = dump
	}
	// Flatten the queued transactions
	for account, batches := range queue {
		dump := make(map[string][]*RPCTransaction)
		for nonce, txs := range batches {
			nonce := fmt.Sprintf("%d", nonce)
			for _, tx := range txs {
				dump[nonce] = append(dump[nonce], newRPCPendingTransaction(tx))
			}
		}
		content["queued"][account.Hex()] = dump
	}
	return content
}

// Status returns the number of pending and queued transaction in the pool.
func (s *PublicTxPoolAPI) Status() map[string]*rpc.HexNumber {
	pending, queue := s.e.TxPool().Stats()
	return map[string]*rpc.HexNumber{
		"pending": rpc.NewHexNumber(pending),
		"queued":  rpc.NewHexNumber(queue),
	}
}

// Inspect retrieves the content of the transaction pool and flattens it into an
// easily inspectable list.
func (s *PublicTxPoolAPI) Inspect() map[string]map[string]map[string][]string {
	content := map[string]map[string]map[string][]string{
		"pending": make(map[string]map[string][]string),
		"queued":  make(map[string]map[string][]string),
	}
	pending, queue := s.e.TxPool().Content()

	// Define a formatter to flatten a transaction into a string
	var format = func(tx *types.Transaction) string {
		if to := tx.To(); to != nil {
			return fmt.Sprintf("%s: %v wei + %v × %v gas", tx.To().Hex(), tx.Value(), tx.Gas(), tx.GasPrice())
		}
		return fmt.Sprintf("contract creation: %v wei + %v × %v gas", tx.Value(), tx.Gas(), tx.GasPrice())
	}
	// Flatten the pending transactions
	for account, batches := range pending {
		dump := make(map[string][]string)
		for nonce, txs := range batches {
			nonce := fmt.Sprintf("%d", nonce)
			for _, tx := range txs {
				dump[nonce] = append(dump[nonce], format(tx))
			}
		}
		content["pending"][account.Hex()] = dump
	}
	// Flatten the queued transactions
	for account, batches := range queue {
		dump := make(map[string][]string)
		for nonce, txs := range batches {
			nonce := fmt.Sprintf("%d", nonce)
			for _, tx := range txs {
				dump[nonce] = append(dump[nonce], format(tx))
			}
		}
		content["queued"][account.Hex()] = dump
	}
	return content
}

// PublicAccountAPI provides an API to access accounts managed by this node.
// It offers only methods that can retrieve accounts.
type PublicAccountAPI struct {
	am *accounts.Manager
}

// NewPublicAccountAPI creates a new PublicAccountAPI.
func NewPublicAccountAPI(am *accounts.Manager) *PublicAccountAPI {
	return &PublicAccountAPI{am: am}
}

// Accounts returns the collection of accounts this node manages
func (s *PublicAccountAPI) Accounts() ([]accounts.Account, error) {
	return s.am.Accounts()
}

// PrivateAccountAPI provides an API to access accounts managed by this node.
// It offers methods to create, (un)lock en list accounts.
type PrivateAccountAPI struct {
	am *accounts.Manager
}

// NewPrivateAccountAPI create a new PrivateAccountAPI.
func NewPrivateAccountAPI(am *accounts.Manager) *PrivateAccountAPI {
	return &PrivateAccountAPI{am}
}

// ListAccounts will return a list of addresses for accounts this node manages.
func (s *PrivateAccountAPI) ListAccounts() ([]common.Address, error) {
	accounts, err := s.am.Accounts()
	if err != nil {
		return nil, err
	}

	addresses := make([]common.Address, len(accounts))
	for i, acc := range accounts {
		addresses[i] = acc.Address
	}
	return addresses, nil
}

// NewAccount will create a new account and returns the address for the new account.
func (s *PrivateAccountAPI) NewAccount(password string) (common.Address, error) {
	acc, err := s.am.NewAccount(password)
	if err == nil {
		return acc.Address, nil
	}
	return common.Address{}, err
}

// UnlockAccount will unlock the account associated with the given address with the given password for duration seconds.
// It returns an indication if the action was successful.
func (s *PrivateAccountAPI) UnlockAccount(addr common.Address, password string, duration int) bool {
	if err := s.am.TimedUnlock(addr, password, time.Duration(duration)*time.Second); err != nil {
		glog.V(logger.Info).Infof("%v\n", err)
		return false
	}
	return true
}

// LockAccount will lock the account associated with the given address when it's unlocked.
func (s *PrivateAccountAPI) LockAccount(addr common.Address) bool {
	return s.am.Lock(addr) == nil
}

// PublicBlockChainAPI provides an API to access the Ethereum blockchain.
// It offers only methods that operate on public data that is freely available to anyone.
type PublicBlockChainAPI struct {
	bc       *core.BlockChain
	chainDb  ethdb.Database
	eventMux *event.TypeMux
	am       *accounts.Manager
	miner    *miner.Miner
}

// NewPublicBlockChainAPI creates a new Etheruem blockchain API.
func NewPublicBlockChainAPI(bc *core.BlockChain, m *miner.Miner, chainDb ethdb.Database, eventMux *event.TypeMux, am *accounts.Manager) *PublicBlockChainAPI {
	return &PublicBlockChainAPI{bc: bc, miner: m, chainDb: chainDb, eventMux: eventMux, am: am}
}

// BlockNumber returns the block number of the chain head.
func (s *PublicBlockChainAPI) BlockNumber() *big.Int {
	return s.bc.CurrentHeader().Number
}

// GetBalance returns the amount of wei for the given address in the state of the given block number.
// When block number equals rpc.LatestBlockNumber the current block is used.
func (s *PublicBlockChainAPI) GetBalance(address common.Address, blockNr rpc.BlockNumber) (*big.Int, error) {
	block := blockByNumber(s.miner, s.bc, blockNr)
	if block == nil {
		return nil, nil
	}

	state, err := state.New(block.Root(), s.chainDb)
	if err != nil {
		return nil, err
	}
	return state.GetBalance(address), nil
}

// GetBlockByNumber returns the requested block. When blockNr is -1 the chain head is returned. When fullTx is true all
// transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetBlockByNumber(blockNr rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	if block := blockByNumber(s.miner, s.bc, blockNr); block != nil {
		return s.rpcOutputBlock(block, true, fullTx)
	}
	return nil, nil
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetBlockByHash(blockHash common.Hash, fullTx bool) (map[string]interface{}, error) {
	if block := s.bc.GetBlock(blockHash); block != nil {
		return s.rpcOutputBlock(block, true, fullTx)
	}
	return nil, nil
}

// GetUncleByBlockNumberAndIndex returns the uncle block for the given block hash and index. When fullTx is true
// all transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetUncleByBlockNumberAndIndex(blockNr rpc.BlockNumber, index rpc.HexNumber) (map[string]interface{}, error) {
	if blockNr == rpc.PendingBlockNumber {
		return nil, nil
	}

	if block := blockByNumber(s.miner, s.bc, blockNr); block != nil {
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
func (s *PublicBlockChainAPI) GetUncleByBlockHashAndIndex(blockHash common.Hash, index rpc.HexNumber) (map[string]interface{}, error) {
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
func (s *PublicBlockChainAPI) GetUncleCountByBlockNumber(blockNr rpc.BlockNumber) *rpc.HexNumber {
	if blockNr == rpc.PendingBlockNumber {
		return rpc.NewHexNumber(0)
	}

	if block := blockByNumber(s.miner, s.bc, blockNr); block != nil {
		return rpc.NewHexNumber(len(block.Uncles()))
	}
	return nil
}

// GetUncleCountByBlockHash returns number of uncles in the block for the given block hash
func (s *PublicBlockChainAPI) GetUncleCountByBlockHash(blockHash common.Hash) *rpc.HexNumber {
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
func (s *PublicBlockChainAPI) NewBlocks(args NewBlocksArgs) (rpc.Subscription, error) {
	sub := s.eventMux.Subscribe(core.ChainEvent{})

	output := func(rawBlock interface{}) interface{} {
		if event, ok := rawBlock.(core.ChainEvent); ok {
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
func (s *PublicBlockChainAPI) GetCode(address common.Address, blockNr rpc.BlockNumber) (string, error) {
	return s.GetData(address, blockNr)
}

// GetData returns the data stored at the given address in the state for the given block number.
func (s *PublicBlockChainAPI) GetData(address common.Address, blockNr rpc.BlockNumber) (string, error) {
	if block := blockByNumber(s.miner, s.bc, blockNr); block != nil {
		state, err := state.New(block.Root(), s.chainDb)
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
func (s *PublicBlockChainAPI) GetStorageAt(address common.Address, key string, blockNr rpc.BlockNumber) (string, error) {
	if block := blockByNumber(s.miner, s.bc, blockNr); block != nil {
		state, err := state.New(block.Root(), s.chainDb)
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

func (s *PublicBlockChainAPI) doCall(args CallArgs, blockNr rpc.BlockNumber) (string, *big.Int, error) {
	if block := blockByNumber(s.miner, s.bc, blockNr); block != nil {
		stateDb, err := state.New(block.Root(), s.chainDb)
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
		vmenv := core.NewEnv(stateDb, s.bc, msg, header)
		gp := new(core.GasPool).AddGas(common.MaxBig)
		res, gas, err := core.ApplyMessage(vmenv, msg, gp)
		if len(res) == 0 { // backwards compatability
			return "0x", gas, err
		}
		return common.ToHex(res), gas, err
	}

	return "0x", common.Big0, nil
}

// Call executes the given transaction on the state for the given block number.
// It doesn't make and changes in the state/blockchain and is usefull to execute and retrieve values.
func (s *PublicBlockChainAPI) Call(args CallArgs, blockNr rpc.BlockNumber) (string, error) {
	result, _, err := s.doCall(args, blockNr)
	return result, err
}

// EstimateGas returns an estimate of the amount of gas needed to execute the given transaction.
func (s *PublicBlockChainAPI) EstimateGas(args CallArgs) (*rpc.HexNumber, error) {
	_, gas, err := s.doCall(args, rpc.LatestBlockNumber)
	return rpc.NewHexNumber(gas), err
}

// rpcOutputBlock converts the given block to the RPC output which depends on fullTx. If inclTx is true transactions are
// returned. When fullTx is true the returned block contains full transaction details, otherwise it will only contain
// transaction hashes.
func (s *PublicBlockChainAPI) rpcOutputBlock(b *types.Block, inclTx bool, fullTx bool) (map[string]interface{}, error) {
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

// PublicTransactionPoolAPI exposes methods for the RPC interface
type PublicTransactionPoolAPI struct {
	eventMux *event.TypeMux
	chainDb  ethdb.Database
	bc       *core.BlockChain
	miner    *miner.Miner
	am       *accounts.Manager
	txPool   *core.TxPool
	txMu     sync.Mutex
}

// NewPublicTransactionPoolAPI creates a new RPC service with methods specific for the transaction pool.
func NewPublicTransactionPoolAPI(txPool *core.TxPool, m *miner.Miner, chainDb ethdb.Database, eventMux *event.TypeMux, bc *core.BlockChain, am *accounts.Manager) *PublicTransactionPoolAPI {
	return &PublicTransactionPoolAPI{
		eventMux: eventMux,
		chainDb:  chainDb,
		bc:       bc,
		am:       am,
		txPool:   txPool,
		miner:    m,
	}
}

func getTransaction(chainDb ethdb.Database, txPool *core.TxPool, txHash common.Hash) (*types.Transaction, bool, error) {
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
func (s *PublicTransactionPoolAPI) GetBlockTransactionCountByNumber(blockNr rpc.BlockNumber) *rpc.HexNumber {
	if blockNr == rpc.PendingBlockNumber {
		return rpc.NewHexNumber(0)
	}

	if block := blockByNumber(s.miner, s.bc, blockNr); block != nil {
		return rpc.NewHexNumber(len(block.Transactions()))
	}

	return nil
}

// GetBlockTransactionCountByHash returns the number of transactions in the block with the given hash.
func (s *PublicTransactionPoolAPI) GetBlockTransactionCountByHash(blockHash common.Hash) *rpc.HexNumber {
	if block := s.bc.GetBlock(blockHash); block != nil {
		return rpc.NewHexNumber(len(block.Transactions()))
	}
	return nil
}

// GetTransactionByBlockNumberAndIndex returns the transaction for the given block number and index.
func (s *PublicTransactionPoolAPI) GetTransactionByBlockNumberAndIndex(blockNr rpc.BlockNumber, index rpc.HexNumber) (*RPCTransaction, error) {
	if block := blockByNumber(s.miner, s.bc, blockNr); block != nil {
		return newRPCTransactionFromBlockIndex(block, index.Int())
	}
	return nil, nil
}

// GetTransactionByBlockHashAndIndex returns the transaction for the given block hash and index.
func (s *PublicTransactionPoolAPI) GetTransactionByBlockHashAndIndex(blockHash common.Hash, index rpc.HexNumber) (*RPCTransaction, error) {
	if block := s.bc.GetBlock(blockHash); block != nil {
		return newRPCTransactionFromBlockIndex(block, index.Int())
	}
	return nil, nil
}

// GetTransactionCount returns the number of transactions the given address has sent for the given block number
func (s *PublicTransactionPoolAPI) GetTransactionCount(address common.Address, blockNr rpc.BlockNumber) (*rpc.HexNumber, error) {
	block := blockByNumber(s.miner, s.bc, blockNr)
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
func (s *PublicTransactionPoolAPI) GetTransactionByHash(txHash common.Hash) (*RPCTransaction, error) {
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
func (s *PublicTransactionPoolAPI) GetTransactionReceipt(txHash common.Hash) (map[string]interface{}, error) {
	receipt := core.GetReceipt(s.chainDb, txHash)
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
func (s *PublicTransactionPoolAPI) sign(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
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
func (s *PublicTransactionPoolAPI) SendTransaction(args SendTxArgs) (common.Hash, error) {
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

	s.txPool.SetLocal(signedTx)
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
func (s *PublicTransactionPoolAPI) SendRawTransaction(encodedTx string) (string, error) {
	tx := new(types.Transaction)
	if err := rlp.DecodeBytes(common.FromHex(encodedTx), tx); err != nil {
		return "", err
	}

	s.txPool.SetLocal(tx)
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
func (s *PublicTransactionPoolAPI) Sign(address common.Address, data string) (string, error) {
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
func (s *PublicTransactionPoolAPI) SignTransaction(args *SignTransactionArgs) (*SignTransactionResult, error) {
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
func (s *PublicTransactionPoolAPI) PendingTransactions() ([]*RPCTransaction, error) {
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
func (s *PublicTransactionPoolAPI) NewPendingTransactions() (rpc.Subscription, error) {
	sub := s.eventMux.Subscribe(core.TxPreEvent{})

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

		tx := transaction.(core.TxPreEvent)
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
func (s *PublicTransactionPoolAPI) Resend(tx *Tx, gasPrice, gasLimit *rpc.HexNumber) (common.Hash, error) {

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

// PrivateAdminAPI is the collection of Etheruem APIs exposed over the private
// admin endpoint.
type PrivateAdminAPI struct {
	eth *Ethereum
}

// NewPrivateAdminAPI creates a new API definition for the private admin methods
// of the Ethereum service.
func NewPrivateAdminAPI(eth *Ethereum) *PrivateAdminAPI {
	return &PrivateAdminAPI{eth: eth}
}

// SetSolc sets the Solidity compiler path to be used by the node.
func (api *PrivateAdminAPI) SetSolc(path string) (string, error) {
	solc, err := api.eth.SetSolc(path)
	if err != nil {
		return "", err
	}
	return solc.Info(), nil
}

// ExportChain exports the current blockchain into a local file.
func (api *PrivateAdminAPI) ExportChain(file string) (bool, error) {
	// Make sure we can create the file to export into
	out, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return false, err
	}
	defer out.Close()

	// Export the blockchain
	if err := api.eth.BlockChain().Export(out); err != nil {
		return false, err
	}
	return true, nil
}

func hasAllBlocks(chain *core.BlockChain, bs []*types.Block) bool {
	for _, b := range bs {
		if !chain.HasBlock(b.Hash()) {
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

	// Run actual the import in pre-configured batches
	stream := rlp.NewStream(in, 0)

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

// PublicDebugAPI is the collection of Etheruem APIs exposed over the public
// debugging endpoint.
type PublicDebugAPI struct {
	eth *Ethereum
}

// NewPublicDebugAPI creates a new API definition for the public debug methods
// of the Ethereum service.
func NewPublicDebugAPI(eth *Ethereum) *PublicDebugAPI {
	return &PublicDebugAPI{eth: eth}
}

// DumpBlock retrieves the entire state of the database at a given block.
func (api *PublicDebugAPI) DumpBlock(number uint64) (state.World, error) {
	block := api.eth.BlockChain().GetBlockByNumber(number)
	if block == nil {
		return state.World{}, fmt.Errorf("block #%d not found", number)
	}
	stateDb, err := state.New(block.Root(), api.eth.ChainDb())
	if err != nil {
		return state.World{}, err
	}
	return stateDb.RawDump(), nil
}

// GetBlockRlp retrieves the RLP encoded for of a single block.
func (api *PublicDebugAPI) GetBlockRlp(number uint64) (string, error) {
	block := api.eth.BlockChain().GetBlockByNumber(number)
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	encoded, err := rlp.EncodeToBytes(block)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", encoded), nil
}

// PrintBlock retrieves a block and returns its pretty printed form.
func (api *PublicDebugAPI) PrintBlock(number uint64) (string, error) {
	block := api.eth.BlockChain().GetBlockByNumber(number)
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	return fmt.Sprintf("%s", block), nil
}

// SeedHash retrieves the seed hash of a block.
func (api *PublicDebugAPI) SeedHash(number uint64) (string, error) {
	block := api.eth.BlockChain().GetBlockByNumber(number)
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	hash, err := ethash.GetSeedHash(number)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("0x%x", hash), nil
}

// PrivateDebugAPI is the collection of Etheruem APIs exposed over the private
// debugging endpoint.
type PrivateDebugAPI struct {
	eth *Ethereum
}

// NewPrivateDebugAPI creates a new API definition for the private debug methods
// of the Ethereum service.
func NewPrivateDebugAPI(eth *Ethereum) *PrivateDebugAPI {
	return &PrivateDebugAPI{eth: eth}
}

// ProcessBlock reprocesses an already owned block.
func (api *PrivateDebugAPI) ProcessBlock(number uint64) (bool, error) {
	// Fetch the block that we aim to reprocess
	block := api.eth.BlockChain().GetBlockByNumber(number)
	if block == nil {
		return false, fmt.Errorf("block #%d not found", number)
	}
	// Temporarily enable debugging
	defer func(old bool) { vm.Debug = old }(vm.Debug)
	vm.Debug = true

	// Validate and reprocess the block
	var (
		blockchain = api.eth.BlockChain()
		validator  = blockchain.Validator()
		processor  = blockchain.Processor()
	)
	if err := core.ValidateHeader(blockchain.AuxValidator(), block.Header(), blockchain.GetHeader(block.ParentHash()), true, false); err != nil {
		return false, err
	}
	statedb, err := state.New(blockchain.GetBlock(block.ParentHash()).Root(), api.eth.ChainDb())
	if err != nil {
		return false, err
	}
	receipts, _, usedGas, err := processor.Process(block, statedb)
	if err != nil {
		return false, err
	}
	if err := validator.ValidateState(block, blockchain.GetBlock(block.ParentHash()), statedb, receipts, usedGas); err != nil {
		return false, err
	}
	return true, nil
}

// SetHead rewinds the head of the blockchain to a previous block.
func (api *PrivateDebugAPI) SetHead(number uint64) {
	api.eth.BlockChain().SetHead(number)
}

// StructLogRes stores a structured log emitted by the evm while replaying a
// transaction in debug mode
type structLogRes struct {
	Pc      uint64            `json:"pc"`
	Op      string            `json:"op"`
	Gas     *big.Int          `json:"gas"`
	GasCost *big.Int          `json:"gasCost"`
	Error   error             `json:"error"`
	Stack   []string          `json:"stack"`
	Memory  map[string]string `json:"memory"`
	Storage map[string]string `json:"storage"`
}

// TransactionExecutionRes groups all structured logs emitted by the evm
// while replaying a transaction in debug mode as well as the amount of
// gas used and the return value
type TransactionExecutionResult struct {
	Gas         *big.Int       `json:"gas"`
	ReturnValue string         `json:"returnValue"`
	StructLogs  []structLogRes `json:"structLogs"`
}

func (s *PrivateDebugAPI) doReplayTransaction(txHash common.Hash) ([]vm.StructLog, []byte, *big.Int, error) {
	// Retrieve the tx from the chain
	tx, _, blockIndex, _ := core.GetTransaction(s.eth.ChainDb(), txHash)

	if tx == nil {
		return nil, nil, nil, fmt.Errorf("Transaction not found")
	}

	block := s.eth.BlockChain().GetBlockByNumber(blockIndex - 1)
	if block == nil {
		return nil, nil, nil, fmt.Errorf("Unable to retrieve prior block")
	}

	// Create the state database
	stateDb, err := state.New(block.Root(), s.eth.ChainDb())
	if err != nil {
		return nil, nil, nil, err
	}

	txFrom, err := tx.From()

	if err != nil {
		return nil, nil, nil, fmt.Errorf("Unable to create transaction sender")
	}
	from := stateDb.GetOrNewStateObject(txFrom)
	msg := callmsg{
		from:     from,
		to:       tx.To(),
		gas:      tx.Gas(),
		gasPrice: tx.GasPrice(),
		value:    tx.Value(),
		data:     tx.Data(),
	}

	vmenv := core.NewEnv(stateDb, s.eth.BlockChain(), msg, block.Header())
	gp := new(core.GasPool).AddGas(block.GasLimit())
	vm.GenerateStructLogs = true
	defer func() { vm.GenerateStructLogs = false }()

	ret, gas, err := core.ApplyMessage(vmenv, msg, gp)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Error executing transaction %v", err)
	}

	return vmenv.StructLogs(), ret, gas, nil
}

// Executes a transaction and returns the structured logs of the evm
// gathered during the execution
func (s *PrivateDebugAPI) ReplayTransaction(txHash common.Hash, stackDepth int, memorySize int, storageSize int) (*TransactionExecutionResult, error) {

	structLogs, ret, gas, err := s.doReplayTransaction(txHash)

	if err != nil {
		return nil, err
	}

	res := TransactionExecutionResult{
		Gas:         gas,
		ReturnValue: fmt.Sprintf("%x", ret),
		StructLogs:  make([]structLogRes, len(structLogs)),
	}

	for index, trace := range structLogs {

		stackLength := len(trace.Stack)

		// Return full stack by default
		if stackDepth != -1 && stackDepth < stackLength {
			stackLength = stackDepth
		}

		res.StructLogs[index] = structLogRes{
			Pc:      trace.Pc,
			Op:      trace.Op.String(),
			Gas:     trace.Gas,
			GasCost: trace.GasCost,
			Error:   trace.Err,
			Stack:   make([]string, stackLength),
			Memory:  make(map[string]string),
			Storage: make(map[string]string),
		}

		for i := 0; i < stackLength; i++ {
			res.StructLogs[index].Stack[i] = fmt.Sprintf("%x", common.LeftPadBytes(trace.Stack[i].Bytes(), 32))
		}

		addr := 0
		memorySizeLocal := memorySize

		// Return full memory by default
		if memorySize == -1 {
			memorySizeLocal = len(trace.Memory)
		}

		for i := 0; i+16 <= len(trace.Memory) && addr < memorySizeLocal; i += 16 {
			res.StructLogs[index].Memory[fmt.Sprintf("%04d", addr*16)] = fmt.Sprintf("%x", trace.Memory[i:i+16])
			addr++
		}

		storageLength := len(trace.Stack)
		if storageSize != -1 && storageSize < storageLength {
			storageLength = storageSize
		}

		i := 0
		for storageIndex, storageValue := range trace.Storage {
			if i >= storageLength {
				break
			}
			res.StructLogs[index].Storage[fmt.Sprintf("%x", storageIndex)] = fmt.Sprintf("%x", storageValue)
			i++
		}
	}
	return &res, nil
}

// PublicNetAPI offers network related RPC methods
type PublicNetAPI struct {
	net            *p2p.Server
	networkVersion int
}

// NewPublicNetAPI creates a new net api instance.
func NewPublicNetAPI(net *p2p.Server, networkVersion int) *PublicNetAPI {
	return &PublicNetAPI{net, networkVersion}
}

// Listening returns an indication if the node is listening for network connections.
func (s *PublicNetAPI) Listening() bool {
	return true // always listening
}

// Peercount returns the number of connected peers
func (s *PublicNetAPI) PeerCount() *rpc.HexNumber {
	return rpc.NewHexNumber(s.net.PeerCount())
}

// ProtocolVersion returns the current ethereum protocol version.
func (s *PublicNetAPI) Version() string {
	return fmt.Sprintf("%d", s.networkVersion)
}
