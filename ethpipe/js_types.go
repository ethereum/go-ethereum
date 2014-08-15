package ethpipe

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethutil"
)

// Block interface exposed to QML
type JSBlock struct {
	ref          *ethchain.Block
	Number       int    `json:"number"`
	Hash         string `json:"hash"`
	Transactions string `json:"transactions"`
	Time         int64  `json:"time"`
	Coinbase     string `json:"coinbase"`
	Name         string `json:"name"`
	GasLimit     string `json:"gasLimit"`
	GasUsed      string `json:"gasUsed"`
}

// Creates a new QML Block from a chain block
func NewJSBlock(block *ethchain.Block) *JSBlock {
	if block == nil {
		return nil
	}

	var ptxs []JSTransaction
	for _, tx := range block.Transactions() {
		ptxs = append(ptxs, *NewJSTx(tx))
	}

	txJson, err := json.Marshal(ptxs)
	if err != nil {
		return nil
	}

	return &JSBlock{ref: block, Number: int(block.Number.Uint64()), GasUsed: block.GasUsed.String(), GasLimit: block.GasLimit.String(), Hash: ethutil.Bytes2Hex(block.Hash()), Transactions: string(txJson), Time: block.Time, Coinbase: ethutil.Bytes2Hex(block.Coinbase)}
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

	return NewJSTx(tx)
}

type JSTransaction struct {
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
	Confirmations   int    `json:"confirmations"`
}

func NewJSTx(tx *ethchain.Transaction) *JSTransaction {
	hash := ethutil.Bytes2Hex(tx.Hash())
	receiver := ethutil.Bytes2Hex(tx.Recipient)
	if receiver == "0000000000000000000000000000000000000000" {
		receiver = ethutil.Bytes2Hex(tx.CreationAddress())
	}
	sender := ethutil.Bytes2Hex(tx.Sender())
	createsContract := tx.CreatesContract()

	var data string
	if tx.CreatesContract() {
		data = strings.Join(ethchain.Disassemble(tx.Data), "\n")
	} else {
		data = ethutil.Bytes2Hex(tx.Data)
	}

	return &JSTransaction{ref: tx, Hash: hash, Value: ethutil.CurrencyToString(tx.Value), Address: receiver, Contract: tx.CreatesContract(), Gas: tx.Gas.String(), GasPrice: tx.GasPrice.String(), Data: data, Sender: sender, CreatesContract: createsContract, RawData: ethutil.Bytes2Hex(tx.Data)}
}

func (self *JSTransaction) ToString() string {
	return self.ref.String()
}

type JSKey struct {
	Address    string `json:"address"`
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

func NewJSKey(key *ethcrypto.KeyPair) *JSKey {
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
	ref          *ethchain.Peer
	Inbound      bool   `json:"isInbound"`
	LastSend     int64  `json:"lastSend"`
	LastPong     int64  `json:"lastPong"`
	Ip           string `json:"ip"`
	Port         int    `json:"port"`
	Version      string `json:"version"`
	LastResponse string `json:"lastResponse"`
	Latency      string `json:"latency"`
}

func NewJSPeer(peer ethchain.Peer) *JSPeer {
	if peer == nil {
		return nil
	}

	var ip []string
	for _, i := range peer.Host() {
		ip = append(ip, strconv.Itoa(int(i)))
	}
	ipAddress := strings.Join(ip, ".")

	return &JSPeer{ref: &peer, Inbound: peer.Inbound(), LastSend: peer.LastSend().Unix(), LastPong: peer.LastPong(), Version: peer.Version(), Ip: ipAddress, Port: int(peer.Port()), Latency: peer.PingTime()}
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
