package vm

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/exp/maps"
)

// OpCodeInfo contains information required to identify an EVM operation.
type OpCodeInfo struct {
	Number OpCode
	Name   string
}

// Operation is an utility struct that wraps the private type
// operation.
type Operation struct {
	Op *operation
}

// ExtendActivators allows to merge the go ethereum activators map
// with additional custom activators.
func ExtendActivators(eips map[int]func(*JumpTable)) error {
	// Catch early duplicated eip.
	keys := make([]int, 0, len(eips))
	for k := range eips {
		if ValidEip(k) {
			return fmt.Errorf("duplicate activation: %d is already present in %s", k, ActivateableEips())
		}
		keys = append(keys, k)
	}

	// Sorting keys to ensure deterministic execution.
	sort.Ints(keys)

	for _, k := range keys {
		activators[k] = eips[k]
	}
	return nil
}

// GetActivatorsEipNumbers returns the name of EIPs registered in
// the activators map.
// Used only in tests.
func GetActivatorsEipNumbers() []int {
	keys := maps.Keys(activators)

	sort.Ints(keys)
	return keys
}

// ExtendOperations returns an instance of the new operation and register it in the list
// of available ones.
// Return an error if an operation with the same name is already present.
// This function is used to prevent the overwrite of an already existent operation.
func ExtendOperations(
	opInfo OpCodeInfo,
	execute executionFunc,
	constantGas uint64,
	dynamicGas gasFunc,
	minStack int,
	maxStack int,
	memorySize memorySizeFunc,
) (*Operation, error) {
	opName := strings.ToUpper(strings.TrimSpace(opInfo.Name))
	if err := extendOpCodeStringLists(opInfo.Number, opName); err != nil {
		return nil, err
	}

	operation := newOperation(execute, constantGas, dynamicGas, minStack, maxStack, memorySize)
	op := &Operation{operation}

	return op, nil
}

// newOperation returns an instance of a new EVM operation.
func newOperation(
	execute executionFunc,
	constantGas uint64,
	dynamicGas gasFunc,
	minStack int,
	maxStack int,
	memorySize memorySizeFunc,
) *operation {
	return &operation{
		execute:     execute,
		constantGas: constantGas,
		dynamicGas:  dynamicGas,
		minStack:    minStack,
		maxStack:    maxStack,
		memorySize:  memorySize,
	}
}

// GetConstantGas return the constant gas used by the operation.
func (o *operation) GetConstantGas() uint64 {
	return o.constantGas
}

// SetExecute sets the execution function of the operation.
func (o *operation) SetExecute(ef executionFunc) {
	o.execute = ef
}

// SetConstantGas changes the constant gas of the operation.
func (o *operation) SetConstantGas(gas uint64) {
	o.constantGas = gas
}

// SetDynamicGas sets the dynamic gas function of the operation.
func (o *operation) SetDynamicGas(gf gasFunc) {
	o.dynamicGas = gf
}

// SetMinStack sets the minimum stack size required for the operation.
func (o *operation) SetMinStack(minStack int) {
	o.minStack = minStack
}

// SetMaxStack sets the maximum stack size for the operation.
func (o *operation) SetMaxStack(maxStack int) {
	o.maxStack = maxStack
}

// SetMemorySize sets the memory size function for the operation.
func (o *operation) SetMemorySize(msf memorySizeFunc) {
	o.memorySize = msf
}

// extendOpCodeStringLists updates the lists mapping opcode number to the name
// and viceversa. Return an error if the key is already set.
//
// ASSUMPTION: no opcode is registered as an empty string.
func extendOpCodeStringLists(newOpCode OpCode, newOpName string) error {
	opName := opCodeToString[newOpCode]
	if opName != "" {
		return fmt.Errorf("opcode %d already exists: %s", newOpCode, opName)
	}
	opNumber := stringToOp[newOpName]
	// We need to check against the STOP opcode name because we have to discriminate
	// between 0x00 of this opcode and the default value of an empty key.
	stopName := opCodeToString[STOP]
	if opNumber != 0x00 || newOpName == stopName {
		return fmt.Errorf("opcode with name %s already exists", newOpName)
	}
	opCodeToString[newOpCode] = newOpName
	stringToOp[newOpName] = newOpCode
	return nil
}
