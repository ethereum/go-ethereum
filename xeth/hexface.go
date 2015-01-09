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

type JSXEth struct {
	*XEth
}

func NewJSXEth(eth core.EthManager) *JSXEth {
	return &JSXEth{New(eth)}
}

func (self *JSXEth) BlockByHash(strHash string) *JSBlock {
	hash := fromHex(strHash)
	block := self.obj.ChainManager().GetBlock(hash)

	return NewJSBlock(block)
}

func (self *JSXEth) BlockByNumber(num int32) *JSBlock {
	if num == -1 {
		return NewJSBlock(self.obj.ChainManager().CurrentBlock())
	}

	return NewJSBlock(self.obj.ChainManager().GetBlockByNumber(uint64(num)))
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

func (self *JSXEth) Key() *JSKey {
	return NewJSKey(self.obj.KeyManager().KeyPair())
}

func (self *JSXEth) Accounts() []string {
	return []string{toHex(self.obj.KeyManager().Address())}
}

func (self *JSXEth) StateObject(addr string) *JSObject {
	object := &Object{self.World().safeGet(fromHex(addr))}

	return NewJSObject(object)
}

func (self *JSXEth) PeerCount() int {
	return self.obj.PeerCount()
}

func (self *JSXEth) Peers() []JSPeer {
	var peers []JSPeer
	for _, peer := range self.obj.Peers() {
		peers = append(peers, *NewJSPeer(peer))
	}

	return peers
}

func (self *JSXEth) IsMining() bool {
	return self.obj.IsMining()
}

func (self *JSXEth) IsListening() bool {
	return self.obj.IsListening()
}

func (self *JSXEth) CoinBase() string {
	return toHex(self.obj.KeyManager().Address())
}

func (self *JSXEth) NumberToHuman(balance string) string {
	b := ethutil.Big(balance)

	return ethutil.CurrencyToString(b)
}

func (self *JSXEth) StorageAt(addr, storageAddr string) string {
	storage := self.World().SafeGet(fromHex(addr)).Storage(fromHex(storageAddr))

	return toHex(storage.Bytes())
}

func (self *JSXEth) BalanceAt(addr string) string {
	return self.World().SafeGet(fromHex(addr)).Balance().String()
}

func (self *JSXEth) TxCountAt(address string) int {
	return int(self.World().SafeGet(fromHex(address)).Nonce)
}

func (self *JSXEth) CodeAt(address string) string {
	return toHex(self.World().SafeGet(fromHex(address)).Code)
}

func (self *JSXEth) IsContract(address string) bool {
	return len(self.World().SafeGet(fromHex(address)).Code) > 0
}

func (self *JSXEth) SecretToAddress(key string) string {
	pair, err := crypto.NewKeyPairFromSec(fromHex(key))
	if err != nil {
		return ""
	}

	return toHex(pair.Address())
}

func (self *JSXEth) Execute(addr, value, gas, price, data string) (string, error) {
	ret, err := self.ExecuteObject(&Object{
		self.World().safeGet(fromHex(addr))},
		fromHex(data),
		ethutil.NewValue(value),
		ethutil.NewValue(gas),
		ethutil.NewValue(price),
	)

	return toHex(ret), err
}

type KeyVal struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (self *JSXEth) EachStorage(addr string) string {
	var values []KeyVal
	object := self.World().SafeGet(fromHex(addr))
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
	var (
		to       []byte
		value    = ethutil.NewValue(valueStr)
		gas      = ethutil.NewValue(gasStr)
		gasPrice = ethutil.NewValue(gasPriceStr)
		data     []byte
	)

	data = fromHex(codeStr)

	to = fromHex(toStr)

	keyPair, err := crypto.NewKeyPairFromSec([]byte(fromHex(key)))
	if err != nil {
		return "", err
	}

	tx, err := self.XEth.Transact(keyPair, to, value, gas, gasPrice, data)
	if err != nil {
		return "", err
	}
	if types.IsContractAddr(to) {
		return toHex(core.AddressFromMessage(tx)), nil
	}

	return toHex(tx.Hash()), nil
}

func (self *JSXEth) PushTx(txStr string) (*JSReceipt, error) {
	tx := types.NewTransactionFromBytes(fromHex(txStr))
	err := self.obj.TxPool().Add(tx)
	if err != nil {
		return nil, err
	}

	return NewJSReciept(core.MessageCreatesContract(tx), core.AddressFromMessage(tx), tx.Hash(), tx.From()), nil
}

func (self *JSXEth) CompileMutan(code string) string {
	data, err := self.XEth.CompileMutan(code)
	if err != nil {
		return err.Error()
	}

	return toHex(data)
}

func (self *JSXEth) FindInConfig(str string) string {
	return toHex(self.World().Config().Get(str).Address())
}

func ToJSMessages(messages state.Messages) *ethutil.List {
	var msgs []JSMessage
	for _, m := range messages {
		msgs = append(msgs, NewJSMessage(m))
	}

	return ethutil.NewList(msgs)
}
