package ethui

import (
	"fmt"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/go-ethereum/utils"
	"github.com/obscuren/secp256k1-go"
	"strings"
)

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
	_, prv := secp256k1.GenerateKeyPair()
	keyPair, err := ethutil.GetKeyRing().NewKeyPair(prv)
	if err != nil {
		panic(err)
	}

	mne := ethutil.MnemonicEncode(ethutil.Hex(keyPair.PrivateKey))
	mnemonicString := strings.Join(mne, " ")
	return mnemonicString, fmt.Sprintf("%x", keyPair.Address()), ethutil.Hex(keyPair.PrivateKey), ethutil.Hex(keyPair.PublicKey)
}
