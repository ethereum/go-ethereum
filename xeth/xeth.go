package xeth

/*
 * eXtended ETHereum
 */

import (
	"bytes"
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/ui"
	"github.com/ethereum/go-ethereum/whisper"
)

var pipelogger = logger.NewLogger("XETH")

// to resolve the import cycle
type Backend interface {
	BlockProcessor() *core.BlockProcessor
	ChainManager() *core.ChainManager
	AccountManager() *accounts.Manager
	TxPool() *core.TxPool
	PeerCount() int
	IsListening() bool
	Peers() []*p2p.Peer
	BlockDb() ethutil.Database
	StateDb() ethutil.Database
	EventMux() *event.TypeMux
	Whisper() *whisper.Whisper
	Miner() *miner.Miner
}

type XEth struct {
	eth            Backend
	blockProcessor *core.BlockProcessor
	chainManager   *core.ChainManager
	accountManager *accounts.Manager
	state          *State
	whisper        *Whisper
	miner          *miner.Miner

	frontend ui.Interface
}

type TmpFrontend struct{}

func (TmpFrontend) UnlockAccount([]byte) bool                  { panic("UNLOCK ACCOUNT") }
func (TmpFrontend) ConfirmTransaction(*types.Transaction) bool { panic("CONFIRM TRANSACTION") }

func New(eth Backend, frontend ui.Interface) *XEth {
	xeth := &XEth{
		eth:            eth,
		blockProcessor: eth.BlockProcessor(),
		chainManager:   eth.ChainManager(),
		accountManager: eth.AccountManager(),
		whisper:        NewWhisper(eth.Whisper()),
		miner:          eth.Miner(),
	}

	if frontend == nil {
		xeth.frontend = TmpFrontend{}
	}

	xeth.state = NewState(xeth, xeth.chainManager.TransState())

	return xeth
}

func (self *XEth) Backend() Backend { return self.eth }
func (self *XEth) UseState(statedb *state.StateDB) *XEth {
	xeth := &XEth{
		eth:            self.eth,
		blockProcessor: self.blockProcessor,
		chainManager:   self.chainManager,
		whisper:        self.whisper,
		miner:          self.miner,
	}

	xeth.state = NewState(xeth, statedb)
	return xeth
}
func (self *XEth) State() *State { return self.state }

func (self *XEth) Whisper() *Whisper   { return self.whisper }
func (self *XEth) Miner() *miner.Miner { return self.miner }

func (self *XEth) BlockByHash(strHash string) *Block {
	hash := fromHex(strHash)
	block := self.chainManager.GetBlock(hash)

	return NewBlock(block)
}

func (self *XEth) BlockByNumber(num int32) *Block {
	if num == -1 {
		return NewBlock(self.chainManager.CurrentBlock())
	}

	return NewBlock(self.chainManager.GetBlockByNumber(uint64(num)))
}

func (self *XEth) Block(v interface{}) *Block {
	if n, ok := v.(int32); ok {
		return self.BlockByNumber(n)
	} else if str, ok := v.(string); ok {
		return self.BlockByHash(str)
	} else if f, ok := v.(float64); ok { // Don't ask ...
		return self.BlockByNumber(int32(f))
	}

	return nil
}

func (self *XEth) Accounts() []string {
	// TODO: check err?
	accounts, _ := self.eth.AccountManager().Accounts()
	accountAddresses := make([]string, len(accounts))
	for i, ac := range accounts {
		accountAddresses[i] = toHex(ac.Address)
	}
	return accountAddresses
}

func (self *XEth) PeerCount() int {
	return self.eth.PeerCount()
}

func (self *XEth) IsMining() bool {
	return self.miner.Mining()
}

func (self *XEth) SetMining(shouldmine bool) bool {
	ismining := self.miner.Mining()
	if shouldmine && !ismining {
		self.miner.Start()
	}
	if ismining && !shouldmine {
		self.miner.Stop()
	}
	return self.miner.Mining()
}

func (self *XEth) IsListening() bool {
	return self.eth.IsListening()
}

func (self *XEth) Coinbase() string {
	cb, _ := self.eth.AccountManager().Coinbase()
	return toHex(cb)
}

func (self *XEth) NumberToHuman(balance string) string {
	b := ethutil.Big(balance)

	return ethutil.CurrencyToString(b)
}

func (self *XEth) StorageAt(addr, storageAddr string) string {
	storage := self.State().SafeGet(addr).StorageString(storageAddr)

	return toHex(storage.Bytes())
}

func (self *XEth) BalanceAt(addr string) string {
	return self.State().SafeGet(addr).Balance().String()
}

func (self *XEth) TxCountAt(address string) int {
	return int(self.State().SafeGet(address).Nonce())
}

func (self *XEth) CodeAt(address string) string {
	return toHex(self.State().SafeGet(address).Code())
}

func (self *XEth) IsContract(address string) bool {
	return len(self.State().SafeGet(address).Code()) > 0
}

func (self *XEth) SecretToAddress(key string) string {
	pair, err := crypto.NewKeyPairFromSec(fromHex(key))
	if err != nil {
		return ""
	}

	return toHex(pair.Address())
}

func (self *XEth) Execute(addr, value, gas, price, data string) (string, error) {
	return "", nil
}

type KeyVal struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (self *XEth) EachStorage(addr string) string {
	var values []KeyVal
	object := self.State().SafeGet(addr)
	it := object.Trie().Iterator()
	for it.Next() {
		values = append(values, KeyVal{toHex(it.Key), toHex(it.Value)})
	}

	valuesJson, err := json.Marshal(values)
	if err != nil {
		return ""
	}

	return string(valuesJson)
}

func (self *XEth) ToAscii(str string) string {
	padded := ethutil.RightPadBytes([]byte(str), 32)

	return "0x" + toHex(padded)
}

func (self *XEth) FromAscii(str string) string {
	if ethutil.IsHex(str) {
		str = str[2:]
	}

	return string(bytes.Trim(fromHex(str), "\x00"))
}

func (self *XEth) FromNumber(str string) string {
	if ethutil.IsHex(str) {
		str = str[2:]
	}

	return ethutil.BigD(fromHex(str)).String()
}

func (self *XEth) PushTx(encodedTx string) (string, error) {
	tx := types.NewTransactionFromBytes(fromHex(encodedTx))
	err := self.eth.TxPool().Add(tx)
	if err != nil {
		return "", err
	}

	if tx.To() == nil {
		addr := core.AddressFromMessage(tx)
		return toHex(addr), nil
	}
	return toHex(tx.Hash()), nil
}

func (self *XEth) Call(fromStr, toStr, valueStr, gasStr, gasPriceStr, dataStr string) (string, error) {
	if len(gasStr) == 0 {
		gasStr = "100000"
	}
	if len(gasPriceStr) == 0 {
		gasPriceStr = "1"
	}

	statedb := self.State().State() //self.chainManager.TransState()
	msg := callmsg{
		from:     statedb.GetOrNewStateObject(fromHex(fromStr)),
		to:       fromHex(toStr),
		gas:      ethutil.Big(gasStr),
		gasPrice: ethutil.Big(gasPriceStr),
		value:    ethutil.Big(valueStr),
		data:     fromHex(dataStr),
	}
	block := self.chainManager.CurrentBlock()
	vmenv := core.NewEnv(statedb, self.chainManager, msg, block)

	res, err := vmenv.Call(msg.from, msg.to, msg.data, msg.gas, msg.gasPrice, msg.value)
	return toHex(res), err
}

func (self *XEth) Transact(fromStr, toStr, valueStr, gasStr, gasPriceStr, codeStr string) (string, error) {

	var (
		from             []byte
		to               []byte
		value            = ethutil.NewValue(valueStr)
		gas              = ethutil.NewValue(gasStr)
		price            = ethutil.NewValue(gasPriceStr)
		data             []byte
		contractCreation bool
	)

	from = fromHex(fromStr)
	data = fromHex(codeStr)
	to = fromHex(toStr)
	if len(to) == 0 {
		contractCreation = true
	}

	var tx *types.Transaction
	if contractCreation {
		tx = types.NewContractCreationTx(value.BigInt(), gas.BigInt(), price.BigInt(), data)
	} else {
		tx = types.NewTransactionMessage(to, value.BigInt(), gas.BigInt(), price.BigInt(), data)
	}

	state := self.chainManager.TransState()
	nonce := state.GetNonce(from)

	tx.SetNonce(nonce)
	sig, err := self.accountManager.Sign(accounts.Account{Address: from}, tx.Hash())
	if err != nil {
		return "", err
	}
	tx.SetSignatureValues(sig)

	err = self.eth.TxPool().Add(tx)
	if err != nil {
		return "", err
	}
	state.SetNonce(from, nonce+1)

	if contractCreation {
		addr := core.AddressFromMessage(tx)
		pipelogger.Infof("Contract addr %x\n", addr)
	}

	if types.IsContractAddr(to) {
		return toHex(core.AddressFromMessage(tx)), nil
	}

	return toHex(tx.Hash()), nil
}

// callmsg is the message type used for call transations.
type callmsg struct {
	from          *state.StateObject
	to            []byte
	gas, gasPrice *big.Int
	value         *big.Int
	data          []byte
}

// accessor boilerplate to implement core.Message
func (m callmsg) From() []byte       { return m.from.Address() }
func (m callmsg) Nonce() uint64      { return m.from.Nonce() }
func (m callmsg) To() []byte         { return m.to }
func (m callmsg) GasPrice() *big.Int { return m.gasPrice }
func (m callmsg) Gas() *big.Int      { return m.gas }
func (m callmsg) Value() *big.Int    { return m.value }
func (m callmsg) Data() []byte       { return m.data }
