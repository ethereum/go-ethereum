package ethpub

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
)

type PEthereum struct {
	manager      ethchain.EthManager
	stateManager *ethchain.StateManager
	blockChain   *ethchain.BlockChain
	txPool       *ethchain.TxPool
}

func NewPEthereum(manager ethchain.EthManager) *PEthereum {
	return &PEthereum{
		manager,
		manager.StateManager(),
		manager.BlockChain(),
		manager.TxPool(),
	}
}

func (lib *PEthereum) GetBlock(hexHash string) *PBlock {
	hash := ethutil.FromHex(hexHash)

	block := lib.blockChain.GetBlock(hash)

	var blockInfo *PBlock

	if block != nil {
		blockInfo = &PBlock{Number: int(block.BlockInfo().Number), Hash: ethutil.Hex(block.Hash())}
	} else {
		blockInfo = &PBlock{Number: -1, Hash: ""}
	}

	return blockInfo
}

func (lib *PEthereum) GetKey() *PKey {
	keyPair, err := ethchain.NewKeyPairFromSec(ethutil.Config.Db.GetKeys()[0].PrivateKey)
	if err != nil {
		return nil
	}

	return NewPKey(keyPair)
}

func (lib *PEthereum) GetStateObject(address string) *PStateObject {
	stateObject := lib.stateManager.ProcState().GetContract(ethutil.FromHex(address))
	if stateObject != nil {
		return NewPStateObject(stateObject)
	}

	// See GetStorage for explanation on "nil"
	return NewPStateObject(nil)
}

func (lib *PEthereum) GetPeerCount() int {
	return lib.manager.PeerCount()
}

func (lib *PEthereum) GetIsMining() bool {
	return lib.manager.IsMining()
}

func (lib *PEthereum) GetIsListening() bool {
	return lib.manager.IsListening()
}

func (lib *PEthereum) GetCoinBase() string {
	data, _ := ethutil.Config.Db.Get([]byte("KeyRing"))
	keyRing := ethutil.NewValueFromBytes(data)
	key := keyRing.Get(0).Bytes()

	return lib.SecretToAddress(hex.EncodeToString(key))
}

func (lib *PEthereum) GetStorage(address, storageAddress string) string {
	return lib.GetStateObject(address).GetStorage(storageAddress)
}

func (lib *PEthereum) GetTxCountAt(address string) int {
	fmt.Println("GO")
	return lib.GetStateObject(address).Nonce()
}

func (lib *PEthereum) IsContract(address string) bool {
	return lib.GetStateObject(address).IsContract()
}

func (lib *PEthereum) SecretToAddress(key string) string {
	pair, err := ethchain.NewKeyPairFromSec(ethutil.FromHex(key))
	if err != nil {
		return ""
	}

	return ethutil.Hex(pair.Address())
}

func (lib *PEthereum) Transact(key, recipient, valueStr, gasStr, gasPriceStr, dataStr string) (*PReceipt, error) {
	return lib.createTx(key, recipient, valueStr, gasStr, gasPriceStr, dataStr, "")
}

func (lib *PEthereum) Create(key, valueStr, gasStr, gasPriceStr, initStr, bodyStr string) (*PReceipt, error) {
	return lib.createTx(key, "", valueStr, gasStr, gasPriceStr, initStr, bodyStr)
}

func (lib *PEthereum) createTx(key, recipient, valueStr, gasStr, gasPriceStr, initStr, scriptStr string) (*PReceipt, error) {
	var hash []byte
	var contractCreation bool
	if len(recipient) == 0 {
		contractCreation = true
	} else {
		hash = ethutil.FromHex(recipient)
	}

	var keyPair *ethchain.KeyPair
	var err error
	if key[0:2] == "0x" {
		keyPair, err = ethchain.NewKeyPairFromSec([]byte(ethutil.FromHex(key[0:2])))
	} else {
		keyPair, err = ethchain.NewKeyPairFromSec([]byte(ethutil.FromHex(key)))
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
		var initScript, mainScript []byte
		var err error
		if ethutil.IsHex(initStr) {
			initScript = ethutil.FromHex(initStr[2:])
		} else {
			initScript, err = ethutil.Compile(initStr)
			if err != nil {
				return nil, err
			}
		}

		if ethutil.IsHex(scriptStr) {
			mainScript = ethutil.FromHex(scriptStr[2:])
		} else {
			mainScript, err = ethutil.Compile(scriptStr)
			if err != nil {
				return nil, err
			}
		}

		tx = ethchain.NewContractCreationTx(value, gas, gasPrice, mainScript, initScript)
	} else {
		// Just in case it was submitted as a 0x prefixed string
		if len(initStr) > 0 && initStr[0:2] == "0x" {
			initStr = initStr[2:len(initStr)]
		}
		tx = ethchain.NewTransactionMessage(hash, value, gas, gasPrice, ethutil.FromHex(initStr))
	}

	acc := lib.stateManager.TransState().GetStateObject(keyPair.Address())
	//acc := lib.stateManager.GetAddrState(keyPair.Address())
	tx.Nonce = acc.Nonce
	lib.stateManager.TransState().SetStateObject(acc)

	tx.Sign(keyPair.PrivateKey)
	lib.txPool.QueueTransaction(tx)

	if contractCreation {
		ethutil.Config.Log.Infof("Contract addr %x", tx.CreationAddress())
	} else {
		ethutil.Config.Log.Infof("Tx hash %x", tx.Hash())
	}

	return NewPReciept(contractCreation, tx.CreationAddress(), tx.Hash(), keyPair.Address()), nil
}
