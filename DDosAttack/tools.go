package DDosAttack

import (
	"encoding/hex"
	"github.com/ethereum/go-ethereum/core/asm"
	"github.com/holiman/uint256"
	"math/big"
)

func Disasm(code string) ([]IOperation, error) {
	script, err := hex.DecodeString(code)
	if err != nil {
		return []IOperation{}, err
	}

	var operations []IOperation

	operationFactory := GetOperationFactory()

	it := asm.NewInstructionIterator(script)
	for it.Next() {
		//if it.Op() == vm.JUMPI {
		//	fmt.Println(111)
		//}
		if it.Arg() != nil && 0 < len(it.Arg()) {
			operation := operationFactory.GetOperationByOpCode(it.Op())
			z := new(big.Int)
			z.SetBytes(it.Arg())
			value, _ := uint256.FromBig(z)
			operationData := OperationData{
				len(it.Arg()),
				*value,
			}
			operationWithData, err2 := NewOperationWithData(operation, operationData)
			if err2 != nil {
				return []IOperation{}, err2
			}
			operations = append(operations, &operationWithData)
		} else {
			operation := operationFactory.GetOperationByOpCode(it.Op())
			operations = append(operations, &operation)
		}
	}
	return operations, it.Error()
}
