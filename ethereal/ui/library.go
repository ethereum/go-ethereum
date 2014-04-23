package ethui

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/go-ethereum/utils"
	"github.com/obscuren/secp256k1-go"
	"strings"
)

type Contract struct {
	object *ethchain.StateObject
}

func NewContract(object *ethchain.StateObject) *Contract {
	return &Contract{object: object}
}

func (c *Contract) GetStorage(address string) string {
	val := c.object.GetMem(ethutil.Big("0x" + address))

	return val.BigInt().String()
}

type EthLib struct {
	stateManager *ethchain.StateManager
	blockChain   *ethchain.BlockChain
	txPool       *ethchain.TxPool
	Db           *Debugger
}

func (lib *EthLib) ImportAndSetPrivKey(privKey string) bool {
	fmt.Println(privKey)
	mnemonic := strings.Split(privKey, " ")
	if len(mnemonic) == 24 {
		fmt.Println("Got mnemonic key, importing.")
		key := ethutil.MnemonicDecode(mnemonic)
		utils.ImportPrivateKey(key)
	} else if len(mnemonic) == 1 {
		fmt.Println("Got hex key, importing.")
		utils.ImportPrivateKey(privKey)
	} else {
		fmt.Println("Did not recognise format, exiting.")
		return false
	}
	return true
}

func (lib *EthLib) CreateAndSetPrivKey() (string, string, string, string) {
	pub, prv := secp256k1.GenerateKeyPair()
	pair := &ethutil.Key{PrivateKey: prv, PublicKey: pub}
	ethutil.Config.Db.Put([]byte("KeyRing"), pair.RlpEncode())
	mne := ethutil.MnemonicEncode(ethutil.Hex(prv))
	mnemonicString := strings.Join(mne, " ")
	return mnemonicString, fmt.Sprintf("%x", pair.Address()), fmt.Sprintf("%x", prv), fmt.Sprintf("%x", pub)
}

func (lib *EthLib) GetKey() string {
	return ethutil.Hex(ethutil.Config.Db.GetKeys()[0].Address())
}

func (lib *EthLib) GetStateObject(address string) *Contract {
	stateObject := lib.stateManager.ProcState().GetContract(ethutil.FromHex(address))

	return NewContract(stateObject)
}

func (lib *EthLib) CreateTx(recipient, valueStr, gasStr, gasPriceStr, dataStr string) (string, error) {
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
		// Compile script
		mainScript, initScript, err := utils.CompileScript(dataStr)
		if err != nil {
			return "", err
		}

		tx = ethchain.NewContractCreationTx(value, gas, gasPrice, mainScript, initScript)
	} else {
		lines := strings.Split(dataStr, "\n")
		var data []byte
		for _, line := range lines {
			data = append(data, ethutil.BigToBytes(ethutil.Big(line), 256)...)
		}

		tx = ethchain.NewTransactionMessage(hash, value, gas, gasPrice, data)
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

func (lib *EthLib) GetBlock(hexHash string) *Block {
	hash, err := hex.DecodeString(hexHash)
	if err != nil {
		return nil
	}

	block := lib.blockChain.GetBlock(hash)

	return &Block{Number: int(block.BlockInfo().Number), Hash: ethutil.Hex(block.Hash())}
}
