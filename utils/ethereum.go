package utils

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

func (lib *PEthereum) GetKey() string {
	return ethutil.Hex(ethutil.Config.Db.GetKeys()[0].Address())
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

	keyPair, err := ethchain.NewKeyPairFromSec([]byte(key))
	if err != nil {
		return "", err
	}

	value := ethutil.Big(valueStr)
	gas := ethutil.Big(gasStr)
	gasPrice := ethutil.Big(gasPriceStr)
	var tx *ethchain.Transaction
	// Compile and assemble the given data
	if contractCreation {
		initScript, err := Compile(initStr)
		if err != nil {
			return "", err
		}
		mainScript, err := Compile(scriptStr)
		if err != nil {
			return "", err
		}

		tx = ethchain.NewContractCreationTx(value, gas, gasPrice, mainScript, initScript)
	} else {
		/*
			lines := strings.Split(dataStr, "\n")
			var data []byte
			for _, line := range lines {
				data = append(data, ethutil.BigToBytes(ethutil.Big(line), 256)...)
			}
		*/

		tx = ethchain.NewTransactionMessage(hash, value, gas, gasPrice, []byte(initStr))
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
