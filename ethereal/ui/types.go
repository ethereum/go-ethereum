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
	object *ethchain.StateObject
}

func NewQStateObject(object *ethchain.StateObject) *QStateObject {
	return &QStateObject{object: object}
}

func (c *QStateObject) GetStorage(address string) string {
	// Because somehow, even if you return nil to QML it
	// still has some magical object so we can't rely on
	// undefined or null at the QML side
	if c.object != nil {
		val := c.object.GetMem(ethutil.Big("0x" + address))

		return val.BigInt().String()
	}

	return ""
}

func (c *QStateObject) Value() string {
	if c.object != nil {
		return c.object.Amount.String()
	}

	return ""
}

func (c *QStateObject) Address() string {
	if c.object != nil {
		return ethutil.Hex(c.object.Address())
	}

	return ""
}
