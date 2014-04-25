package ethui

import (
	"encoding/hex"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
)

// Block interface exposed to QML
type QBlock struct {
	Number int
	Hash   string
}

// Creates a new QML Block from a chain block
func NewQBlock(block *ethchain.Block) *QBlock {
	info := block.BlockInfo()
	hash := hex.EncodeToString(block.Hash())

	return &QBlock{Number: int(info.Number), Hash: hash}
}

type QTx struct {
	Value, Hash, Address string
	Contract             bool
}

func NewQTx(tx *ethchain.Transaction) *QTx {
	hash := hex.EncodeToString(tx.Hash())
	sender := hex.EncodeToString(tx.Recipient)
	isContract := len(tx.Data) > 0

	return &QTx{Hash: hash, Value: ethutil.CurrencyToString(tx.Value), Address: sender, Contract: isContract}
}

type QKey struct {
	Address string
}

type QKeyRing struct {
	Keys []interface{}
}

func NewQKeyRing(keys []interface{}) *QKeyRing {
	return &QKeyRing{Keys: keys}
}

type QStateObject struct {
	Address string
	Amount  string
	Nonce   int
}

func NewQStateObject(stateObject *ethchain.StateObject) *QStateObject {
	return &QStateObject{ethutil.Hex(stateObject.Address()), stateObject.Amount.String(), int(stateObject.Nonce)}
}
