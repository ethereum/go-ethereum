// eXtended ETHereum
package xeth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/event/filter"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	filterTickerTime = 5 * time.Minute
	defaultGasPrice  = big.NewInt(10000000000000) //150000000000
	defaultGas       = big.NewInt(90000)          //500000
)

func DefaultGas() *big.Int      { return new(big.Int).Set(defaultGas) }
func DefaultGasPrice() *big.Int { return new(big.Int).Set(defaultGasPrice) }

type XEth struct {
	backend  *eth.Ethereum
	frontend Frontend

	state   *State
	whisper *Whisper

	quit          chan struct{}
	filterManager *filter.FilterManager

	logMut sync.RWMutex
	logs   map[int]*logFilter

	messagesMut sync.RWMutex
	messages    map[int]*whisperFilter

	// regmut   sync.Mutex
	// register map[string][]*interface{} // TODO improve return type

	agent *miner.RemoteAgent
}

// New creates an XEth that uses the given frontend.
// If a nil Frontend is provided, a default frontend which
// confirms all transactions will be used.
func New(eth *eth.Ethereum, frontend Frontend) *XEth {
	xeth := &XEth{
		backend:       eth,
		frontend:      frontend,
		whisper:       NewWhisper(eth.Whisper()),
		quit:          make(chan struct{}),
		filterManager: filter.NewFilterManager(eth.EventMux()),
		logs:          make(map[int]*logFilter),
		messages:      make(map[int]*whisperFilter),
		agent:         miner.NewRemoteAgent(),
	}
	eth.Miner().Register(xeth.agent)

	if frontend == nil {
		xeth.frontend = dummyFrontend{}
	}
	xeth.state = NewState(xeth, xeth.backend.ChainManager().TransState())

	go xeth.start()
	go xeth.filterManager.Start()

	return xeth
}

func (self *XEth) start() {
	timer := time.NewTicker(2 * time.Second)
done:
	for {
		select {
		case <-timer.C:
			self.logMut.Lock()
			self.messagesMut.Lock()
			for id, filter := range self.logs {
				if time.Since(filter.timeout) > filterTickerTime {
					self.filterManager.UninstallFilter(id)
					delete(self.logs, id)
				}
			}

			for id, filter := range self.messages {
				if time.Since(filter.activity()) > filterTickerTime {
					self.Whisper().Unwatch(id)
					delete(self.messages, id)
				}
			}
			self.messagesMut.Unlock()
			self.logMut.Unlock()
		case <-self.quit:
			break done
		}
	}
}

func (self *XEth) stop() {
	close(self.quit)
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

func (self *XEth) RemoteMining() *miner.RemoteAgent { return self.agent }

func (self *XEth) AtStateNum(num int64) *XEth {
	var st *state.StateDB
	switch num {
	case -2:
		st = self.backend.Miner().PendingState().Copy()
	default:
		if block := self.getBlockByHeight(num); block != nil {
			st = state.New(block.Root(), self.backend.StateDb())
		} else {
			st = state.New(self.backend.ChainManager().GetBlockByNumber(0).Root(), self.backend.StateDb())
		}
	}

	return self.WithState(st)
}

func (self *XEth) WithState(statedb *state.StateDB) *XEth {
	xeth := &XEth{
		backend: self.backend,
	}

	xeth.state = NewState(xeth, statedb)
	return xeth
}

func (self *XEth) State() *State { return self.state }

func (self *XEth) Whisper() *Whisper { return self.whisper }

func (self *XEth) getBlockByHeight(height int64) *types.Block {
	var num uint64

	switch height {
	case -2:
		return self.backend.Miner().PendingBlock()
	case -1:
		return self.CurrentBlock()
	default:
		if height < 0 {
			return nil
		}

		num = uint64(height)
	}

	return self.backend.ChainManager().GetBlockByNumber(num)
}

func (self *XEth) BlockByHash(strHash string) *Block {
	hash := common.HexToHash(strHash)
	block := self.backend.ChainManager().GetBlock(hash)

	return NewBlock(block)
}

func (self *XEth) EthBlockByHash(strHash string) *types.Block {
	hash := common.HexToHash(strHash)
	block := self.backend.ChainManager().GetBlock(hash)

	return block
}

func (self *XEth) EthTransactionByHash(hash string) (tx *types.Transaction, blhash common.Hash, blnum *big.Int, txi uint64) {
	data, _ := self.backend.ExtraDb().Get(common.FromHex(hash))
	if len(data) != 0 {
		tx = types.NewTransactionFromBytes(data)
	}

	// meta
	var txExtra struct {
		BlockHash  common.Hash
		BlockIndex uint64
		Index      uint64
	}

	v, _ := self.backend.ExtraDb().Get(append(common.FromHex(hash), 0x0001))
	r := bytes.NewReader(v)
	err := rlp.Decode(r, &txExtra)
	if err == nil {
		blhash = txExtra.BlockHash
		blnum = big.NewInt(int64(txExtra.BlockIndex))
		txi = txExtra.Index
	} else {
		glog.V(logger.Error).Infoln(err)
	}

	return
}

func (self *XEth) BlockByNumber(num int64) *Block {
	return NewBlock(self.getBlockByHeight(num))
}

func (self *XEth) EthBlockByNumber(num int64) *types.Block {
	return self.getBlockByHeight(num)
}

func (self *XEth) CurrentBlock() *types.Block {
	return self.backend.ChainManager().CurrentBlock()
}

func (self *XEth) GasLimit() *big.Int {
	return self.backend.ChainManager().GasLimit()
}

func (self *XEth) Block(v interface{}) *Block {
	if n, ok := v.(int32); ok {
		return self.BlockByNumber(int64(n))
	} else if str, ok := v.(string); ok {
		return self.BlockByHash(str)
	} else if f, ok := v.(float64); ok { // Don't ask ...
		return self.BlockByNumber(int64(f))
	}

	return nil
}

func (self *XEth) Accounts() []string {
	// TODO: check err?
	accounts, _ := self.backend.AccountManager().Accounts()
	accountAddresses := make([]string, len(accounts))
	for i, ac := range accounts {
		accountAddresses[i] = common.ToHex(ac.Address)
	}
	return accountAddresses
}

func (self *XEth) DbPut(key, val []byte) bool {
	self.backend.ExtraDb().Put(key, val)
	return true
}

func (self *XEth) DbGet(key []byte) ([]byte, error) {
	val, err := self.backend.ExtraDb().Get(key)
	return val, err
}

func (self *XEth) PeerCount() int {
	return self.backend.PeerCount()
}

func (self *XEth) IsMining() bool {
	return self.backend.IsMining()
}

func (self *XEth) HashRate() int64 {
	return self.backend.Miner().HashRate()
}

func (self *XEth) EthVersion() string {
	return fmt.Sprintf("%d", self.backend.EthVersion())
}

func (self *XEth) NetworkVersion() string {
	return fmt.Sprintf("%d", self.backend.NetVersion())
}

func (self *XEth) WhisperVersion() string {
	return fmt.Sprintf("%d", self.backend.ShhVersion())
}

func (self *XEth) ClientVersion() string {
	return self.backend.ClientVersion()
}

func (self *XEth) SetMining(shouldmine bool) bool {
	ismining := self.backend.IsMining()
	if shouldmine && !ismining {
		err := self.backend.StartMining()
		return err == nil
	}
	if ismining && !shouldmine {
		self.backend.StopMining()
	}
	return self.backend.IsMining()
}

func (self *XEth) IsListening() bool {
	return self.backend.IsListening()
}

func (self *XEth) Coinbase() string {
	eb, _ := self.backend.Etherbase()
	return eb.Hex()
}

func (self *XEth) NumberToHuman(balance string) string {
	b := common.Big(balance)

	return common.CurrencyToString(b)
}

func (self *XEth) StorageAt(addr, storageAddr string) string {
	return common.ToHex(self.State().state.GetState(common.HexToAddress(addr), common.HexToHash(storageAddr)))
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

func (self *XEth) SecretToAddress(key string) string {
	pair, err := crypto.NewKeyPairFromSec(common.FromHex(key))
	if err != nil {
		return ""
	}

	return common.ToHex(pair.Address())
}

func (self *XEth) RegisterFilter(earliest, latest int64, skip, max int, address []string, topics [][]string) int {
	var id int
	filter := core.NewFilter(self.backend)
	filter.SetEarliestBlock(earliest)
	filter.SetLatestBlock(latest)
	filter.SetSkip(skip)
	filter.SetMax(max)
	filter.SetAddress(cAddress(address))
	filter.SetTopics(cTopics(topics))
	filter.LogsCallback = func(logs state.Logs) {
		self.logMut.Lock()
		defer self.logMut.Unlock()

		self.logs[id].add(logs...)
	}
	id = self.filterManager.InstallFilter(filter)
	self.logs[id] = &logFilter{timeout: time.Now()}

	return id
}

func (self *XEth) UninstallFilter(id int) bool {
	if _, ok := self.logs[id]; ok {
		delete(self.logs, id)
		self.filterManager.UninstallFilter(id)
		return true
	}

	return false
}

func (self *XEth) NewFilterString(word string) int {
	var id int
	filter := core.NewFilter(self.backend)

	switch word {
	case "pending":
		filter.PendingCallback = func(tx *types.Transaction) {
			self.logMut.Lock()
			defer self.logMut.Unlock()

			self.logs[id].add(&state.Log{})
		}
	case "latest":
		filter.BlockCallback = func(block *types.Block, logs state.Logs) {
			self.logMut.Lock()
			defer self.logMut.Unlock()

			for _, log := range logs {
				self.logs[id].add(log)
			}
			self.logs[id].add(&state.Log{})
		}
	}

	id = self.filterManager.InstallFilter(filter)
	self.logs[id] = &logFilter{timeout: time.Now()}

	return id
}

func (self *XEth) FilterChanged(id int) state.Logs {
	self.logMut.Lock()
	defer self.logMut.Unlock()

	if self.logs[id] != nil {
		return self.logs[id].get()
	}

	return nil
}

func (self *XEth) Logs(id int) state.Logs {
	self.logMut.Lock()
	defer self.logMut.Unlock()

	filter := self.filterManager.GetFilter(id)
	if filter != nil {
		return filter.Find()
	}

	return nil
}

func (self *XEth) AllLogs(earliest, latest int64, skip, max int, address []string, topics [][]string) state.Logs {
	filter := core.NewFilter(self.backend)
	filter.SetEarliestBlock(earliest)
	filter.SetLatestBlock(latest)
	filter.SetSkip(skip)
	filter.SetMax(max)
	filter.SetAddress(cAddress(address))
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
		p.messagesMut.RLock() // Only read lock to the filter pool
		defer p.messagesMut.RUnlock()
		p.messages[id].insert(msg)
	}
	// Initialize the core whisper filter and wrap into xeth
	id = p.Whisper().Watch(to, from, topics, callback)

	p.messagesMut.Lock()
	p.messages[id] = newWhisperFilter(id, p.Whisper())
	p.messagesMut.Unlock()

	return id
}

// UninstallWhisperFilter disables and removes an existing filter.
func (p *XEth) UninstallWhisperFilter(id int) bool {
	p.messagesMut.Lock()
	defer p.messagesMut.Unlock()

	if _, ok := p.messages[id]; ok {
		delete(p.messages, id)
		return true
	}
	return false
}

// WhisperMessages retrieves all the known messages that match a specific filter.
func (self *XEth) WhisperMessages(id int) []WhisperMessage {
	self.messagesMut.RLock()
	defer self.messagesMut.RUnlock()

	if self.messages[id] != nil {
		return self.messages[id].messages()
	}
	return nil
}

// WhisperMessagesChanged retrieves all the new messages matched by a filter
// since the last retrieval
func (self *XEth) WhisperMessagesChanged(id int) []WhisperMessage {
	self.messagesMut.RLock()
	defer self.messagesMut.RUnlock()

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
	tx := types.NewTransactionFromBytes(common.FromHex(encodedTx))
	err := self.backend.TxPool().Add(tx)
	if err != nil {
		return "", err
	}

	if tx.To() == nil {
		addr := core.AddressFromMessage(tx)
		return addr.Hex(), nil
	}
	return tx.Hash().Hex(), nil
}

func (self *XEth) Call(fromStr, toStr, valueStr, gasStr, gasPriceStr, dataStr string) (string, error) {
	statedb := self.State().State() //self.eth.ChainManager().TransState()
	var from *state.StateObject
	if len(fromStr) == 0 {
		accounts, err := self.backend.AccountManager().Accounts()
		if err != nil || len(accounts) == 0 {
			from = statedb.GetOrNewStateObject(common.Address{})
		} else {
			from = statedb.GetOrNewStateObject(common.BytesToAddress(accounts[0].Address))
		}
	} else {
		from = statedb.GetOrNewStateObject(common.HexToAddress(fromStr))
	}

	msg := callmsg{
		from:     from,
		to:       common.HexToAddress(toStr),
		gas:      common.Big(gasStr),
		gasPrice: common.Big(gasPriceStr),
		value:    common.Big(valueStr),
		data:     common.FromHex(dataStr),
	}

	if msg.gas.Cmp(big.NewInt(0)) == 0 {
		msg.gas = DefaultGas()
	}

	if msg.gasPrice.Cmp(big.NewInt(0)) == 0 {
		msg.gasPrice = DefaultGasPrice()
	}

	block := self.CurrentBlock()
	vmenv := core.NewEnv(statedb, self.backend.ChainManager(), msg, block)

	res, err := vmenv.Call(msg.from, msg.to, msg.data, msg.gas, msg.gasPrice, msg.value)
	return common.ToHex(res), err
}

func (self *XEth) ConfirmTransaction(tx string) bool {

	return self.frontend.ConfirmTransaction(tx)

}

func (self *XEth) Transact(fromStr, toStr, valueStr, gasStr, gasPriceStr, codeStr string) (string, error) {
	var (
		from             = common.HexToAddress(fromStr)
		to               = common.HexToAddress(toStr)
		value            = common.NewValue(valueStr)
		gas              = common.Big(gasStr)
		price            = common.Big(gasPriceStr)
		data             []byte
		contractCreation bool
	)

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

	// TODO: align default values to have the same type, e.g. not depend on
	// common.Value conversions later on
	if gas.Cmp(big.NewInt(0)) == 0 {
		gas = DefaultGas()
	}

	if price.Cmp(big.NewInt(0)) == 0 {
		price = DefaultGasPrice()
	}

	data = common.FromHex(codeStr)
	if len(toStr) == 0 {
		contractCreation = true
	}

	var tx *types.Transaction
	if contractCreation {
		tx = types.NewContractCreationTx(value.BigInt(), gas, price, data)
	} else {
		tx = types.NewTransactionMessage(to, value.BigInt(), gas, price, data)
	}

	state := self.backend.ChainManager().TxState()
	nonce := state.NewNonce(from)
	tx.SetNonce(nonce)

	if err := self.sign(tx, from, false); err != nil {
		return "", err
	}
	if err := self.backend.TxPool().Add(tx); err != nil {
		return "", err
	}

	if contractCreation {
		addr := core.AddressFromMessage(tx)
		glog.V(logger.Info).Infof("Tx(%x) created: %x\n", tx.Hash(), addr)

		return core.AddressFromMessage(tx).Hex(), nil
	} else {
		glog.V(logger.Info).Infof("Tx(%x) to: %x\n", tx.Hash(), tx.To())
	}
	return tx.Hash().Hex(), nil
}

func (self *XEth) sign(tx *types.Transaction, from common.Address, didUnlock bool) error {
	sig, err := self.backend.AccountManager().Sign(accounts.Account{Address: from.Bytes()}, tx.Hash().Bytes())
	if err == accounts.ErrLocked {
		if didUnlock {
			return fmt.Errorf("sender account still locked after successful unlock")
		}
		if !self.frontend.UnlockAccount(from.Bytes()) {
			return fmt.Errorf("could not unlock sender account")
		}
		// retry signing, the account should now be unlocked.
		return self.sign(tx, from, true)
	} else if err != nil {
		return err
	}
	tx.SetSignatureValues(sig)
	return nil
}

// callmsg is the message type used for call transations.
type callmsg struct {
	from          *state.StateObject
	to            common.Address
	gas, gasPrice *big.Int
	value         *big.Int
	data          []byte
}

// accessor boilerplate to implement core.Message
func (m callmsg) From() (common.Address, error) { return m.from.Address(), nil }
func (m callmsg) Nonce() uint64                 { return m.from.Nonce() }
func (m callmsg) To() *common.Address           { return &m.to }
func (m callmsg) GasPrice() *big.Int            { return m.gasPrice }
func (m callmsg) Gas() *big.Int                 { return m.gas }
func (m callmsg) Value() *big.Int               { return m.value }
func (m callmsg) Data() []byte                  { return m.data }

type logFilter struct {
	logs    state.Logs
	timeout time.Time
	id      int
}

func (l *logFilter) add(logs ...*state.Log) {
	l.logs = append(l.logs, logs...)
}

func (l *logFilter) get() state.Logs {
	l.timeout = time.Now()
	tmp := l.logs
	l.logs = nil
	return tmp
}
