package ethpub

import (
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
	_ "log"
	"strings"
)

// Block interface exposed to QML
type PBlock struct {
	ref          *ethchain.Block
	Number       int    `json:"number"`
	Hash         string `json:"hash"`
	Transactions string `json:"transactions"`
	Time         int64  `json:"time"`
}

// Creates a new QML Block from a chain block
func NewPBlock(block *ethchain.Block) *PBlock {
	if block == nil {
		return nil
	}

	var ptxs []PTx
	for _, tx := range block.Transactions() {
		ptxs = append(ptxs, *NewPTx(tx))
	}

	txJson, err := json.Marshal(ptxs)
	if err != nil {
		return nil
	}

	return &PBlock{ref: block, Number: int(block.Number.Uint64()), Hash: ethutil.Hex(block.Hash()), Transactions: string(txJson), Time: block.Time}
}

func (self *PBlock) ToString() string {
	if self.ref != nil {
		return self.ref.String()
	}

	return ""
}

func (self *PBlock) GetTransaction(hash string) *PTx {
	tx := self.ref.GetTransaction(ethutil.FromHex(hash))
	if tx == nil {
		return nil
	}

	return NewPTx(tx)
}

type PTx struct {
	ref *ethchain.Transaction

	Value           string `json:"value"`
	Gas             string `json:"gas"`
	GasPrice        string `json:"gasPrice"`
	Hash            string `json:"hash"`
	Address         string `json:"address"`
	Sender          string `json:"sender"`
	RawData         string `json:"rawData"`
	Data            string `json:"data"`
	Contract        bool   `json:"isContract"`
	CreatesContract bool   `json:"createsContract"`
}

func NewPTx(tx *ethchain.Transaction) *PTx {
	hash := hex.EncodeToString(tx.Hash())
	receiver := hex.EncodeToString(tx.Recipient)

	if receiver == "" {
		receiver = hex.EncodeToString(tx.CreationAddress())
	}
	sender := hex.EncodeToString(tx.Sender())
	createsContract := tx.CreatesContract()

	data := strings.Join(ethchain.Disassemble(tx.Data), "\n")

	isContract := len(tx.Data) > 0

	return &PTx{ref: tx, Hash: hash, Value: ethutil.CurrencyToString(tx.Value), Address: receiver, Contract: isContract, Gas: tx.Gas.String(), GasPrice: tx.GasPrice.String(), Data: data, Sender: sender, CreatesContract: createsContract, RawData: hex.EncodeToString(tx.Data)}
}

func (self *PTx) ToString() string {
	return self.ref.String()
}

type PKey struct {
	Address    string `json:"address"`
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

func NewPKey(key *ethutil.KeyPair) *PKey {
	return &PKey{ethutil.Hex(key.Address()), ethutil.Hex(key.PrivateKey), ethutil.Hex(key.PublicKey)}
}

type PReceipt struct {
	CreatedContract bool   `json:"createdContract"`
	Address         string `json:"address"`
	Hash            string `json:"hash"`
	Sender          string `json:"sender"`
}

func NewPReciept(contractCreation bool, creationAddress, hash, address []byte) *PReceipt {
	return &PReceipt{
		contractCreation,
		ethutil.Hex(creationAddress),
		ethutil.Hex(hash),
		ethutil.Hex(address),
	}
}

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

func (c *PStateObject) Nonce() int {
	if c.object != nil {
		return int(c.object.Nonce)
	}

	return 0
}

func (c *PStateObject) Root() string {
	if c.object != nil {
		return ethutil.Hex(ethutil.NewValue(c.object.State().Root()).Bytes())
	}

	return "<err>"
}

func (c *PStateObject) IsContract() bool {
	if c.object != nil {
		return len(c.object.Script()) > 0
	}

	return false
}

func (c *PStateObject) Script() string {
	if c.object != nil {
		return strings.Join(ethchain.Disassemble(c.object.Script()), " ")
	}

	return ""
}

type PStorageState struct {
	StateAddress string
	Address      string
	Value        string
}

func NewPStorageState(storageObject *ethchain.StorageState) *PStorageState {
	return &PStorageState{ethutil.Hex(storageObject.StateAddress), ethutil.Hex(storageObject.Address), storageObject.Value.String()}
}
