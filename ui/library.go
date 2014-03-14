package ethui

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
	"strings"
)

type EthLib struct {
	stateManager *ethchain.StateManager
	blockChain   *ethchain.BlockChain
	txPool       *ethchain.TxPool
}

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
	keyRing := ethutil.NewValueFromBytes(k)

	amount := ethutil.Big(a)
	code := ethchain.Compile(strings.Split(data, "\n"))
	tx := ethchain.NewTransaction(hash, amount, code)
	tx.Nonce = lib.stateManager.GetAddrState(keyRing.Get(1).Bytes()).Nonce

	tx.Sign(keyRing.Get(0).Bytes())

	lib.txPool.QueueTransaction(tx)

	if len(receiver) == 0 {
		ethutil.Config.Log.Infof("Contract addr %x", tx.Hash()[12:])
	} else {
		ethutil.Config.Log.Infof("Tx hash %x", tx.Hash())
	}

	return ethutil.Hex(tx.Hash())
}

func (lib *EthLib) GetBlock(hexHash string) *Block {
	hash, err := hex.DecodeString(hexHash)
	if err != nil {
		return nil
	}

	block := lib.blockChain.GetBlock(hash)
	fmt.Println(block)

	return &Block{Number: int(block.BlockInfo().Number), Hash: ethutil.Hex(block.Hash())}
}
