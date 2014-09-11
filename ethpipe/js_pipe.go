package ethpipe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethreact"
	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethutil"
)

type JSPipe struct {
	*Pipe
}

func NewJSPipe(eth ethchain.EthManager) *JSPipe {
	return &JSPipe{New(eth)}
}

func (self *JSPipe) BlockByHash(strHash string) *JSBlock {
	hash := ethutil.Hex2Bytes(strHash)
	block := self.obj.BlockChain().GetBlock(hash)

	return NewJSBlock(block)
}

func (self *JSPipe) BlockByNumber(num int32) *JSBlock {
	if num == -1 {
		return NewJSBlock(self.obj.BlockChain().CurrentBlock)
	}

	return NewJSBlock(self.obj.BlockChain().GetBlockByNumber(uint64(num)))
}

func (self *JSPipe) Block(v interface{}) *JSBlock {
	if n, ok := v.(int32); ok {
		return self.BlockByNumber(n)
	} else if str, ok := v.(string); ok {
		return self.BlockByHash(str)
	} else if f, ok := v.(float64); ok { // Don't ask ...
		return self.BlockByNumber(int32(f))
	}

	return nil
}

func (self *JSPipe) Key() *JSKey {
	return NewJSKey(self.obj.KeyManager().KeyPair())
}

func (self *JSPipe) StateObject(addr string) *JSObject {
	object := &Object{self.World().safeGet(ethutil.Hex2Bytes(addr))}

	return NewJSObject(object)
}

func (self *JSPipe) PeerCount() int {
	return self.obj.PeerCount()
}

func (self *JSPipe) Peers() []JSPeer {
	var peers []JSPeer
	for peer := self.obj.Peers().Front(); peer != nil; peer = peer.Next() {
		p := peer.Value.(ethchain.Peer)
		// we only want connected peers
		if atomic.LoadInt32(p.Connected()) != 0 {
			peers = append(peers, *NewJSPeer(p))
		}
	}

	return peers
}

func (self *JSPipe) IsMining() bool {
	return self.obj.IsMining()
}

func (self *JSPipe) IsListening() bool {
	return self.obj.IsListening()
}

func (self *JSPipe) CoinBase() string {
	return ethutil.Bytes2Hex(self.obj.KeyManager().Address())
}

func (self *JSPipe) BalanceAt(addr string) string {
	return self.World().SafeGet(ethutil.Hex2Bytes(addr)).Balance.String()
}

func (self *JSPipe) NumberToHuman(balance string) string {
	b := ethutil.Big(balance)

	return ethutil.CurrencyToString(b)
}

func (self *JSPipe) StorageAt(addr, storageAddr string) string {
	storage := self.World().SafeGet(ethutil.Hex2Bytes(addr)).Storage(ethutil.Hex2Bytes(storageAddr))

	return ethutil.Bytes2Hex(storage.Bytes())
}

func (self *JSPipe) TxCountAt(address string) int {
	return int(self.World().SafeGet(ethutil.Hex2Bytes(address)).Nonce)
}

func (self *JSPipe) IsContract(address string) bool {
	return len(self.World().SafeGet(ethutil.Hex2Bytes(address)).Code) > 0
}

func (self *JSPipe) SecretToAddress(key string) string {
	pair, err := ethcrypto.NewKeyPairFromSec(ethutil.Hex2Bytes(key))
	if err != nil {
		return ""
	}

	return ethutil.Bytes2Hex(pair.Address())
}

type KeyVal struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (self *JSPipe) EachStorage(addr string) string {
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

func (self *JSPipe) ToAscii(str string) string {
	padded := ethutil.RightPadBytes([]byte(str), 32)

	return "0x" + ethutil.Bytes2Hex(padded)
}

func (self *JSPipe) FromAscii(str string) string {
	if ethutil.IsHex(str) {
		str = str[2:]
	}

	return string(bytes.Trim(ethutil.Hex2Bytes(str), "\x00"))
}

func (self *JSPipe) FromNumber(str string) string {
	if ethutil.IsHex(str) {
		str = str[2:]
	}

	return ethutil.BigD(ethutil.Hex2Bytes(str)).String()
}

func (self *JSPipe) Transact(key, toStr, valueStr, gasStr, gasPriceStr, codeStr string) (*JSReceipt, error) {
	var hash []byte
	var contractCreation bool
	if len(toStr) == 0 {
		contractCreation = true
	} else {
		// Check if an address is stored by this address
		addr := self.World().Config().Get("NameReg").StorageString(toStr).Bytes()
		if len(addr) > 0 {
			hash = addr
		} else {
			hash = ethutil.Hex2Bytes(toStr)
		}
	}

	var keyPair *ethcrypto.KeyPair
	var err error
	if ethutil.IsHex(key) {
		keyPair, err = ethcrypto.NewKeyPairFromSec([]byte(ethutil.Hex2Bytes(key[2:])))
	} else {
		keyPair, err = ethcrypto.NewKeyPairFromSec([]byte(ethutil.Hex2Bytes(key)))
	}

	if err != nil {
		return nil, err
	}

	var (
		value    = ethutil.Big(valueStr)
		gas      = ethutil.Big(gasStr)
		gasPrice = ethutil.Big(gasPriceStr)
		data     []byte
		tx       *ethchain.Transaction
	)

	if ethutil.IsHex(codeStr) {
		data = ethutil.Hex2Bytes(codeStr[2:])
	} else {
		data = ethutil.Hex2Bytes(codeStr)
	}

	if contractCreation {
		tx = ethchain.NewContractCreationTx(value, gas, gasPrice, data)
	} else {
		tx = ethchain.NewTransactionMessage(hash, value, gas, gasPrice, data)
	}

	acc := self.obj.StateManager().TransState().GetOrNewStateObject(keyPair.Address())
	tx.Nonce = acc.Nonce
	acc.Nonce += 1
	self.obj.StateManager().TransState().UpdateStateObject(acc)

	tx.Sign(keyPair.PrivateKey)
	self.obj.TxPool().QueueTransaction(tx)

	if contractCreation {
		logger.Infof("Contract addr %x", tx.CreationAddress())
	}

	return NewJSReciept(contractCreation, tx.CreationAddress(), tx.Hash(), keyPair.Address()), nil
}

func (self *JSPipe) PushTx(txStr string) (*JSReceipt, error) {
    tx := ethchain.NewTransactionFromBytes(ethutil.Hex2Bytes(txStr))
    self.obj.TxPool().QueueTransaction(tx)
    return NewJSReciept(tx.CreatesContract(), tx.CreationAddress(), tx.Hash(), tx.Sender()), nil
}

func (self *JSPipe) CompileMutan(code string) string {
	data, err := self.Pipe.CompileMutan(code)
	if err != nil {
		return err.Error()
	}

	return ethutil.Bytes2Hex(data)
}

func (self *JSPipe) Watch(object map[string]interface{}) *JSFilter {
	return NewJSFilterFromMap(object, self.Pipe.obj)
	/*} else if str, ok := object.(string); ok {
	println("str")
	return NewJSFilterFromString(str, self.Pipe.obj)
	*/
}

func (self *JSPipe) Messages(object map[string]interface{}) string {
	filter := self.Watch(object)
	filter.Uninstall()

	return filter.Messages()

}

type JSFilter struct {
	eth ethchain.EthManager
	*ethchain.Filter
	quit chan bool

	BlockCallback   func(*ethchain.Block)
	MessageCallback func(ethstate.Messages)
}

func NewJSFilterFromMap(object map[string]interface{}, eth ethchain.EthManager) *JSFilter {
	filter := &JSFilter{eth, ethchain.NewFilterFromMap(object, eth), make(chan bool), nil, nil}

	go filter.mainLoop()

	return filter
}

func NewJSFilterFromString(str string, eth ethchain.EthManager) *JSFilter {
	return nil
}

func (self *JSFilter) MessagesToJson(messages ethstate.Messages) string {
	var msgs []JSMessage
	for _, m := range messages {
		msgs = append(msgs, NewJSMessage(m))
	}

	// Return an empty array instead of "null"
	if len(msgs) == 0 {
		return "[]"
	}

	b, err := json.Marshal(msgs)
	if err != nil {
		return "{\"error\":" + err.Error() + "}"
	}

	return string(b)
}

func (self *JSFilter) Messages() string {
	return self.MessagesToJson(self.Find())
}

func (self *JSFilter) mainLoop() {
	blockChan := make(chan ethreact.Event, 5)
	messageChan := make(chan ethreact.Event, 5)
	// Subscribe to events
	reactor := self.eth.Reactor()
	reactor.Subscribe("newBlock", blockChan)
	reactor.Subscribe("messages", messageChan)
out:
	for {
		select {
		case <-self.quit:
			break out
		case block := <-blockChan:
			if block, ok := block.Resource.(*ethchain.Block); ok {
				if self.BlockCallback != nil {
					self.BlockCallback(block)
				}
			}
		case msg := <-messageChan:
			if messages, ok := msg.Resource.(ethstate.Messages); ok {
				if self.MessageCallback != nil {
					println("messages!")
					msgs := self.FilterMessages(messages)
					if len(msgs) > 0 {
						self.MessageCallback(msgs)
					}
				}
			}
		}
	}
}

func (self *JSFilter) Changed(object interface{}) {
	fmt.Printf("%T\n", object)
}

func (self *JSFilter) Uninstall() {
	self.quit <- true
}
