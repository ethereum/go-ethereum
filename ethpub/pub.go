package ethpub

import (
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
)

type PEthereum struct {
	stateManager *ethchain.StateManager
	blockChain   *ethchain.BlockChain
	txPool       *ethchain.TxPool
}

func NewPEthereum(eth *eth.Ethereum) *PEthereum {
	return &PEthereum{
		eth.StateManager(),
		eth.BlockChain(),
		eth.TxPool(),
	}
}

func (lib *PEthereum) GetBlock(hexHash string) *PBlock {
	hash := ethutil.FromHex(hexHash)

	block := lib.blockChain.GetBlock(hash)

	return &PBlock{Number: int(block.BlockInfo().Number), Hash: ethutil.Hex(block.Hash())}
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

func (lib *PEthereum) Transact(key, recipient, valueStr, gasStr, gasPriceStr, dataStr string) (string, error) {
	return lib.createTx(key, recipient, valueStr, gasStr, gasPriceStr, dataStr, "")
}

func (lib *PEthereum) Create(key, valueStr, gasStr, gasPriceStr, initStr, bodyStr string) (string, error) {
	return lib.createTx(key, "", valueStr, gasStr, gasPriceStr, initStr, bodyStr)
}

func (lib *PEthereum) createTx(key, recipient, valueStr, gasStr, gasPriceStr, initStr, scriptStr string) (string, error) {
	var hash []byte
	var contractCreation bool
	if len(recipient) == 0 {
		contractCreation = true
	} else {
		hash = ethutil.FromHex(recipient)
	}

	keyPair, err := ethchain.NewKeyPairFromSec([]byte(ethutil.FromHex(key)))
	if err != nil {
		return "", err
	}

	value := ethutil.Big(valueStr)
	gas := ethutil.Big(gasStr)
	gasPrice := ethutil.Big(gasPriceStr)
	var tx *ethchain.Transaction
	// Compile and assemble the given data
	if contractCreation {
		initScript, err := ethutil.Compile(initStr)
		if err != nil {
			return "", err
		}
		mainScript, err := ethutil.Compile(scriptStr)
		if err != nil {
			return "", err
		}

		tx = ethchain.NewContractCreationTx(value, gas, gasPrice, mainScript, initScript)
	} else {
		// Just in case it was submitted as a 0x prefixed string
		if initStr[0:2] == "0x" {
			initStr = initStr[2:len(initStr)]
		}
		tx = ethchain.NewTransactionMessage(hash, value, gas, gasPrice, ethutil.FromHex(initStr))
	}

	acc := lib.stateManager.GetAddrState(keyPair.Address())
	tx.Nonce = acc.Nonce
	tx.Sign(keyPair.PrivateKey)
	lib.txPool.QueueTransaction(tx)

	if contractCreation {
		ethutil.Config.Log.Infof("Contract addr %x", tx.Hash()[12:])
	} else {
		ethutil.Config.Log.Infof("Tx hash %x", tx.Hash())
	}

	return ethutil.Hex(tx.Hash()), nil
}
