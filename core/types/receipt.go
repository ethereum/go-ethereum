package types

import (
	"bytes"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/state"
)

type Receipt struct {
	PostState         []byte
	CumulativeGasUsed *big.Int
	Bloom             Bloom
	logs              state.Logs
}

func NewReceipt(root []byte, cumalativeGasUsed *big.Int) *Receipt {
	return &Receipt{PostState: common.CopyBytes(root), CumulativeGasUsed: new(big.Int).Set(cumalativeGasUsed)}
}

func (self *Receipt) SetLogs(logs state.Logs) {
	self.logs = logs
}

func (self *Receipt) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{self.PostState, self.CumulativeGasUsed, self.Bloom, self.logs})
}

/*
func (self *Receipt) RlpData() interface{} {
	return []interface{}{self.PostState, self.CumulativeGasUsed, self.Bloom, self.logs.RlpData()}
}
*/

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

/*
func (self Receipts) RlpData() interface{} {
	data := make([]interface{}, len(self))
	for i, receipt := range self {
		data[i] = receipt.RlpData()
	}

	return data
}
*/

func (self Receipts) RlpEncode() []byte {
	bytes, err := rlp.EncodeToBytes(self)
	if err != nil {
		fmt.Println("TMP -- RECEIPTS ENCODE ERROR", err)
	}
	return bytes
}

func (self Receipts) Len() int            { return len(self) }
func (self Receipts) GetRlp(i int) []byte { return common.Rlp(self[i]) }
