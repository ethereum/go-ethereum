package ethpipe

import (
	"encoding/json"
	"sync/atomic"

	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethutil"
)

type JSPipe struct {
	*Pipe
}

func NewJSPipe(eth ethchain.EthManager) *JSPipe {
	return &JSPipe{New(eth)}
}

func (self *JSPipe) GetBlockByHash(strHash string) *JSBlock {
	hash := ethutil.Hex2Bytes(strHash)
	block := self.obj.BlockChain().GetBlock(hash)

	return NewJSBlock(block)
}

func (self *JSPipe) GetKey() *JSKey {
	return NewJSKey(self.obj.KeyManager().KeyPair())
}

func (self *JSPipe) GetStateObject(addr string) *JSObject {
	object := &Object{self.World().safeGet(ethutil.Hex2Bytes(addr))}

	return NewJSObject(object)
}

func (self *JSPipe) GetPeerCount() int {
	return self.obj.PeerCount()
}

func (self *JSPipe) GetPeers() []JSPeer {
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

func (self *JSPipe) GetIsMining() bool {
	return self.obj.IsMining()
}

func (self *JSPipe) GetIsListening() bool {
	return self.obj.IsListening()
}

func (self *JSPipe) GetCoinBase() string {
	return ethutil.Bytes2Hex(self.obj.KeyManager().Address())
}

func (self *JSPipe) GetStorage(addr, storageAddr string) string {
	return self.World().SafeGet(ethutil.Hex2Bytes(addr)).Storage(ethutil.Hex2Bytes(storageAddr)).Str()
}

func (self *JSPipe) GetTxCountAt(address string) int {
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

func (self *JSPipe) GetEachStorage(addr string) string {
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

func (self *JSPipe) CompileMutan(code string) string {
	data, err := self.Pipe.CompileMutan(code)
	if err != nil {
		return err.Error()
	}

	return ethutil.Bytes2Hex(data)
}
