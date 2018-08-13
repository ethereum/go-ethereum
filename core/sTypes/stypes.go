package stypes

import (
	"time"

	"github.com/ShyftNetwork/go-empyrean/common"
)

//SBlock type
type SBlock struct {
	Hash       string
	Coinbase   string
	AgeGet     string
	Age        time.Time
	ParentHash string
	UncleHash  string
	Difficulty string
	Size       string
	Rewards    string
	Number     string
	GasUsed    uint64
	GasLimit   uint64
	Nonce      uint64
	TxCount    int
	UncleCount int
	Blocks     []SBlock
}

type InteralWrite struct {
	ID      int
	Hash    string
	Action  string
	From    string
	To      string
	Value   string
	Gas     uint64
	GasUsed uint64
	Input   string
	Output  string
	Time    string
}

type InternalArray struct {
	InternalEntry []InteralWrite
}

//blockRes struct
type BlockRes struct {
	hash     string
	coinbase string
	number   string
	Blocks   []SBlock
}

type SAccounts struct {
	Addr         string
	Balance      string
	AccountNonce string
}

type AccountRes struct {
	addr        string
	balance     string
	AllAccounts []SAccounts
}

type TxRes struct {
	TxEntry []ShyftTxEntryPretty
}

type ShyftTxEntryPretty struct {
	TxHash      string
	To          *common.Address
	ToGet       string
	From        string
	BlockHash   string
	BlockNumber string
	Amount      string
	GasPrice    uint64
	Gas         uint64
	GasLimit    uint64
	Cost        uint64
	Nonce       uint64
	Status      string
	IsContract  bool
	Age         time.Time
	Data        []byte
}

type SendAndReceive struct {
	To           string
	From         string
	Amount       string
	Address      string
	Balance      string
	AccountNonce uint64 `json:",string"`
}
