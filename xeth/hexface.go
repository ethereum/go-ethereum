package xeth

import (
	"bytes"
	"encoding/json"
	"sync/atomic"

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
	hash := ethutil.Hex2Bytes(strHash)
	block := self.obj.ChainManager().GetBlock(hash)

	return NewJSBlock(block)
}

func (self *JSXEth) BlockByNumber(num int32) *JSBlock {
	if num == -1 {
		return NewJSBlock(self.obj.ChainManager().CurrentBlock)
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

func (self *JSXEth) StateObject(addr string) *JSObject {
	object := &Object{self.World().safeGet(ethutil.Hex2Bytes(addr))}

	return NewJSObject(object)
}

func (self *JSXEth) PeerCount() int {
	return self.obj.PeerCount()
}

func (self *JSXEth) Peers() []JSPeer {
	var peers []JSPeer
	for peer := self.obj.Peers().Front(); peer != nil; peer = peer.Next() {
		p := peer.Value.(core.Peer)
		// we only want connected peers
		if atomic.LoadInt32(p.Connected()) != 0 {
			peers = append(peers, *NewJSPeer(p))
		}
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
	return ethutil.Bytes2Hex(self.obj.KeyManager().Address())
}

func (self *JSXEth) NumberToHuman(balance string) string {
	b := ethutil.Big(balance)

	return ethutil.CurrencyToString(b)
}

func (self *JSXEth) StorageAt(addr, storageAddr string) string {
	storage := self.World().SafeGet(ethutil.Hex2Bytes(addr)).Storage(ethutil.Hex2Bytes(storageAddr))

	return ethutil.Bytes2Hex(storage.Bytes())
}

func (self *JSXEth) BalanceAt(addr string) string {
	return self.World().SafeGet(ethutil.Hex2Bytes(addr)).Balance().String()
}

func (self *JSXEth) TxCountAt(address string) int {
	return int(self.World().SafeGet(ethutil.Hex2Bytes(address)).Nonce)
}

func (self *JSXEth) CodeAt(address string) string {
	return ethutil.Bytes2Hex(self.World().SafeGet(ethutil.Hex2Bytes(address)).Code)
}

func (self *JSXEth) IsContract(address string) bool {
	return len(self.World().SafeGet(ethutil.Hex2Bytes(address)).Code) > 0
}

func (self *JSXEth) SecretToAddress(key string) string {
	pair, err := crypto.NewKeyPairFromSec(ethutil.Hex2Bytes(key))
	if err != nil {
		return ""
	}

	return ethutil.Bytes2Hex(pair.Address())
}

func (self *JSXEth) Execute(addr, value, gas, price, data string) (string, error) {
	ret, err := self.ExecuteObject(&Object{
		self.World().safeGet(ethutil.Hex2Bytes(addr))},
		ethutil.Hex2Bytes(data),
		ethutil.NewValue(value),
		ethutil.NewValue(gas),
		ethutil.NewValue(price),
	)

	return ethutil.Bytes2Hex(ret), err
}

type KeyVal struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (self *JSXEth) EachStorage(addr string) string {
	var values []KeyVal
	object := self.World().SafeGet(ethutil.Hex2Bytes(addr))
	object.EachStorage(func(name string, value *ethutil.Value) {
		value.Decode()
		values = append(values, KeyVal{ethutil.Bytes2Hex([]byte(name)), ethutil.Bytes2Hex(value.Bytes())})
	})

	valuesJson, err := json.Marshal(values)
	if err != nil {
		return ""
	}

	return string(valuesJson)
}

func (self *JSXEth) ToAscii(str string) string {
	padded := ethutil.RightPadBytes([]byte(str), 32)

	return "0x" + ethutil.Bytes2Hex(padded)
}

func (self *JSXEth) FromAscii(str string) string {
	if ethutil.IsHex(str) {
		str = str[2:]
	}

	return string(bytes.Trim(ethutil.Hex2Bytes(str), "\x00"))
}

func (self *JSXEth) FromNumber(str string) string {
	if ethutil.IsHex(str) {
		str = str[2:]
	}

	return ethutil.BigD(ethutil.Hex2Bytes(str)).String()
}

func (self *JSXEth) Transact(key, toStr, valueStr, gasStr, gasPriceStr, codeStr string) (string, error) {
	var (
		to       []byte
		value    = ethutil.NewValue(valueStr)
		gas      = ethutil.NewValue(gasStr)
		gasPrice = ethutil.NewValue(gasPriceStr)
		data     []byte
	)

	if ethutil.IsHex(codeStr) {
		data = ethutil.Hex2Bytes(codeStr[2:])
	} else {
		data = ethutil.Hex2Bytes(codeStr)
	}

	if ethutil.IsHex(toStr) {
		to = ethutil.Hex2Bytes(toStr[2:])
	} else {
		to = ethutil.Hex2Bytes(toStr)
	}

	var keyPair *crypto.KeyPair
	var err error
	if ethutil.IsHex(key) {
		keyPair, err = crypto.NewKeyPairFromSec([]byte(ethutil.Hex2Bytes(key[2:])))
	} else {
		keyPair, err = crypto.NewKeyPairFromSec([]byte(ethutil.Hex2Bytes(key)))
	}

	if err != nil {
		return "", err
	}

	tx, err := self.XEth.Transact(keyPair, to, value, gas, gasPrice, data)
	if err != nil {
		return "", err
	}
	if types.IsContractAddr(to) {
		return ethutil.Bytes2Hex(tx.CreationAddress(nil)), nil
	}

	return ethutil.Bytes2Hex(tx.Hash()), nil
}

func (self *JSXEth) PushTx(txStr string) (*JSReceipt, error) {
	tx := types.NewTransactionFromBytes(ethutil.Hex2Bytes(txStr))
	err := self.obj.TxPool().Add(tx)
	if err != nil {
		return nil, err
	}

	return NewJSReciept(tx.CreatesContract(), tx.CreationAddress(self.World().State()), tx.Hash(), tx.Sender()), nil
}

func (self *JSXEth) CompileMutan(code string) string {
	data, err := self.XEth.CompileMutan(code)
	if err != nil {
		return err.Error()
	}

	return ethutil.Bytes2Hex(data)
}

func (self *JSXEth) FindInConfig(str string) string {
	return ethutil.Bytes2Hex(self.World().Config().Get(str).Address())
}

func ToJSMessages(messages state.Messages) *ethutil.List {
	var msgs []JSMessage
	for _, m := range messages {
		msgs = append(msgs, NewJSMessage(m))
	}

	return ethutil.NewList(msgs)
}
