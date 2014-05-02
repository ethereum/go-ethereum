package utils

import (
	"encoding/hex"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
)

// Block interface exposed to QML
type PBlock struct {
	Number int
	Hash   string
}

// Creates a new QML Block from a chain block
func NewPBlock(block *ethchain.Block) *PBlock {
	info := block.BlockInfo()
	hash := hex.EncodeToString(block.Hash())

	return &PBlock{Number: int(info.Number), Hash: hash}
}

type PTx struct {
	Value, Hash, Address string
	Contract             bool
}

func NewPTx(tx *ethchain.Transaction) *PTx {
	hash := hex.EncodeToString(tx.Hash())
	sender := hex.EncodeToString(tx.Recipient)
	isContract := len(tx.Data) > 0

	return &PTx{Hash: hash, Value: ethutil.CurrencyToString(tx.Value), Address: sender, Contract: isContract}
}

type PKey struct {
	Address    string
	PrivateKey string
	PublicKey  string
}

func NewPKey(key *ethchain.KeyPair) *PKey {
	return &PKey{ethutil.Hex(key.Address()), ethutil.Hex(key.PrivateKey), ethutil.Hex(key.PublicKey)}
}

/*
type PKeyRing struct {
	Keys []interface{}
}

func NewPKeyRing(keys []interface{}) *PKeyRing {
	return &PKeyRing{Keys: keys}
}
*/

type PStateObject struct {
	object *ethchain.StateObject
}

func NewPStateObject(object *ethchain.StateObject) *PStateObject {
	return &PStateObject{object: object}
}

func (c *PStateObject) GetStorage(address string) string {
	// Because somehow, even if you return nil to QML it
	// still has some magical object so we can't rely on
	// undefined or null at the QML side
	if c.object != nil {
		val := c.object.GetMem(ethutil.Big("0x" + address))

		return val.BigInt().String()
	}

	return ""
}

func (c *PStateObject) Value() string {
	if c.object != nil {
		return c.object.Amount.String()
	}

	return ""
}

func (c *PStateObject) Address() string {
	if c.object != nil {
		return ethutil.Hex(c.object.Address())
	}

	return ""
}
