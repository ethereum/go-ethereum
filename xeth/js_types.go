package xeth

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/state"
)

func toHex(b []byte) string {
	return "0x" + ethutil.Bytes2Hex(b)
}
func fromHex(s string) []byte {
	if len(s) > 1 {
		if s[0:2] == "0x" {
			s = s[2:]
		}
		return ethutil.Hex2Bytes(s)
	}
	return nil
}

// Block interface exposed to QML
type JSBlock struct {
	//Transactions string `json:"transactions"`
	ref          *types.Block
	Size         string        `json:"size"`
	Number       int           `json:"number"`
	Hash         string        `json:"hash"`
	Transactions *ethutil.List `json:"transactions"`
	Uncles       *ethutil.List `json:"uncles"`
	Time         int64         `json:"time"`
	Coinbase     string        `json:"coinbase"`
	Name         string        `json:"name"`
	GasLimit     string        `json:"gasLimit"`
	GasUsed      string        `json:"gasUsed"`
	PrevHash     string        `json:"prevHash"`
	Bloom        string        `json:"bloom"`
	Raw          string        `json:"raw"`
}

// Creates a new QML Block from a chain block
func NewJSBlock(block *types.Block) *JSBlock {
	if block == nil {
		return &JSBlock{}
	}

	ptxs := make([]*JSTransaction, len(block.Transactions()))
	for i, tx := range block.Transactions() {
		ptxs[i] = NewJSTx(tx)
	}
	txlist := ethutil.NewList(ptxs)

	puncles := make([]*JSBlock, len(block.Uncles()))
	for i, uncle := range block.Uncles() {
		puncles[i] = NewJSBlock(types.NewBlockWithHeader(uncle))
	}
	ulist := ethutil.NewList(puncles)

	return &JSBlock{
		ref: block, Size: block.Size().String(),
		Number: int(block.NumberU64()), GasUsed: block.GasUsed().String(),
		GasLimit: block.GasLimit().String(), Hash: toHex(block.Hash()),
		Transactions: txlist, Uncles: ulist,
		Time:     block.Time(),
		Coinbase: toHex(block.Coinbase()),
		PrevHash: toHex(block.ParentHash()),
		Bloom:    toHex(block.Bloom()),
		Raw:      block.String(),
	}
}

func (self *JSBlock) ToString() string {
	if self.ref != nil {
		return self.ref.String()
	}

	return ""
}

func (self *JSBlock) GetTransaction(hash string) *JSTransaction {
	tx := self.ref.Transaction(fromHex(hash))
	if tx == nil {
		return nil
	}

	return NewJSTx(tx)
}

type JSTransaction struct {
	ref *types.Transaction

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
	Confirmations   int    `json:"confirmations"`
}

func NewJSTx(tx *types.Transaction) *JSTransaction {
	hash := toHex(tx.Hash())
	receiver := toHex(tx.To())
	if receiver == "0000000000000000000000000000000000000000" {
		receiver = toHex(core.AddressFromMessage(tx))
	}
	sender := toHex(tx.From())
	createsContract := core.MessageCreatesContract(tx)

	var data string
	if createsContract {
		data = strings.Join(core.Disassemble(tx.Data()), "\n")
	} else {
		data = toHex(tx.Data())
	}

	return &JSTransaction{ref: tx, Hash: hash, Value: ethutil.CurrencyToString(tx.Value()), Address: receiver, Contract: createsContract, Gas: tx.Gas().String(), GasPrice: tx.GasPrice().String(), Data: data, Sender: sender, CreatesContract: createsContract, RawData: toHex(tx.Data())}
}

func (self *JSTransaction) ToString() string {
	return self.ref.String()
}

type JSKey struct {
	Address    string `json:"address"`
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

func NewJSKey(key *crypto.KeyPair) *JSKey {
	return &JSKey{toHex(key.Address()), toHex(key.PrivateKey), toHex(key.PublicKey)}
}

type JSObject struct {
	*Object
}

func NewJSObject(object *Object) *JSObject {
	return &JSObject{object}
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
		toHex(creationAddress),
		toHex(hash),
		toHex(address),
	}
}

// Peer interface exposed to QML

type JSPeer struct {
	ref     *p2p.Peer
	Ip      string `json:"ip"`
	Version string `json:"version"`
	Caps    string `json:"caps"`
}

func NewJSPeer(peer *p2p.Peer) *JSPeer {
	var caps []string
	for _, cap := range peer.Caps() {
		caps = append(caps, fmt.Sprintf("%s/%d", cap.Name, cap.Version))
	}

	return &JSPeer{
		ref:     peer,
		Ip:      fmt.Sprintf("%v", peer.RemoteAddr()),
		Version: fmt.Sprintf("%v", peer.Identity()),
		Caps:    fmt.Sprintf("%v", caps),
	}
}

type JSReceipt struct {
	CreatedContract bool   `json:"createdContract"`
	Address         string `json:"address"`
	Hash            string `json:"hash"`
	Sender          string `json:"sender"`
}

func NewJSReciept(contractCreation bool, creationAddress, hash, address []byte) *JSReceipt {
	return &JSReceipt{
		contractCreation,
		toHex(creationAddress),
		toHex(hash),
		toHex(address),
	}
}

type JSMessage struct {
	To        string `json:"to"`
	From      string `json:"from"`
	Input     string `json:"input"`
	Output    string `json:"output"`
	Path      int32  `json:"path"`
	Origin    string `json:"origin"`
	Timestamp int32  `json:"timestamp"`
	Coinbase  string `json:"coinbase"`
	Block     string `json:"block"`
	Number    int32  `json:"number"`
	Value     string `json:"value"`
}

func NewJSMessage(message *state.Message) JSMessage {
	return JSMessage{
		To:        toHex(message.To),
		From:      toHex(message.From),
		Input:     toHex(message.Input),
		Output:    toHex(message.Output),
		Path:      int32(message.Path),
		Origin:    toHex(message.Origin),
		Timestamp: int32(message.Timestamp),
		Coinbase:  toHex(message.Origin),
		Block:     toHex(message.Block),
		Number:    int32(message.Number.Int64()),
		Value:     message.Value.String(),
	}
}
