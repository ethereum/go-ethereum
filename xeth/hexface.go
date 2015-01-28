package xeth

import (
	"bytes"
	"encoding/json"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/state"
)

// to resolve the import cycle
type Backend interface {
	BlockProcessor() *core.BlockProcessor
	ChainManager() *core.ChainManager
	Coinbase() []byte
	KeyManager() *crypto.KeyManager
	IsMining() bool
	IsListening() bool
	PeerCount() int
	Db() ethutil.Database
	TxPool() *core.TxPool
}

type JSXEth struct {
	eth            Backend
	blockProcessor *core.BlockProcessor
	chainManager   *core.ChainManager
	world          *State
}

func NewJSXEth(eth Backend) *JSXEth {
	xeth := &JSXEth{
		eth:            eth,
		blockProcessor: eth.BlockProcessor(),
		chainManager:   eth.ChainManager(),
	}
	xeth.world = NewState(xeth)

	return xeth
}

func (self *JSXEth) State() *State { return self.world }

func (self *JSXEth) BlockByHash(strHash string) *JSBlock {
	hash := fromHex(strHash)
	block := self.chainManager.GetBlock(hash)

	return NewJSBlock(block)
}

func (self *JSXEth) BlockByNumber(num int32) *JSBlock {
	if num == -1 {
		return NewJSBlock(self.chainManager.CurrentBlock())
	}

	return NewJSBlock(self.chainManager.GetBlockByNumber(uint64(num)))
}

func (self *JSXEth) Block(v interface{}) *JSBlock {
	if n, ok := v.(int32); ok {
		return self.BlockByNumber(n)
	} else if str, ok := v.(string); ok {
		return self.BlockByHash(str)
	} else if f, ok := v.(float64); ok { // Don't ask ...
		return self.BlockByNumber(int32(f))
	}

	return nil
}

func (self *JSXEth) Accounts() []string {
	return []string{toHex(self.eth.KeyManager().Address())}
}

/*
func (self *JSXEth) StateObject(addr string) *JSObject {
	object := &Object{self.State().safeGet(fromHex(addr))}

	return NewJSObject(object)
}
*/

func (self *JSXEth) PeerCount() int {
	return self.eth.PeerCount()
}

func (self *JSXEth) IsMining() bool {
	return self.eth.IsMining()
}

func (self *JSXEth) IsListening() bool {
	return self.eth.IsListening()
}

func (self *JSXEth) Coinbase() string {
	return toHex(self.eth.KeyManager().Address())
}

func (self *JSXEth) NumberToHuman(balance string) string {
	b := ethutil.Big(balance)

	return ethutil.CurrencyToString(b)
}

func (self *JSXEth) StorageAt(addr, storageAddr string) string {
	storage := self.State().SafeGet(addr).StorageString(storageAddr)

	return toHex(storage.Bytes())
}

func (self *JSXEth) BalanceAt(addr string) string {
	return self.State().SafeGet(addr).Balance().String()
}

func (self *JSXEth) TxCountAt(address string) int {
	return int(self.State().SafeGet(address).Nonce)
}

func (self *JSXEth) CodeAt(address string) string {
	return toHex(self.State().SafeGet(address).Code)
}

func (self *JSXEth) IsContract(address string) bool {
	return len(self.State().SafeGet(address).Code) > 0
}

func (self *JSXEth) SecretToAddress(key string) string {
	pair, err := crypto.NewKeyPairFromSec(fromHex(key))
	if err != nil {
		return ""
	}

	return toHex(pair.Address())
}

func (self *JSXEth) Execute(addr, value, gas, price, data string) (string, error) {
	return "", nil
}

type KeyVal struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (self *JSXEth) EachStorage(addr string) string {
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

func (self *JSXEth) ToAscii(str string) string {
	padded := ethutil.RightPadBytes([]byte(str), 32)

	return "0x" + toHex(padded)
}

func (self *JSXEth) FromAscii(str string) string {
	if ethutil.IsHex(str) {
		str = str[2:]
	}

	return string(bytes.Trim(fromHex(str), "\x00"))
}

func (self *JSXEth) FromNumber(str string) string {
	if ethutil.IsHex(str) {
		str = str[2:]
	}

	return ethutil.BigD(fromHex(str)).String()
}

func (self *JSXEth) Transact(key, toStr, valueStr, gasStr, gasPriceStr, codeStr string) (string, error) {
	return "", nil
}

func ToJSMessages(messages state.Messages) *ethutil.List {
	var msgs []JSMessage
	for _, m := range messages {
		msgs = append(msgs, NewJSMessage(m))
	}

	return ethutil.NewList(msgs)
}

func (self *JSXEth) PushTx(encodedTx string) (string, error) {
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
