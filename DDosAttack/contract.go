package DDosAttack

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type ILowThroughputContract interface {
	AddOperation(op IOperation)
	replaceOperation(index int, op IOperation)
	Operations() []IOperation
	Size() int
	StackSize() int
	ToHex() string
	ToBytes() []byte
	Show()
}

type LowThroughputContract struct {
	operations []IOperation
	stackSize  int
}

func NewContractGenerator() *LowThroughputContract {
	return &LowThroughputContract{
		[]IOperation{},
		0,
	}
}

func (c *LowThroughputContract) AddOperation(op IOperation) {
	c.operations = append(c.operations, op)
}

func (c *LowThroughputContract) replaceOperation(index int, op IOperation) {
	c.operations[index] = op
}

func (c *LowThroughputContract) Operations() []IOperation {
	return c.operations
}

func (c *LowThroughputContract) Size() int {
	return len(c.operations)
}

func (c *LowThroughputContract) StackSize() int {
	return c.stackSize
}

func (c *LowThroughputContract) ToHex() string {
	return hexutil.Encode(c.ToBytes())
}

func (c *LowThroughputContract) ToBytes() []byte {
	var ret []byte
	for _, op := range c.operations {
		ret = append(ret, op.ToBytes()...)
	}
	return ret
}

func (c *LowThroughputContract) Show() {
	for _, op := range c.operations {
		fmt.Printf("%v 0x%x\n", op.OpString(), op.Arg())
	}
}
