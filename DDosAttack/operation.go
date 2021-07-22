package DDosAttack

import (
	"errors"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

var operationFactorySingleInstance = &OperationFactory{
	OperationMap: make(map[vm.OpCode]Operation),
}

type IOperation interface {
	ToHex() string
	ToBytes() []byte
	Arg() []byte
	OpString() string
}

type Operation struct {
	Opcode *vm.OpCode
	Name   string
}

type OperationData struct {
	size  int
	value uint256.Int
}

type OperationWithData struct {
	operation *Operation
	data      *OperationData
}

func IsValidOperationData(size int, value uint256.Int) bool {
	return true
}

func NewOperationWithData(op Operation, data OperationData) (OperationWithData, error) {
	if IsValidOperationData(data.size, data.value) {
		return OperationWithData{
			&op,
			&data,
		}, nil
	} else {
		return OperationWithData{}, errors.New("data is invalid")
	}
}

func (o OperationWithData) ToHex() string {
	return hexutil.Encode(o.ToBytes())
}

func (o OperationWithData) ToBytes() []byte {
	return append([]byte{o.operation.ToByte()}, o.data.ToBytes()...)
}

func (o OperationWithData) Arg() []byte {
	return o.data.ToBytes()
}

func (o OperationWithData) OpString() string {
	return o.operation.Name
}

func (opData *OperationData) ToHex() string {
	return hexutil.Encode(opData.ToBytes())
}

func (opData *OperationData) ToBytes() []byte {
	bytes := opData.value.Bytes()
	if len(bytes) < opData.size {
		lackNumber := opData.size - len(bytes)
		var lackBytes []byte
		for i := 0; i < lackNumber; i++ {
			lackBytes = append(lackBytes, byte(0))
		}
		bytes = append(lackBytes, bytes...)
	}
	return bytes
}

func (opData *OperationData) Arg() []byte {
	return []byte{}
}

func GetOperationFactory() *OperationFactory {
	return operationFactorySingleInstance
}

type OperationFactory struct {
	OperationMap map[vm.OpCode]Operation
}

func (operationFactory *OperationFactory) GetOperationByName(name string) Operation {
	opCode := vm.StringToOp(name)

	if operation, ok := operationFactory.OperationMap[opCode]; ok {
		return operation
	} else {
		operation = *NewOperationByName(name)
		operationFactory.OperationMap[opCode] = operation
		return operation
	}
}

func (operationFactory *OperationFactory) GetOperationByOpCode(opCode vm.OpCode) Operation {
	return operationFactory.GetOperationByName(opCode.String())
}

func NewOperationByName(name string) *Operation {
	opCode := vm.StringToOp(name)
	return &Operation{
		&opCode,
		name,
	}
}

func (operation *Operation) ToByte() byte {
	return byte(*operation.Opcode)
}

func (operation *Operation) ToHex() string {
	return hexutil.Encode([]byte{operation.ToByte()})
}

func (operation *Operation) ToBytes() []byte {
	return []byte{operation.ToByte()}
}

func (operation *Operation) Arg() []byte {
	return []byte{}
}

func (operation *Operation) OpString() string {
	return operation.Name
}
