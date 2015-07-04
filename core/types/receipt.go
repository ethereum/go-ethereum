package types

import (
	"bytes"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/rlp"
)

type Receipt struct {
	PostState         []byte
	CumulativeGasUsed *big.Int
	Bloom             Bloom
	TxHash            common.Hash
	ContractAddress   common.Address
	logs              state.Logs
}

func NewReceipt(root []byte, cumalativeGasUsed *big.Int) *Receipt {
	return &Receipt{PostState: common.CopyBytes(root), CumulativeGasUsed: new(big.Int).Set(cumalativeGasUsed)}
}

func (self *Receipt) SetLogs(logs state.Logs) {
	self.logs = logs
}

func (self *Receipt) Logs() state.Logs {
	return self.logs
}

func (self *Receipt) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{self.PostState, self.CumulativeGasUsed, self.Bloom, self.logs})
}

func (self *Receipt) DecodeRLP(s *rlp.Stream) error {
	var r struct {
		PostState         []byte
		CumulativeGasUsed *big.Int
		Bloom             Bloom
		TxHash            common.Hash
		ContractAddress   common.Address
		Logs              state.Logs
	}
	if err := s.Decode(&r); err != nil {
		return err
	}
	self.PostState, self.CumulativeGasUsed, self.Bloom, self.TxHash, self.ContractAddress, self.logs = r.PostState, r.CumulativeGasUsed, r.Bloom, r.TxHash, r.ContractAddress, r.Logs

	return nil
}

type ReceiptForStorage Receipt

func (self *ReceiptForStorage) EncodeRLP(w io.Writer) error {
	storageLogs := make([]*state.LogForStorage, len(self.logs))
	for i, log := range self.logs {
		storageLogs[i] = (*state.LogForStorage)(log)
	}
	return rlp.Encode(w, []interface{}{self.PostState, self.CumulativeGasUsed, self.Bloom, self.TxHash, self.ContractAddress, storageLogs})
}

func (self *Receipt) RlpEncode() []byte {
	bytes, err := rlp.EncodeToBytes(self)
	if err != nil {
		fmt.Println("TMP -- RECEIPT ENCODE ERROR", err)
	}
	return bytes
}

func (self *Receipt) Cmp(other *Receipt) bool {
	if bytes.Compare(self.PostState, other.PostState) != 0 {
		return false
	}

	return true
}

func (self *Receipt) String() string {
	return fmt.Sprintf("receipt{med=%x cgas=%v bloom=%x logs=%v}", self.PostState, self.CumulativeGasUsed, self.Bloom, self.logs)
}

type Receipts []*Receipt

func (self Receipts) RlpEncode() []byte {
	bytes, err := rlp.EncodeToBytes(self)
	if err != nil {
		fmt.Println("TMP -- RECEIPTS ENCODE ERROR", err)
	}
	return bytes
}

func (self Receipts) Len() int            { return len(self) }
func (self Receipts) GetRlp(i int) []byte { return common.Rlp(self[i]) }
