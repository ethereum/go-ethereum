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
	"github.com/ethereum/go-ethereum/miner"
)

var (
	pipelogger       = logger.NewLogger("XETH")
	filterTickerTime = 5 * time.Minute
	defaultGasPrice  = big.NewInt(10000000000000) //150000000000
	defaultGas       = big.NewInt(90000)          //500000
)

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
				if time.Since(filter.timeout) > filterTickerTime {
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

func (self *XEth) DefaultGas() *big.Int      { return defaultGas }
func (self *XEth) DefaultGasPrice() *big.Int { return defaultGasPrice }

func (self *XEth) RemoteMining() *miner.RemoteAgent { return self.agent }

func (self *XEth) AtStateNum(num int64) *XEth {
	block := self.getBlockByHeight(num)

	var st *state.StateDB
	if block != nil {
		st = state.New(block.Root(), self.backend.StateDb())
	} else {
		st = self.backend.ChainManager().State()
	}

	return self.withState(st)
}

func (self *XEth) withState(statedb *state.StateDB) *XEth {
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

	if height < 0 {
		num = self.CurrentBlock().NumberU64() + uint64(-1*height)
	} else {
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

func (self *XEth) EthTransactionByHash(hash string) *types.Transaction {
	data, _ := self.backend.ExtraDb().Get(common.FromHex(hash))
	if len(data) != 0 {
		return types.NewTransactionFromBytes(data)
	}
	return nil
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

func (self *XEth) PeerCount() int {
	return self.backend.PeerCount()
}

func (self *XEth) IsMining() bool {
	return self.backend.IsMining()
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
	cb, _ := self.backend.AccountManager().Coinbase()
	return common.ToHex(cb)
}

func (self *XEth) NumberToHuman(balance string) string {
	b := common.Big(balance)

	return common.CurrencyToString(b)
}

func (self *XEth) StorageAt(addr, storageAddr string) string {
	storage := self.State().SafeGet(addr).StorageString(storageAddr)

	return common.ToHex(storage.Bytes())
}

func (self *XEth) BalanceAt(addr string) string {
	return self.State().SafeGet(addr).Balance().String()
}

func (self *XEth) TxCountAt(address string) int {
	return int(self.State().SafeGet(address).Nonce())
}

func (self *XEth) CodeAt(address string) string {
	return common.ToHex(self.State().SafeGet(address).Code())
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

func (self *XEth) RegisterFilter(args *core.FilterOptions) int {
	var id int
	filter := core.NewFilter(self.backend)
	filter.SetOptions(args)
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

			self.logs[id].add(&state.StateLog{})
		}
	case "latest":
		filter.BlockCallback = func(block *types.Block, logs state.Logs) {
			self.logMut.Lock()
			defer self.logMut.Unlock()

			for _, log := range logs {
				self.logs[id].add(log)
			}
			self.logs[id].add(&state.StateLog{})
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

func (self *XEth) AllLogs(args *core.FilterOptions) state.Logs {
	filter := core.NewFilter(self.backend)
	filter.SetOptions(args)

	return filter.Find()
}

func (p *XEth) NewWhisperFilter(opts *Options) int {
	var id int
	opts.Fn = func(msg WhisperMessage) {
		p.messagesMut.Lock()
		defer p.messagesMut.Unlock()
		p.messages[id].add(msg) // = append(p.messages[id], msg)
	}
	id = p.Whisper().Watch(opts)
	p.messages[id] = &whisperFilter{timeout: time.Now()}
	return id
}

func (p *XEth) UninstallWhisperFilter(id int) bool {
	if _, ok := p.messages[id]; ok {
		delete(p.messages, id)
		return true
	}

	return false
}

func (self *XEth) MessagesChanged(id int) []WhisperMessage {
	self.messagesMut.Lock()
	defer self.messagesMut.Unlock()

	if self.messages[id] != nil {
		return self.messages[id].get()
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
	msg := callmsg{
		from:     statedb.GetOrNewStateObject(common.HexToAddress(fromStr)),
		to:       common.HexToAddress(toStr),
		gas:      common.Big(gasStr),
		gasPrice: common.Big(gasPriceStr),
		value:    common.Big(valueStr),
		data:     common.FromHex(dataStr),
	}
	if msg.gas.Cmp(big.NewInt(0)) == 0 {
		msg.gas = defaultGas
	}

	if msg.gasPrice.Cmp(big.NewInt(0)) == 0 {
		msg.gasPrice = defaultGasPrice
	}

	block := self.CurrentBlock()
	vmenv := core.NewEnv(statedb, self.backend.ChainManager(), msg, block)

	res, err := vmenv.Call(msg.from, msg.to, msg.data, msg.gas, msg.gasPrice, msg.value)
	return common.ToHex(res), err
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
		gas = defaultGas
	}

	if price.Cmp(big.NewInt(0)) == 0 {
		price = defaultGasPrice
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
		pipelogger.Infof("Contract addr %x\n", addr)

		return core.AddressFromMessage(tx).Hex(), nil
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

type whisperFilter struct {
	messages []WhisperMessage
	timeout  time.Time
	id       int
}

func (w *whisperFilter) add(msgs ...WhisperMessage) {
	w.messages = append(w.messages, msgs...)
}
func (w *whisperFilter) get() []WhisperMessage {
	w.timeout = time.Now()
	tmp := w.messages
	w.messages = nil
	return tmp
}

type logFilter struct {
	logs    state.Logs
	timeout time.Time
	id      int
}

func (l *logFilter) add(logs ...state.Log) {
	l.logs = append(l.logs, logs...)
}

func (l *logFilter) get() state.Logs {
	l.timeout = time.Now()
	tmp := l.logs
	l.logs = nil
	return tmp
}
