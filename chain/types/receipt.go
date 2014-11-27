package types

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/state"
)

type Receipt struct {
	PostState         []byte
	CumulativeGasUsed *big.Int
	Bloom             []byte
	logs              state.Logs
}

func NewReceipt(root []byte, cumalativeGasUsed *big.Int) *Receipt {
	return &Receipt{PostState: ethutil.CopyBytes(root), CumulativeGasUsed: cumalativeGasUsed}
}

func NewRecieptFromValue(val *ethutil.Value) *Receipt {
	r := &Receipt{}
	r.RlpValueDecode(val)

	return r
}

func (self *Receipt) SetLogs(logs state.Logs) {
	self.logs = logs
}

func (self *Receipt) RlpValueDecode(decoder *ethutil.Value) {
	self.PostState = decoder.Get(0).Bytes()
	self.CumulativeGasUsed = decoder.Get(1).BigInt()
	self.Bloom = decoder.Get(2).Bytes()

	it := decoder.Get(3).NewIterator()
	for it.Next() {
		self.logs = append(self.logs, state.NewLogFromValue(it.Value()))
	}
}

func (self *Receipt) RlpData() interface{} {
	return []interface{}{self.PostState, self.CumulativeGasUsed, self.Bloom, self.logs.RlpData()}
}

func (self *Receipt) RlpEncode() []byte {
	return ethutil.Encode(self.RlpData())
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

func (self Receipts) Len() int            { return len(self) }
func (self Receipts) GetRlp(i int) []byte { return ethutil.Rlp(self[i]) }
