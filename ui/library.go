package ethui

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
)

type EthLib struct {
	blockManager *ethchain.BlockManager
	blockChain   *ethchain.BlockChain
	txPool       *ethchain.TxPool
}

func (lib *EthLib) CreateTx(receiver, a string) string {
	hash, err := hex.DecodeString(receiver)
	if err != nil {
		return err.Error()
	}
	data, _ := ethutil.Config.Db.Get([]byte("KeyRing"))
	keyRing := ethutil.NewValueFromBytes(data)

	amount := ethutil.Big(a)
	tx := ethchain.NewTransaction(hash, amount, []string{""})
	tx.Nonce = lib.blockManager.GetAddrState(keyRing.Get(1).Bytes()).Nonce

	tx.Sign(keyRing.Get(0).Bytes())

	lib.txPool.QueueTransaction(tx)

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
