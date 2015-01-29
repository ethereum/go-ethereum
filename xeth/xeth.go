package xeth

/*
 * eXtended ETHereum
 */

import (
	"bytes"
	"encoding/json"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/state"
)

var pipelogger = logger.NewLogger("XETH")

// to resolve the import cycle
type Backend interface {
	BlockProcessor() *core.BlockProcessor
	ChainManager() *core.ChainManager
	KeyManager() *crypto.KeyManager
	IsMining() bool
	IsListening() bool
	PeerCount() int
	Db() ethutil.Database
	TxPool() *core.TxPool
}

type XEth struct {
	eth            Backend
	blockProcessor *core.BlockProcessor
	chainManager   *core.ChainManager
	world          *State
}

func New(eth Backend) *XEth {
	xeth := &XEth{
		eth:            eth,
		blockProcessor: eth.BlockProcessor(),
		chainManager:   eth.ChainManager(),
	}
	xeth.world = NewState(xeth)

	return xeth
}

func (self *XEth) State() *State { return self.world }

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
	return []string{toHex(self.eth.KeyManager().Address())}
}

func (self *XEth) PeerCount() int {
	return self.eth.PeerCount()
}

func (self *XEth) IsMining() bool {
	return self.eth.IsMining()
}

func (self *XEth) IsListening() bool {
	return self.eth.IsListening()
}

func (self *XEth) Coinbase() string {
	return toHex(self.eth.KeyManager().Address())
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
	return int(self.State().SafeGet(address).Nonce)
}

func (self *XEth) CodeAt(address string) string {
	return toHex(self.State().SafeGet(address).Code)
}

func (self *XEth) IsContract(address string) bool {
	return len(self.State().SafeGet(address).Code) > 0
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

func ToMessages(messages state.Messages) *ethutil.List {
	var msgs []Message
	for _, m := range messages {
		msgs = append(msgs, NewMessage(m))
	}

	return ethutil.NewList(msgs)
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

func (self *XEth) Call(toStr, valueStr, gasStr, gasPriceStr, dataStr string) (string, error) {
	if len(gasStr) == 0 {
		gasStr = "100000"
	}
	if len(gasPriceStr) == 0 {
		gasPriceStr = "1"
	}

	var (
		statedb   = self.chainManager.TransState()
		initiator = state.NewStateObject(self.eth.KeyManager().KeyPair().Address(), self.eth.Db())
		block     = self.chainManager.CurrentBlock()
		to        = statedb.GetOrNewStateObject(fromHex(toStr))
		data      = fromHex(dataStr)
		gas       = ethutil.Big(gasStr)
		price     = ethutil.Big(gasPriceStr)
		value     = ethutil.Big(valueStr)
	)

	vmenv := NewEnv(self.chainManager, statedb, block, value, initiator.Address())
	res, err := vmenv.Call(initiator, to.Address(), data, gas, price, value)
	if err != nil {
		return "", err
	}

	return toHex(res), nil
}

func (self *XEth) Transact(toStr, valueStr, gasStr, gasPriceStr, codeStr string) (string, error) {

	var (
		to               []byte
		value            = ethutil.NewValue(valueStr)
		gas              = ethutil.NewValue(gasStr)
		price            = ethutil.NewValue(gasPriceStr)
		data             []byte
		key              = self.eth.KeyManager().KeyPair()
		contractCreation bool
	)

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
	nonce := state.GetNonce(key.Address())

	tx.SetNonce(nonce)
	tx.Sign(key.PrivateKey)

	// Do some pre processing for our "pre" events  and hooks
	block := self.chainManager.NewBlock(key.Address())
	coinbase := state.GetOrNewStateObject(key.Address())
	coinbase.SetGasPool(block.GasLimit())
	self.blockProcessor.ApplyTransactions(coinbase, state, block, types.Transactions{tx}, true)

	err := self.eth.TxPool().Add(tx)
	if err != nil {
		return "", err
	}
	state.SetNonce(key.Address(), nonce+1)

	if contractCreation {
		addr := core.AddressFromMessage(tx)
		pipelogger.Infof("Contract addr %x\n", addr)
	}

	if types.IsContractAddr(to) {
		return toHex(core.AddressFromMessage(tx)), nil
	}

	return toHex(tx.Hash()), nil
}
