package DDosAttack

import (
	"fmt"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"testing"
)

func TestOperationFactory_GetOpCodeByOpCode(t *testing.T) {
	operationFactory := GetOperationFactory()
	op1 := operationFactory.GetOperationByOpCode(vm.PUSH1)
	op2 := operationFactory.GetOperationByOpCode(vm.PUSH1)
	op3 := operationFactory.GetOperationByName("PUSH1")
	if op1 != op2 || op2 != op3 {
		t.Errorf("operation factory wrong")
	}
	fmt.Print(op1.ToByte())
}

func TestOperation_ToByte(t *testing.T) {
	operationFactory := GetOperationFactory()
	op1 := operationFactory.GetOperationByOpCode(vm.PUSH1)
	if op1.ToByte() != 96 {
		t.Errorf("operation ToByte wrong")
	}
}

func TestNewOperationWithData(t *testing.T) {
	operationFactory := GetOperationFactory()
	op1 := operationFactory.GetOperationByOpCode(vm.PUSH1)
	op2 := operationFactory.GetOperationByOpCode(vm.PUSH1)
	if op1 != op2 {
		t.Errorf("operation factory wrong")
	}
	opd1, err := NewOperationWithData(operationFactory.GetOperationByOpCode(vm.PUSH1), OperationData{
		1,
		*uint256.NewInt(uint64(12)),
	})
	if err != nil {
		t.Errorf("op1 failed")
		return
	}

	opd2, err2 := NewOperationWithData(operationFactory.GetOperationByOpCode(vm.PUSH1), OperationData{
		1,
		*uint256.NewInt(uint64(12)),
	})
	if err2 != nil {
		t.Errorf("op2 failed")
		return
	}

	if *opd1.operation != *opd2.operation {
		t.Errorf("op1's operation is not equal to op2's operation")
	}
}
