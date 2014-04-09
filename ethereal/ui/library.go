package ethui

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/obscuren/mutan"
	"strings"
)

type EthLib struct {
	stateManager *ethchain.StateManager
	blockChain   *ethchain.BlockChain
	txPool       *ethchain.TxPool
}

func (lib *EthLib) CreateTx(recipient, valueStr, gasStr, gasPriceStr, data string) (string, error) {
	var hash []byte
	var contractCreation bool
	if len(recipient) == 0 {
		contractCreation = true
	} else {
		var err error
		hash, err = hex.DecodeString(recipient)
		if err != nil {
			return "", err
		}
	}

	keyPair := ethutil.Config.Db.GetKeys()[0]
	value := ethutil.Big(valueStr)
	gas := ethutil.Big(gasStr)
	gasPrice := ethutil.Big(gasPriceStr)
	var tx *ethchain.Transaction
	// Compile and assemble the given data
	if contractCreation {
		asm, err := mutan.Compile(strings.NewReader(data), false)
		if err != nil {
			return "", err
		}

		code := ethutil.Assemble(asm...)
		tx = ethchain.NewContractCreationTx(value, gasPrice, code)
	} else {
		tx = ethchain.NewTransactionMessage(hash, value, gasPrice, gas, []string{})
	}
	acc := lib.stateManager.GetAddrState(keyPair.Address())
	tx.Nonce = acc.Nonce
	//acc.Nonce++
	tx.Sign(keyPair.PrivateKey)
	lib.txPool.QueueTransaction(tx)

	if contractCreation {
		ethutil.Config.Log.Infof("Contract addr %x", tx.Hash()[12:])
	} else {
		ethutil.Config.Log.Infof("Tx hash %x", tx.Hash())
	}

	return ethutil.Hex(tx.Hash()), nil
}

/*
func (lib *EthLib) CreateTx(receiver, a, data string) string {
	var hash []byte
	if len(receiver) == 0 {
		hash = ethchain.ContractAddr
	} else {
		var err error
		hash, err = hex.DecodeString(receiver)
		if err != nil {
			return err.Error()
		}
	}

	k, _ := ethutil.Config.Db.Get([]byte("KeyRing"))
	keyPair := ethutil.NewKeyFromBytes(k)

	amount := ethutil.Big(a)
	code := ethchain.Compile(strings.Split(data, "\n"))
	tx := ethchain.NewTx(hash, amount, code)
	tx.Nonce = lib.stateManager.GetAddrState(keyPair.Address()).Nonce

	tx.Sign(keyPair.PrivateKey)

	lib.txPool.QueueTransaction(tx)

	if len(receiver) == 0 {
		ethutil.Config.Log.Infof("Contract addr %x", tx.Hash()[12:])
	} else {
		ethutil.Config.Log.Infof("Tx hash %x", tx.Hash())
	}

	return ethutil.Hex(tx.Hash())
}
*/

func (lib *EthLib) GetBlock(hexHash string) *Block {
	hash, err := hex.DecodeString(hexHash)
	if err != nil {
		return nil
	}

	block := lib.blockChain.GetBlock(hash)
	fmt.Println(block)

	return &Block{Number: int(block.BlockInfo().Number), Hash: ethutil.Hex(block.Hash())}
}
