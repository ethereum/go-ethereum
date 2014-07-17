package ethpub

import (
	"bytes"
	"encoding/json"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
	"strings"
	"sync/atomic"
)

var logger = ethlog.NewLogger("PUB")

// TODO this has to move elsewhere
var cnfCtr = ethutil.Hex2Bytes("661005d2720d855f1d9976f88bb10c1a3398c77f")

type helper struct {
	sm *ethchain.StateManager
}

func EthereumConfig(stateManager *ethchain.StateManager) helper {
	return helper{stateManager}
}
func (self helper) obj() *ethchain.StateObject {
	return self.sm.CurrentState().GetStateObject(cnfCtr)
}

func (self helper) NameReg() *ethchain.StateObject {
	if self.obj() != nil {
		addr := self.obj().GetStorage(big.NewInt(0))
		if len(addr.Bytes()) > 0 {
			return self.sm.CurrentState().GetStateObject(addr.Bytes())
		}
	}

	return nil
}

type PEthereum struct {
	manager      ethchain.EthManager
	stateManager *ethchain.StateManager
	blockChain   *ethchain.BlockChain
	txPool       *ethchain.TxPool
	keyManager   *ethcrypto.KeyManager
}

func NewPEthereum(manager ethchain.EthManager) *PEthereum {
	return &PEthereum{
		manager,
		manager.StateManager(),
		manager.BlockChain(),
		manager.TxPool(),
		manager.KeyManager(),
	}
}

func (lib *PEthereum) GetBlock(hexHash string) *PBlock {
	hash := ethutil.Hex2Bytes(hexHash)
	block := lib.blockChain.GetBlock(hash)

	return NewPBlock(block)
}

func (lib *PEthereum) GetKey() *PKey {
	return NewPKey(lib.keyManager.KeyPair())
}

func (lib *PEthereum) GetStateObject(address string) *PStateObject {
	stateObject := lib.stateManager.CurrentState().GetStateObject(ethutil.Hex2Bytes(address))
	if stateObject != nil {
		return NewPStateObject(stateObject)
	}

	// See GetStorage for explanation on "nil"
	return NewPStateObject(nil)
}

func (lib *PEthereum) GetPeerCount() int {
	return lib.manager.PeerCount()
}

func (lib *PEthereum) GetPeers() []PPeer {
	var peers []PPeer
	for peer := lib.manager.Peers().Front(); peer != nil; peer = peer.Next() {
		p := peer.Value.(ethchain.Peer)
		// we only want connected peers
		if atomic.LoadInt32(p.Connected()) != 0 {
			peers = append(peers, *NewPPeer(p))
		}
	}

	return peers
}

func (lib *PEthereum) GetIsMining() bool {
	return lib.manager.IsMining()
}

func (lib *PEthereum) GetIsListening() bool {
	return lib.manager.IsListening()
}

func (lib *PEthereum) GetCoinBase() string {
	return ethutil.Bytes2Hex(lib.keyManager.Address())
}

func (lib *PEthereum) GetTransactionsFor(address string, asJson bool) interface{} {
	sBlk := lib.manager.BlockChain().LastBlockHash
	blk := lib.manager.BlockChain().GetBlock(sBlk)
	addr := []byte(ethutil.Hex2Bytes(address))

	var txs []*PTx

	for ; blk != nil; blk = lib.manager.BlockChain().GetBlock(sBlk) {
		sBlk = blk.PrevHash

		// Loop through all transactions to see if we missed any while being offline
		for _, tx := range blk.Transactions() {
			if bytes.Compare(tx.Sender(), addr) == 0 || bytes.Compare(tx.Recipient, addr) == 0 {
				ptx := NewPTx(tx)
				//TODO: somehow move this to NewPTx
				ptx.Confirmations = int(lib.manager.BlockChain().LastBlockNumber - blk.BlockInfo().Number)
				txs = append(txs, ptx)
			}
		}
	}
	if asJson {
		txJson, err := json.Marshal(txs)
		if err != nil {
			return nil
		}
		return string(txJson)
	}
	return txs
}

func (lib *PEthereum) GetStorage(address, storageAddress string) string {
	return lib.GetStateObject(address).GetStorage(storageAddress)
}

func (lib *PEthereum) GetTxCountAt(address string) int {
	return lib.GetStateObject(address).Nonce()
}

func (lib *PEthereum) IsContract(address string) bool {
	return lib.GetStateObject(address).IsContract()
}

func (lib *PEthereum) SecretToAddress(key string) string {
	pair, err := ethcrypto.NewKeyPairFromSec(ethutil.Hex2Bytes(key))
	if err != nil {
		return ""
	}

	return ethutil.Bytes2Hex(pair.Address())
}

func (lib *PEthereum) Transact(key, recipient, valueStr, gasStr, gasPriceStr, dataStr string) (*PReceipt, error) {
	return lib.createTx(key, recipient, valueStr, gasStr, gasPriceStr, dataStr)
}

func (lib *PEthereum) Create(key, valueStr, gasStr, gasPriceStr, script string) (*PReceipt, error) {
	return lib.createTx(key, "", valueStr, gasStr, gasPriceStr, script)
}

func FindAddressInNameReg(stateManager *ethchain.StateManager, name string) []byte {
	nameReg := EthereumConfig(stateManager).NameReg()
	if nameReg != nil {
		addr := ethutil.RightPadBytes([]byte(name), 32)

		reg := nameReg.GetStorage(ethutil.BigD(addr))

		return reg.Bytes()
	}

	return nil
}

func FindNameInNameReg(stateManager *ethchain.StateManager, addr []byte) string {
	nameReg := EthereumConfig(stateManager).NameReg()
	if nameReg != nil {
		addr = ethutil.LeftPadBytes(addr, 32)

		reg := nameReg.GetStorage(ethutil.BigD(addr))

		return strings.TrimRight(reg.Str(), "\x00")
	}

	return ""
}

func (lib *PEthereum) createTx(key, recipient, valueStr, gasStr, gasPriceStr, scriptStr string) (*PReceipt, error) {
	var hash []byte
	var contractCreation bool
	if len(recipient) == 0 {
		contractCreation = true
	} else {
		// Check if an address is stored by this address
		addr := FindAddressInNameReg(lib.stateManager, recipient)
		if len(addr) > 0 {
			hash = addr
		} else {
			hash = ethutil.Hex2Bytes(recipient)
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

	value := ethutil.Big(valueStr)
	gas := ethutil.Big(gasStr)
	gasPrice := ethutil.Big(gasPriceStr)
	var tx *ethchain.Transaction
	// Compile and assemble the given data
	if contractCreation {
		var script []byte
		var err error
		if ethutil.IsHex(scriptStr) {
			script = ethutil.Hex2Bytes(scriptStr[2:])
		} else {
			script, err = ethutil.Compile(scriptStr, false)
			if err != nil {
				return nil, err
			}
		}

		tx = ethchain.NewContractCreationTx(value, gas, gasPrice, script)
	} else {
		data := ethutil.StringToByteFunc(scriptStr, func(s string) (ret []byte) {
			slice := strings.Split(s, "\n")
			for _, dataItem := range slice {
				d := ethutil.FormatData(dataItem)
				ret = append(ret, d...)
			}
			return
		})

		tx = ethchain.NewTransactionMessage(hash, value, gas, gasPrice, data)
	}

	acc := lib.stateManager.TransState().GetOrNewStateObject(keyPair.Address())
	tx.Nonce = acc.Nonce
	acc.Nonce += 1
	lib.stateManager.TransState().UpdateStateObject(acc)

	tx.Sign(keyPair.PrivateKey)
	lib.txPool.QueueTransaction(tx)

	if contractCreation {
		logger.Infof("Contract addr %x", tx.CreationAddress())
	}

	return NewPReciept(contractCreation, tx.CreationAddress(), tx.Hash(), keyPair.Address()), nil
}
