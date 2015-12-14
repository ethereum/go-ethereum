// Copyright 2014 The go-ethereum Authors
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

// Package xeth is the interface to all Ethereum functionality.
package xeth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/whisper"
)

var (
	filterTickerTime = 5 * time.Minute
	defaultGasPrice  = big.NewInt(10000000000000) //150000000000
	defaultGas       = big.NewInt(90000)          //500000
	dappStorePre     = []byte("dapp-")
	addrReg          = regexp.MustCompile(`^(0x)?[a-fA-F0-9]{40}$`)
)

// byte will be inferred
const (
	UnknownFilterTy = iota
	BlockFilterTy
	TransactionFilterTy
	LogFilterTy
)

type XEth struct {
	quit chan struct{}

	logMu    sync.RWMutex
	logQueue map[int]*logQueue

	blockMu    sync.RWMutex
	blockQueue map[int]*hashQueue

	transactionMu    sync.RWMutex
	transactionQueue map[int]*hashQueue

	messagesMu sync.RWMutex
	messages   map[int]*whisperFilter

	transactMu sync.Mutex

	// read-only fields
	backend       *node.Node
	frontend      Frontend
	agent         *miner.RemoteAgent
	gpo           *eth.GasPriceOracle
	state         *State
	whisper       *Whisper
	filterManager *filters.FilterSystem
}

func NewTest(stack *node.Node, frontend Frontend) *XEth {
	return &XEth{backend: stack, frontend: frontend}
}

// New creates an XEth that uses the given frontend.
// If a nil Frontend is provided, a default frontend which
// confirms all transactions will be used.
func New(stack *node.Node, frontend Frontend) *XEth {
	var (
		ethereum *eth.Ethereum
		whisper  *whisper.Whisper
	)
	stack.Service(&ethereum)
	stack.Service(&whisper)

	xeth := &XEth{
		backend:          stack,
		frontend:         frontend,
		quit:             make(chan struct{}),
		filterManager:    filters.NewFilterSystem(stack.EventMux()),
		logQueue:         make(map[int]*logQueue),
		blockQueue:       make(map[int]*hashQueue),
		transactionQueue: make(map[int]*hashQueue),
		messages:         make(map[int]*whisperFilter),
		agent:            miner.NewRemoteAgent(),
		gpo:              eth.NewGasPriceOracle(ethereum),
	}
	if whisper != nil {
		xeth.whisper = NewWhisper(whisper)
	}
	ethereum.Miner().Register(xeth.agent)
	if frontend == nil {
		xeth.frontend = dummyFrontend{}
	}
	state, _ := ethereum.BlockChain().State()
	xeth.state = NewState(xeth, state)
	go xeth.start()
	return xeth
}

func (self *XEth) EthereumService() *eth.Ethereum {
	var ethereum *eth.Ethereum
	if err := self.backend.Service(&ethereum); err != nil {
		return nil
	}
	return ethereum
}

func (self *XEth) WhisperService() *whisper.Whisper {
	var whisper *whisper.Whisper
	if err := self.backend.Service(&whisper); err != nil {
		return nil
	}
	return whisper
}

func (self *XEth) start() {
	timer := time.NewTicker(2 * time.Second)
	defer timer.Stop()
done:
	for {
		select {
		case <-timer.C:
			self.logMu.Lock()
			for id, filter := range self.logQueue {
				if time.Since(filter.timeout) > filterTickerTime {
					self.filterManager.Remove(id)
					delete(self.logQueue, id)
				}
			}
			self.logMu.Unlock()

			self.blockMu.Lock()
			for id, filter := range self.blockQueue {
				if time.Since(filter.timeout) > filterTickerTime {
					self.filterManager.Remove(id)
					delete(self.blockQueue, id)
				}
			}
			self.blockMu.Unlock()

			self.transactionMu.Lock()
			for id, filter := range self.transactionQueue {
				if time.Since(filter.timeout) > filterTickerTime {
					self.filterManager.Remove(id)
					delete(self.transactionQueue, id)
				}
			}
			self.transactionMu.Unlock()

			self.messagesMu.Lock()
			for id, filter := range self.messages {
				if time.Since(filter.activity()) > filterTickerTime {
					self.Whisper().Unwatch(id)
					delete(self.messages, id)
				}
			}
			self.messagesMu.Unlock()
		case <-self.quit:
			break done
		}
	}
}

// Stop releases any resources associated with self.
// It may not be called more than once.
func (self *XEth) Stop() {
	close(self.quit)
	self.filterManager.Stop()
	self.EthereumService().Miner().Unregister(self.agent)
}

func cAddress(a []string) []common.Address {
	bslice := make([]common.Address, len(a))
	for i, addr := range a {
		bslice[i] = common.HexToAddress(addr)
	}
	return bslice
}

func cTopics(t [][]string) [][]common.Hash {
	topics := make([][]common.Hash, len(t))
	for i, iv := range t {
		topics[i] = make([]common.Hash, len(iv))
		for j, jv := range iv {
			topics[i][j] = common.HexToHash(jv)
		}
	}
	return topics
}

func DefaultGas() *big.Int { return new(big.Int).Set(defaultGas) }

func (self *XEth) DefaultGasPrice() *big.Int {
	return self.gpo.SuggestPrice()
}

func (self *XEth) RemoteMining() *miner.RemoteAgent { return self.agent }

func (self *XEth) AtStateNum(num int64) *XEth {
	var st *state.StateDB
	var err error
	switch num {
	case -2:
		st = self.EthereumService().Miner().PendingState().Copy()
	default:
		if block := self.getBlockByHeight(num); block != nil {
			st, err = state.New(block.Root(), self.EthereumService().ChainDb())
			if err != nil {
				return nil
			}
		} else {
			st, err = state.New(self.EthereumService().BlockChain().GetBlockByNumber(0).Root(), self.EthereumService().ChainDb())
			if err != nil {
				return nil
			}
		}
	}
	return self.WithState(st)
}

func (self *XEth) WithState(statedb *state.StateDB) *XEth {
	xeth := &XEth{
		backend:  self.backend,
		frontend: self.frontend,
		gpo:      self.gpo,
	}

	xeth.state = NewState(xeth, statedb)
	return xeth
}

func (self *XEth) State() *State { return self.state }

// subscribes to new head block events and
// waits until blockchain height is greater n at any time
// given the current head, waits for the next chain event
// sets the state to the current head
// loop is async and quit by closing the channel
// used in tests and JS console debug module to control advancing private chain manually
// Note: this is not threadsafe, only called in JS single process and tests
func (self *XEth) UpdateState() (wait chan *big.Int) {
	wait = make(chan *big.Int)
	go func() {
		eventSub := self.backend.EventMux().Subscribe(core.ChainHeadEvent{})
		defer eventSub.Unsubscribe()

		var m, n *big.Int
		var ok bool

		eventCh := eventSub.Chan()
		for {
			select {
			case event, ok := <-eventCh:
				if !ok {
					// Event subscription closed, set the channel to nil to stop spinning
					eventCh = nil
					continue
				}
				// A real event arrived, process if new head block assignment
				if event, ok := event.Data.(core.ChainHeadEvent); ok {
					m = event.Block.Number()
					if n != nil && n.Cmp(m) < 0 {
						wait <- n
						n = nil
					}
					statedb, err := state.New(event.Block.Root(), self.EthereumService().ChainDb())
					if err != nil {
						glog.V(logger.Error).Infoln("Could not create new state: %v", err)
						return
					}
					self.state = NewState(self, statedb)
				}
			case n, ok = <-wait:
				if !ok {
					return
				}
			}
		}
	}()
	return
}

func (self *XEth) Whisper() *Whisper { return self.whisper }

func (self *XEth) getBlockByHeight(height int64) *types.Block {
	var num uint64

	switch height {
	case -2:
		return self.EthereumService().Miner().PendingBlock()
	case -1:
		return self.CurrentBlock()
	default:
		if height < 0 {
			return nil
		}

		num = uint64(height)
	}

	return self.EthereumService().BlockChain().GetBlockByNumber(num)
}

func (self *XEth) BlockByHash(strHash string) *Block {
	hash := common.HexToHash(strHash)
	block := self.EthereumService().BlockChain().GetBlock(hash)

	return NewBlock(block)
}

func (self *XEth) EthBlockByHash(strHash string) *types.Block {
	hash := common.HexToHash(strHash)
	block := self.EthereumService().BlockChain().GetBlock(hash)

	return block
}

func (self *XEth) EthTransactionByHash(hash string) (*types.Transaction, common.Hash, uint64, uint64) {
	ethereum := self.EthereumService()
	if tx, hash, number, index := core.GetTransaction(ethereum.ChainDb(), common.HexToHash(hash)); tx != nil {
		return tx, hash, number, index
	}
	return ethereum.TxPool().GetTransaction(common.HexToHash(hash)), common.Hash{}, 0, 0
}

func (self *XEth) BlockByNumber(num int64) *Block {
	return NewBlock(self.getBlockByHeight(num))
}

func (self *XEth) EthBlockByNumber(num int64) *types.Block {
	return self.getBlockByHeight(num)
}

func (self *XEth) Td(hash common.Hash) *big.Int {
	return self.EthereumService().BlockChain().GetTd(hash)
}

func (self *XEth) CurrentBlock() *types.Block {
	return self.EthereumService().BlockChain().CurrentBlock()
}

func (self *XEth) GetBlockReceipts(bhash common.Hash) types.Receipts {
	return core.GetBlockReceipts(self.EthereumService().ChainDb(), bhash)
}

func (self *XEth) GetTxReceipt(txhash common.Hash) *types.Receipt {
	return core.GetReceipt(self.EthereumService().ChainDb(), txhash)
}

func (self *XEth) GasLimit() *big.Int {
	return self.EthereumService().BlockChain().GasLimit()
}

func (self *XEth) Block(v interface{}) *Block {
	if n, ok := v.(int32); ok {
		return self.BlockByNumber(int64(n))
	} else if str, ok := v.(string); ok {
		return self.BlockByHash(str)
	} else if f, ok := v.(float64); ok { // JSON numbers are represented as float64
		return self.BlockByNumber(int64(f))
	}

	return nil
}

func (self *XEth) Accounts() []string {
	// TODO: check err?
	accounts, _ := self.EthereumService().AccountManager().Accounts()
	accountAddresses := make([]string, len(accounts))
	for i, ac := range accounts {
		accountAddresses[i] = ac.Address.Hex()
	}
	return accountAddresses
}

// accessor for solidity compiler.
// memoized if available, retried on-demand if not
func (self *XEth) Solc() (*compiler.Solidity, error) {
	return self.EthereumService().Solc()
}

// set in js console via admin interface or wrapper from cli flags
func (self *XEth) SetSolc(solcPath string) (*compiler.Solidity, error) {
	self.EthereumService().SetSolc(solcPath)
	return self.Solc()
}

// store DApp value in extra database
func (self *XEth) DbPut(key, val []byte) bool {
	self.EthereumService().DappDb().Put(append(dappStorePre, key...), val)
	return true
}

// retrieve DApp value from extra database
func (self *XEth) DbGet(key []byte) ([]byte, error) {
	val, err := self.EthereumService().DappDb().Get(append(dappStorePre, key...))
	return val, err
}

func (self *XEth) PeerCount() int {
	return self.backend.Server().PeerCount()
}

func (self *XEth) IsMining() bool {
	return self.EthereumService().IsMining()
}

func (self *XEth) HashRate() int64 {
	return self.EthereumService().Miner().HashRate()
}

func (self *XEth) EthVersion() string {
	return fmt.Sprintf("%d", self.EthereumService().EthVersion())
}

func (self *XEth) NetworkVersion() string {
	return fmt.Sprintf("%d", self.EthereumService().NetVersion())
}

func (self *XEth) WhisperVersion() string {
	return fmt.Sprintf("%d", self.WhisperService().Version())
}

func (self *XEth) ClientVersion() string {
	return self.backend.Server().Name
}

func (self *XEth) SetMining(shouldmine bool, threads int) bool {
	ismining := self.EthereumService().IsMining()
	if shouldmine && !ismining {
		err := self.EthereumService().StartMining(threads, "")
		return err == nil
	}
	if ismining && !shouldmine {
		self.EthereumService().StopMining()
	}
	return self.EthereumService().IsMining()
}

func (self *XEth) IsListening() bool {
	return true
}

func (self *XEth) Coinbase() string {
	eb, err := self.EthereumService().Etherbase()
	if err != nil {
		return "0x0"
	}
	return eb.Hex()
}

func (self *XEth) NumberToHuman(balance string) string {
	b := common.Big(balance)

	return common.CurrencyToString(b)
}

func (self *XEth) StorageAt(addr, storageAddr string) string {
	return self.State().state.GetState(common.HexToAddress(addr), common.HexToHash(storageAddr)).Hex()
}

func (self *XEth) BalanceAt(addr string) string {
	return common.ToHex(self.State().state.GetBalance(common.HexToAddress(addr)).Bytes())
}

func (self *XEth) TxCountAt(address string) int {
	return int(self.State().state.GetNonce(common.HexToAddress(address)))
}

func (self *XEth) CodeAt(address string) string {
	return common.ToHex(self.State().state.GetCode(common.HexToAddress(address)))
}

func (self *XEth) CodeAtBytes(address string) []byte {
	return self.State().SafeGet(address).Code()
}

func (self *XEth) IsContract(address string) bool {
	return len(self.State().SafeGet(address).Code()) > 0
}

func (self *XEth) UninstallFilter(id int) bool {
	defer self.filterManager.Remove(id)

	if _, ok := self.logQueue[id]; ok {
		self.logMu.Lock()
		defer self.logMu.Unlock()
		delete(self.logQueue, id)
		return true
	}
	if _, ok := self.blockQueue[id]; ok {
		self.blockMu.Lock()
		defer self.blockMu.Unlock()
		delete(self.blockQueue, id)
		return true
	}
	if _, ok := self.transactionQueue[id]; ok {
		self.transactionMu.Lock()
		defer self.transactionMu.Unlock()
		delete(self.transactionQueue, id)
		return true
	}

	return false
}

func (self *XEth) NewLogFilter(earliest, latest int64, skip, max int, address []string, topics [][]string) int {
	self.logMu.Lock()
	defer self.logMu.Unlock()

	filter := filters.New(self.EthereumService().ChainDb())
	id := self.filterManager.Add(filter)
	self.logQueue[id] = &logQueue{timeout: time.Now()}

	filter.SetBeginBlock(earliest)
	filter.SetEndBlock(latest)
	filter.SetAddresses(cAddress(address))
	filter.SetTopics(cTopics(topics))
	filter.LogsCallback = func(logs vm.Logs) {
		self.logMu.Lock()
		defer self.logMu.Unlock()

		if queue := self.logQueue[id]; queue != nil {
			queue.add(logs...)
		}
	}

	return id
}

func (self *XEth) NewTransactionFilter() int {
	self.transactionMu.Lock()
	defer self.transactionMu.Unlock()

	filter := filters.New(self.EthereumService().ChainDb())
	id := self.filterManager.Add(filter)
	self.transactionQueue[id] = &hashQueue{timeout: time.Now()}

	filter.TransactionCallback = func(tx *types.Transaction) {
		self.transactionMu.Lock()
		defer self.transactionMu.Unlock()

		if queue := self.transactionQueue[id]; queue != nil {
			queue.add(tx.Hash())
		}
	}
	return id
}

func (self *XEth) NewBlockFilter() int {
	self.blockMu.Lock()
	defer self.blockMu.Unlock()

	filter := filters.New(self.EthereumService().ChainDb())
	id := self.filterManager.Add(filter)
	self.blockQueue[id] = &hashQueue{timeout: time.Now()}

	filter.BlockCallback = func(block *types.Block, logs vm.Logs) {
		self.blockMu.Lock()
		defer self.blockMu.Unlock()

		if queue := self.blockQueue[id]; queue != nil {
			queue.add(block.Hash())
		}
	}
	return id
}

func (self *XEth) GetFilterType(id int) byte {
	if _, ok := self.blockQueue[id]; ok {
		return BlockFilterTy
	} else if _, ok := self.transactionQueue[id]; ok {
		return TransactionFilterTy
	} else if _, ok := self.logQueue[id]; ok {
		return LogFilterTy
	}

	return UnknownFilterTy
}

func (self *XEth) LogFilterChanged(id int) vm.Logs {
	self.logMu.Lock()
	defer self.logMu.Unlock()

	if self.logQueue[id] != nil {
		return self.logQueue[id].get()
	}
	return nil
}

func (self *XEth) BlockFilterChanged(id int) []common.Hash {
	self.blockMu.Lock()
	defer self.blockMu.Unlock()

	if self.blockQueue[id] != nil {
		return self.blockQueue[id].get()
	}
	return nil
}

func (self *XEth) TransactionFilterChanged(id int) []common.Hash {
	self.blockMu.Lock()
	defer self.blockMu.Unlock()

	if self.transactionQueue[id] != nil {
		return self.transactionQueue[id].get()
	}
	return nil
}

func (self *XEth) Logs(id int) vm.Logs {
	filter := self.filterManager.Get(id)
	if filter != nil {
		return filter.Find()
	}

	return nil
}

func (self *XEth) AllLogs(earliest, latest int64, skip, max int, address []string, topics [][]string) vm.Logs {
	filter := filters.New(self.EthereumService().ChainDb())
	filter.SetBeginBlock(earliest)
	filter.SetEndBlock(latest)
	filter.SetAddresses(cAddress(address))
	filter.SetTopics(cTopics(topics))

	return filter.Find()
}

// NewWhisperFilter creates and registers a new message filter to watch for
// inbound whisper messages. All parameters at this point are assumed to be
// HEX encoded.
func (p *XEth) NewWhisperFilter(to, from string, topics [][]string) int {
	// Pre-define the id to be filled later
	var id int

	// Callback to delegate core whisper messages to this xeth filter
	callback := func(msg WhisperMessage) {
		p.messagesMu.RLock() // Only read lock to the filter pool
		defer p.messagesMu.RUnlock()
		if p.messages[id] != nil {
			p.messages[id].insert(msg)
		}
	}
	// Initialize the core whisper filter and wrap into xeth
	id = p.Whisper().Watch(to, from, topics, callback)

	p.messagesMu.Lock()
	p.messages[id] = newWhisperFilter(id, p.Whisper())
	p.messagesMu.Unlock()

	return id
}

// UninstallWhisperFilter disables and removes an existing filter.
func (p *XEth) UninstallWhisperFilter(id int) bool {
	p.messagesMu.Lock()
	defer p.messagesMu.Unlock()

	if _, ok := p.messages[id]; ok {
		delete(p.messages, id)
		return true
	}
	return false
}

// WhisperMessages retrieves all the known messages that match a specific filter.
func (self *XEth) WhisperMessages(id int) []WhisperMessage {
	self.messagesMu.RLock()
	defer self.messagesMu.RUnlock()

	if self.messages[id] != nil {
		return self.messages[id].messages()
	}
	return nil
}

// WhisperMessagesChanged retrieves all the new messages matched by a filter
// since the last retrieval
func (self *XEth) WhisperMessagesChanged(id int) []WhisperMessage {
	self.messagesMu.RLock()
	defer self.messagesMu.RUnlock()

	if self.messages[id] != nil {
		return self.messages[id].retrieve()
	}
	return nil
}

// func (self *XEth) Register(args string) bool {
// 	self.regmut.Lock()
// 	defer self.regmut.Unlock()

// 	if _, ok := self.register[args]; ok {
// 		self.register[args] = nil // register with empty
// 	}
// 	return true
// }

// func (self *XEth) Unregister(args string) bool {
// 	self.regmut.Lock()
// 	defer self.regmut.Unlock()

// 	if _, ok := self.register[args]; ok {
// 		delete(self.register, args)
// 		return true
// 	}

// 	return false
// }

// // TODO improve return type
// func (self *XEth) PullWatchTx(args string) []*interface{} {
// 	self.regmut.Lock()
// 	defer self.regmut.Unlock()

// 	txs := self.register[args]
// 	self.register[args] = nil

// 	return txs
// }

type KeyVal struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (self *XEth) EachStorage(addr string) string {
	var values []KeyVal
	object := self.State().SafeGet(addr)
	it := object.Trie().Iterator()
	for it.Next() {
		values = append(values, KeyVal{common.ToHex(object.Trie().GetKey(it.Key)), common.ToHex(it.Value)})
	}

	valuesJson, err := json.Marshal(values)
	if err != nil {
		return ""
	}

	return string(valuesJson)
}

func (self *XEth) ToAscii(str string) string {
	padded := common.RightPadBytes([]byte(str), 32)

	return "0x" + common.ToHex(padded)
}

func (self *XEth) FromAscii(str string) string {
	if common.IsHex(str) {
		str = str[2:]
	}

	return string(bytes.Trim(common.FromHex(str), "\x00"))
}

func (self *XEth) FromNumber(str string) string {
	if common.IsHex(str) {
		str = str[2:]
	}

	return common.BigD(common.FromHex(str)).String()
}

func (self *XEth) PushTx(encodedTx string) (string, error) {
	tx := new(types.Transaction)
	err := rlp.DecodeBytes(common.FromHex(encodedTx), tx)
	if err != nil {
		glog.V(logger.Error).Infoln(err)
		return "", err
	}

	err = self.EthereumService().TxPool().Add(tx)
	if err != nil {
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

func (self *XEth) Call(fromStr, toStr, valueStr, gasStr, gasPriceStr, dataStr string) (string, string, error) {
	statedb := self.State().State().Copy()
	var from *state.StateObject
	if len(fromStr) == 0 {
		accounts, err := self.EthereumService().AccountManager().Accounts()
		if err != nil || len(accounts) == 0 {
			from = statedb.GetOrNewStateObject(common.Address{})
		} else {
			from = statedb.GetOrNewStateObject(accounts[0].Address)
		}
	} else {
		from = statedb.GetOrNewStateObject(common.HexToAddress(fromStr))
	}

	from.SetBalance(common.MaxBig)

	msg := callmsg{
		from:     from,
		gas:      common.Big(gasStr),
		gasPrice: common.Big(gasPriceStr),
		value:    common.Big(valueStr),
		data:     common.FromHex(dataStr),
	}
	if len(toStr) > 0 {
		addr := common.HexToAddress(toStr)
		msg.to = &addr
	}

	if msg.gas.Cmp(big.NewInt(0)) == 0 {
		msg.gas = big.NewInt(50000000)
	}

	if msg.gasPrice.Cmp(big.NewInt(0)) == 0 {
		msg.gasPrice = self.DefaultGasPrice()
	}

	header := self.CurrentBlock().Header()
	vmenv := core.NewEnv(statedb, self.EthereumService().BlockChain(), msg, header)
	gp := new(core.GasPool).AddGas(common.MaxBig)
	res, gas, err := core.ApplyMessage(vmenv, msg, gp)
	return common.ToHex(res), gas.String(), err
}

func (self *XEth) ConfirmTransaction(tx string) bool {
	return self.frontend.ConfirmTransaction(tx)
}

func (self *XEth) doSign(from common.Address, hash common.Hash, didUnlock bool) ([]byte, error) {
	sig, err := self.EthereumService().AccountManager().Sign(accounts.Account{Address: from}, hash.Bytes())
	if err == accounts.ErrLocked {
		if didUnlock {
			return nil, fmt.Errorf("signer account still locked after successful unlock")
		}
		if !self.frontend.UnlockAccount(from.Bytes()) {
			return nil, fmt.Errorf("could not unlock signer account")
		}
		// retry signing, the account should now be unlocked.
		return self.doSign(from, hash, true)
	} else if err != nil {
		return nil, err
	}
	return sig, nil
}

func (self *XEth) Sign(fromStr, hashStr string, didUnlock bool) (string, error) {
	var (
		from = common.HexToAddress(fromStr)
		hash = common.HexToHash(hashStr)
	)
	sig, err := self.doSign(from, hash, didUnlock)
	if err != nil {
		return "", err
	}
	return common.ToHex(sig), nil
}

func isAddress(addr string) bool {
	return addrReg.MatchString(addr)
}

func (self *XEth) Frontend() Frontend {
	return self.frontend
}

func (self *XEth) SignTransaction(fromStr, toStr, nonceStr, valueStr, gasStr, gasPriceStr, codeStr string) (*types.Transaction, error) {
	if len(toStr) > 0 && toStr != "0x" && !isAddress(toStr) {
		return nil, errors.New("Invalid address")
	}

	var (
		from             = common.HexToAddress(fromStr)
		to               = common.HexToAddress(toStr)
		value            = common.Big(valueStr)
		gas              *big.Int
		price            *big.Int
		data             []byte
		contractCreation bool
	)

	if len(gasStr) == 0 {
		gas = DefaultGas()
	} else {
		gas = common.Big(gasStr)
	}

	if len(gasPriceStr) == 0 {
		price = self.DefaultGasPrice()
	} else {
		price = common.Big(gasPriceStr)
	}

	data = common.FromHex(codeStr)
	if len(toStr) == 0 {
		contractCreation = true
	}

	var nonce uint64
	if len(nonceStr) != 0 {
		nonce = common.Big(nonceStr).Uint64()
	} else {
		state := self.EthereumService().TxPool().State()
		nonce = state.GetNonce(from)
	}
	var tx *types.Transaction
	if contractCreation {
		tx = types.NewContractCreation(nonce, value, gas, price, data)
	} else {
		tx = types.NewTransaction(nonce, to, value, gas, price, data)
	}

	signed, err := self.sign(tx, from, false)
	if err != nil {
		return nil, err
	}

	return signed, nil
}

func (self *XEth) Transact(fromStr, toStr, nonceStr, valueStr, gasStr, gasPriceStr, codeStr string) (string, error) {

	// this minimalistic recoding is enough (works for natspec.js)
	var jsontx = fmt.Sprintf(`{"params":[{"to":"%s","data": "%s"}]}`, toStr, codeStr)
	if !self.ConfirmTransaction(jsontx) {
		err := fmt.Errorf("Transaction not confirmed")
		return "", err
	}

	if len(toStr) > 0 && toStr != "0x" && !isAddress(toStr) {
		return "", errors.New("Invalid address")
	}

	var (
		from             = common.HexToAddress(fromStr)
		to               = common.HexToAddress(toStr)
		value            = common.Big(valueStr)
		gas              *big.Int
		price            *big.Int
		data             []byte
		contractCreation bool
	)

	if len(gasStr) == 0 {
		gas = DefaultGas()
	} else {
		gas = common.Big(gasStr)
	}

	if len(gasPriceStr) == 0 {
		price = self.DefaultGasPrice()
	} else {
		price = common.Big(gasPriceStr)
	}

	data = common.FromHex(codeStr)
	if len(toStr) == 0 {
		contractCreation = true
	}

	// 2015-05-18 Is this still needed?
	// TODO if no_private_key then
	//if _, exists := p.register[args.From]; exists {
	//	p.register[args.From] = append(p.register[args.From], args)
	//} else {
	/*
		account := accounts.Get(common.FromHex(args.From))
		if account != nil {
			if account.Unlocked() {
				if !unlockAccount(account) {
					return
				}
			}

			result, _ := account.Transact(common.FromHex(args.To), common.FromHex(args.Value), common.FromHex(args.Gas), common.FromHex(args.GasPrice), common.FromHex(args.Data))
			if len(result) > 0 {
				*reply = common.ToHex(result)
			}
		} else if _, exists := p.register[args.From]; exists {
			p.register[ags.From] = append(p.register[args.From], args)
		}
	*/

	self.transactMu.Lock()
	defer self.transactMu.Unlock()

	var nonce uint64
	if len(nonceStr) != 0 {
		nonce = common.Big(nonceStr).Uint64()
	} else {
		state := self.EthereumService().TxPool().State()
		nonce = state.GetNonce(from)
	}
	var tx *types.Transaction
	if contractCreation {
		tx = types.NewContractCreation(nonce, value, gas, price, data)
	} else {
		tx = types.NewTransaction(nonce, to, value, gas, price, data)
	}

	signed, err := self.sign(tx, from, false)
	if err != nil {
		return "", err
	}
	if err = self.EthereumService().TxPool().Add(signed); err != nil {
		return "", err
	}

	if contractCreation {
		addr := crypto.CreateAddress(from, nonce)
		glog.V(logger.Info).Infof("Tx(%s) created: %s\n", signed.Hash().Hex(), addr.Hex())
	} else {
		glog.V(logger.Info).Infof("Tx(%s) to: %s\n", signed.Hash().Hex(), tx.To().Hex())
	}

	return signed.Hash().Hex(), nil
}

func (self *XEth) sign(tx *types.Transaction, from common.Address, didUnlock bool) (*types.Transaction, error) {
	hash := tx.SigHash()
	sig, err := self.doSign(from, hash, didUnlock)
	if err != nil {
		return tx, err
	}
	return tx.WithSignature(sig)
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

type logQueue struct {
	mu sync.Mutex

	logs    vm.Logs
	timeout time.Time
	id      int
}

func (l *logQueue) add(logs ...*vm.Log) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logs = append(l.logs, logs...)
}

func (l *logQueue) get() vm.Logs {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.timeout = time.Now()
	tmp := l.logs
	l.logs = nil
	return tmp
}

type hashQueue struct {
	mu sync.Mutex

	hashes  []common.Hash
	timeout time.Time
	id      int
}

func (l *hashQueue) add(hashes ...common.Hash) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.hashes = append(l.hashes, hashes...)
}

func (l *hashQueue) get() []common.Hash {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.timeout = time.Now()
	tmp := l.hashes
	l.hashes = nil
	return tmp
}
