package state

import (
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

type Log struct {
	Address common.Address
	Topics  []common.Hash
	Data    []byte
	Number  uint64

	TxHash    common.Hash
	TxIndex   uint
	BlockHash common.Hash
	Index     uint
}

func NewLog(address common.Address, topics []common.Hash, data []byte, number uint64) *Log {
	return &Log{Address: address, Topics: topics, Data: data, Number: number}
}

func (self *Log) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{self.Address, self.Topics, self.Data})
}

func (self *Log) String() string {
	return fmt.Sprintf(`log: %x %x %x %x %d %x %d`, self.Address, self.Topics, self.Data, self.TxHash, self.TxIndex, self.BlockHash, self.Index)
}

type Logs []*Log

type LogForStorage Log

func (self *LogForStorage) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{
		self.Address,
		self.Topics,
		self.Data,
		self.Number,
		self.TxHash,
		self.TxIndex,
		self.BlockHash,
		self.Index,
	})
}
