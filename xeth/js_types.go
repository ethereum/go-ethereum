package xeth

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/state"
)

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
		ptxs[i] = NewJSTx(tx, block.State())
	}
	txlist := ethutil.NewList(ptxs)

	puncles := make([]*JSBlock, len(block.Uncles))
	for i, uncle := range block.Uncles {
		puncles[i] = NewJSBlock(uncle)
	}
	ulist := ethutil.NewList(puncles)

	return &JSBlock{
		ref: block, Size: block.Size().String(),
		Number: int(block.Number.Uint64()), GasUsed: block.GasUsed.String(),
		GasLimit: block.GasLimit.String(), Hash: ethutil.Bytes2Hex(block.Hash()),
		Transactions: txlist, Uncles: ulist,
		Time:     block.Time,
		Coinbase: ethutil.Bytes2Hex(block.Coinbase),
		PrevHash: ethutil.Bytes2Hex(block.PrevHash),
		Bloom:    ethutil.Bytes2Hex(block.LogsBloom),
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
	tx := self.ref.GetTransaction(ethutil.Hex2Bytes(hash))
	if tx == nil {
		return nil
	}

	return NewJSTx(tx, self.ref.State())
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

func NewJSTx(tx *types.Transaction, state *state.StateDB) *JSTransaction {
	hash := ethutil.Bytes2Hex(tx.Hash())
	receiver := ethutil.Bytes2Hex(tx.To())
	if receiver == "0000000000000000000000000000000000000000" {
		receiver = ethutil.Bytes2Hex(core.AddressFromMessage(tx))
	}
	sender := ethutil.Bytes2Hex(tx.Sender())
	createsContract := core.MessageCreatesContract(tx)

	var data string
	if createsContract {
		data = strings.Join(core.Disassemble(tx.Data()), "\n")
	} else {
		data = ethutil.Bytes2Hex(tx.Data())
	}

	return &JSTransaction{ref: tx, Hash: hash, Value: ethutil.CurrencyToString(tx.Value()), Address: receiver, Contract: createsContract, Gas: tx.Gas().String(), GasPrice: tx.GasPrice().String(), Data: data, Sender: sender, CreatesContract: createsContract, RawData: ethutil.Bytes2Hex(tx.Data())}
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
	return &JSKey{ethutil.Bytes2Hex(key.Address()), ethutil.Bytes2Hex(key.PrivateKey), ethutil.Bytes2Hex(key.PublicKey)}
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
		ethutil.Bytes2Hex(creationAddress),
		ethutil.Bytes2Hex(hash),
		ethutil.Bytes2Hex(address),
	}
}

// Peer interface exposed to QML

type JSPeer struct {
	ref          *core.Peer
	Inbound      bool   `json:"isInbound"`
	LastSend     int64  `json:"lastSend"`
	LastPong     int64  `json:"lastPong"`
	Ip           string `json:"ip"`
	Port         int    `json:"port"`
	Version      string `json:"version"`
	LastResponse string `json:"lastResponse"`
	Latency      string `json:"latency"`
	Caps         string `json:"caps"`
}

func NewJSPeer(peer core.Peer) *JSPeer {
	if peer == nil {
		return nil
	}

	var ip []string
	for _, i := range peer.Host() {
		ip = append(ip, strconv.Itoa(int(i)))
	}
	ipAddress := strings.Join(ip, ".")

	var caps []string
	capsIt := peer.Caps().NewIterator()
	for capsIt.Next() {
		cap := capsIt.Value().Get(0).Str()
		ver := capsIt.Value().Get(1).Uint()
		caps = append(caps, fmt.Sprintf("%s/%d", cap, ver))
	}

	return &JSPeer{ref: &peer, Inbound: peer.Inbound(), LastSend: peer.LastSend().Unix(), LastPong: peer.LastPong(), Version: peer.Version(), Ip: ipAddress, Port: int(peer.Port()), Latency: peer.PingTime(), Caps: "[" + strings.Join(caps, ", ") + "]"}
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
		ethutil.Bytes2Hex(creationAddress),
		ethutil.Bytes2Hex(hash),
		ethutil.Bytes2Hex(address),
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
		To:        ethutil.Bytes2Hex(message.To),
		From:      ethutil.Bytes2Hex(message.From),
		Input:     ethutil.Bytes2Hex(message.Input),
		Output:    ethutil.Bytes2Hex(message.Output),
		Path:      int32(message.Path),
		Origin:    ethutil.Bytes2Hex(message.Origin),
		Timestamp: int32(message.Timestamp),
		Coinbase:  ethutil.Bytes2Hex(message.Origin),
		Block:     ethutil.Bytes2Hex(message.Block),
		Number:    int32(message.Number.Int64()),
		Value:     message.Value.String(),
	}
}
