// Copyright 2018 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// This file lists the EEI functions, so that they can be bound to any
// ewasm-compatible module, as well as the types of these functions

package vm

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/go-interpreter/wagon/exec"
	"github.com/go-interpreter/wagon/wasm"
)

const (
	// EEICallSuccess is the return value in case of a successful contract execution
	EEICallSuccess = 0
	// ErrEEICallFailure is the return value in case of a contract execution failture
	ErrEEICallFailure = 1
	// ErrEEICallRevert is the return value in case a contract calls `revert`
	ErrEEICallRevert = 2
)

// List of gas costs
const (
	GasCostZero           = 0
	GasCostBase           = 2
	GasCostVeryLow        = 3
	GasCostLow            = 5
	GasCostMid            = 8
	GasCostHigh           = 10
	GasCostExtCode        = 700
	GasCostBalance        = 400
	GasCostSLoad          = 200
	GasCostJumpDest       = 1
	GasCostSSet           = 20000
	GasCostSReset         = 5000
	GasRefundSClear       = 15000
	GasRefundSelfDestruct = 24000
	GasCostCreate         = 32000
	GasCostCall           = 700
	GasCostCallValue      = 9000
	GasCostCallStipend    = 2300
	GasCostLog            = 375
	GasCostLogData        = 8
	GasCostLogTopic       = 375
	GasCostCopy           = 3
	GasCostBlockHash      = 800
)

var eeiTypes = &wasm.SectionTypes{
	Entries: []wasm.FunctionSig{
		{
			ParamTypes:  []wasm.ValueType{wasm.ValueTypeI64},
			ReturnTypes: []wasm.ValueType{},
		},
		{
			ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32},
			ReturnTypes: []wasm.ValueType{},
		},
		{
			ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32},
			ReturnTypes: []wasm.ValueType{},
		},
		{
			ParamTypes:  []wasm.ValueType{wasm.ValueTypeI64, wasm.ValueTypeI32},
			ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
		},
		{
			ParamTypes:  []wasm.ValueType{wasm.ValueTypeI64, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32},
			ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
		},
		{
			ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32},
			ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
		},
		{
			ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32},
			ReturnTypes: []wasm.ValueType{},
		},
		{
			ParamTypes:  []wasm.ValueType{},
			ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
		},
		{
			ParamTypes:  []wasm.ValueType{wasm.ValueTypeI64, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32},
			ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
		},
		{
			ParamTypes:  []wasm.ValueType{wasm.ValueTypeI64, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32},
			ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
		},
		{
			ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32},
			ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
		},
		{
			ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32},
			ReturnTypes: []wasm.ValueType{},
		},
		{
			ParamTypes:  []wasm.ValueType{},
			ReturnTypes: []wasm.ValueType{wasm.ValueTypeI64},
		},
		{
			ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32},
			ReturnTypes: []wasm.ValueType{},
		},
	},
}

func swapEndian(src []byte) []byte {
	ret := make([]byte, len(src))
	for i, v := range src {
		ret[len(src)-i-1] = v
	}
	return ret
}

func (in *InterpreterEWASM) gasAccounting(cost uint64) {
	if in.contract == nil {
		panic("nil contract")
	}
	if cost > in.contract.Gas {
		panic("out of gas")
	}
	in.contract.Gas -= cost
}

func getDebugFuncs(in *InterpreterEWASM) []wasm.Function {
	return []wasm.Function{
		{
			Sig:  &eeiTypes.Entries[2],
			Host: reflect.ValueOf(func(p *exec.Process, o, l int32) { printMemHex(p, in, o, l) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[1],
			Host: reflect.ValueOf(func(p *exec.Process, o int32) { printStorageHex(p, in, o) }),
			Body: &wasm.FunctionBody{},
		},
	}
}

func printMemHex(p *exec.Process, in *InterpreterEWASM, offset, length int32) {
	data := readSize(p, offset, int(length))
	for _, v := range data {
		fmt.Printf("%02x", v)
	}
	fmt.Println("")
}

func printStorageHex(p *exec.Process, in *InterpreterEWASM, pathOffset int32) {

	path := common.BytesToHash(readSize(p, pathOffset, common.HashLength))
	val := in.StateDB.GetState(in.contract.Address(), path)
	for v := range val {
		fmt.Printf("%02x", v)
	}
	fmt.Println("")
}

// Return the list of function descriptors. This is a function instead of
// a variable in order to avoid an initialization loop.
func eeiFuncs(in *InterpreterEWASM) []wasm.Function {
	return []wasm.Function{
		{
			Sig:  &eeiTypes.Entries[0], // TODO use constants or find the right entry in the list
			Host: reflect.ValueOf(func(p *exec.Process, a int64) { useGas(p, in, a) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[1],
			Host: reflect.ValueOf(func(p *exec.Process, r int32) { getAddress(p, in, r) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[2],
			Host: reflect.ValueOf(func(p *exec.Process, a, r int32) { getExternalBalance(p, in, a, r) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[3],
			Host: reflect.ValueOf(func(p *exec.Process, n int64, r int32) int32 { return getBlockHash(p, in, n, r) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[4],
			Host: reflect.ValueOf(func(p *exec.Process, g int64, a, v, d, l int32) int32 { return call(p, in, g, a, v, d, l) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[6],
			Host: reflect.ValueOf(func(p *exec.Process, r, d, l int32) { callDataCopy(p, in, r, d, l) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[7],
			Host: reflect.ValueOf(func(p *exec.Process) int32 { return getCallDataSize(p, in) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[9],
			Host: reflect.ValueOf(func(p *exec.Process, g int64, a, v, d, l int32) int32 { return callCode(p, in, g, a, v, d, l) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[8],
			Host: reflect.ValueOf(func(p *exec.Process, g int64, a, d, l int32) int32 { return callDelegate(p, in, g, a, d, l) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[8],
			Host: reflect.ValueOf(func(p *exec.Process, g int64, a, d, l int32) int32 { return callStatic(p, in, g, a, d, l) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[2],
			Host: reflect.ValueOf(func(pr *exec.Process, p, v int32) { storageStore(pr, in, p, v) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[2],
			Host: reflect.ValueOf(func(pr *exec.Process, p, r int32) { storageLoad(pr, in, p, r) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[1],
			Host: reflect.ValueOf(func(p *exec.Process, r int32) { getCaller(p, in, r) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[2],
			Host: reflect.ValueOf(func(p *exec.Process, r int32) { getCallValue(p, in, r) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[6],
			Host: reflect.ValueOf(func(p *exec.Process, r, c, l int32) { codeCopy(p, in, r, c, l) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[7],
			Host: reflect.ValueOf(func(p *exec.Process) int32 { return getCodeSize(p, in) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[1],
			Host: reflect.ValueOf(func(p *exec.Process, r int32) { getBlockCoinbase(p, in, r) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[10],
			Host: reflect.ValueOf(func(p *exec.Process, v, d, l, r uint32) int32 { return create(p, in, v, d, l, r) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[1],
			Host: reflect.ValueOf(func(p *exec.Process, r int32) { getBlockDifficulty(p, in, r) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[11],
			Host: reflect.ValueOf(func(p *exec.Process, a, r, c, l int32) { externalCodeCopy(p, in, a, r, c, l) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[5],
			Host: reflect.ValueOf(func(p *exec.Process, a int32) int32 { return getExternalCodeSize(p, in, a) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[12],
			Host: reflect.ValueOf(func(p *exec.Process) int64 { return getGasLeft(p, in) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[12],
			Host: reflect.ValueOf(func(p *exec.Process) int64 { return getBlockGasLimit(p, in) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[5],
			Host: reflect.ValueOf(func(p *exec.Process, v int32) { getTxGasPrice(p, in, v) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[13],
			Host: reflect.ValueOf(func(p *exec.Process, d, l, n, t1, t2, t3, t4 int32) { log(p, in, d, l, n, t1, t2, t3, t4) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[12],
			Host: reflect.ValueOf(func(p *exec.Process) int64 { return getBlockNumber(p, in) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[1],
			Host: reflect.ValueOf(func(p *exec.Process, r int32) { getTxOrigin(p, in, r) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[2],
			Host: reflect.ValueOf(func(p *exec.Process, d, l int32) { finish(p, in, d, l) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[2],
			Host: reflect.ValueOf(func(p *exec.Process, d, l int32) { revert(p, in, d, l) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[7],
			Host: reflect.ValueOf(func(p *exec.Process) int32 { return getReturnDataSize(p, in) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[6],
			Host: reflect.ValueOf(func(p *exec.Process, r, d, l int32) { returnDataCopy(p, in, r, d, l) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[1],
			Host: reflect.ValueOf(func(p *exec.Process, a int32) { selfDestruct(p, in, a) }),
			Body: &wasm.FunctionBody{},
		},
		{
			Sig:  &eeiTypes.Entries[12],
			Host: reflect.ValueOf(func(p *exec.Process) int64 { return getBlockTimestamp(p, in) }),
			Body: &wasm.FunctionBody{},
		},
	}
}

func readSize(p *exec.Process, offset int32, size int) []byte {
	// TODO modify the process interface to find out how much memory is
	// available on the system.
	val := make([]byte, size)
	p.ReadAt(val, int64(offset))
	return val
}

func useGas(p *exec.Process, in *InterpreterEWASM, amount int64) {
	in.gasAccounting(uint64(amount))
}

func getAddress(p *exec.Process, in *InterpreterEWASM, resultOffset int32) {
	in.gasAccounting(GasCostBase)
	contractBytes := in.contract.CodeAddr.Bytes()
	p.WriteAt(contractBytes, int64(resultOffset))
}

func getExternalBalance(p *exec.Process, in *InterpreterEWASM, addressOffset int32, resultOffset int32) {
	in.gasAccounting(in.gasTable.Balance)
	addr := common.BytesToAddress(readSize(p, addressOffset, common.AddressLength))
	balance := swapEndian(in.StateDB.GetBalance(addr).Bytes())
	p.WriteAt(balance, int64(resultOffset))
}

func getBlockHash(p *exec.Process, in *InterpreterEWASM, number int64, resultOffset int32) int32 {
	in.gasAccounting(GasCostBlockHash)
	n := big.NewInt(number)
	fmt.Println(n)
	n.Sub(in.evm.Context.BlockNumber, n)
	fmt.Println(n, n.Cmp(big.NewInt(256)), n.Cmp(big.NewInt(0)))
	if n.Cmp(big.NewInt(256)) > 0 || n.Cmp(big.NewInt(0)) <= 0 {
		return 1
	}
	h := in.evm.GetHash(uint64(number))
	p.WriteAt(h.Bytes(), int64(resultOffset))
	return 0
}

func callCommon(in *InterpreterEWASM, contract, targetContract *Contract, input []byte, value *big.Int, snapshot int, gas int64, ro bool) int32 {
	if in.evm.depth > maxCallDepth {
		contract.UseGas(contract.Gas)
		return ErrEEICallFailure
	}

	savedVM := in.vm

	in.Run(targetContract, input, ro)

	in.vm = savedVM
	in.contract = contract

	if value.Cmp(big.NewInt(0)) != 0 {
		in.gasAccounting(uint64(gas) - targetContract.Gas - GasCostCallStipend)
	} else {
		in.gasAccounting(uint64(gas) - targetContract.Gas)
	}

	switch in.terminationType {
	case TerminateFinish:
		return EEICallSuccess
	case TerminateRevert:
		in.StateDB.RevertToSnapshot(snapshot)
		return ErrEEICallRevert
	default:
		in.StateDB.RevertToSnapshot(snapshot)
		contract.UseGas(targetContract.Gas)
		return ErrEEICallFailure
	}
}

func call(p *exec.Process, in *InterpreterEWASM, gas int64, addressOffset int32, valueOffset int32, dataOffset int32, dataLength int32) int32 {
	in.gasAccounting(GasCostCall)

	contract := in.contract

	// Get the address of the contract to call
	addr := common.BytesToAddress(readSize(p, addressOffset, common.AddressLength))

	// Get the value. The [spec](https://github.com/ewasm/design/blob/master/eth_interface.md#call)
	// requires this operation to be U128, which is incompatible with the EVM version that expects
	// a u256.
	value := big.NewInt(0).SetBytes(swapEndian(readSize(p, valueOffset, u128Len)))

	if value.Cmp(big.NewInt(0)) != 0 {
		in.gasAccounting(GasCostCallValue)
		gas += GasCostCallStipend
	}

	// Get the arguments.
	// TODO check the need for callvalue (seems not, a lot of that stuff is
	// already accounted for in the functions that I already called - need to
	// refactor all that)
	input := readSize(p, dataOffset, int(dataLength))

	snapshot := in.StateDB.Snapshot()

	// Check that the contract exists
	if !in.StateDB.Exist(addr) {
		// TODO check that no new account creation stuff is required
		in.StateDB.CreateAccount(addr)
	}

	// Check that there is enough balance to transfer the value
	if in.StateDB.GetBalance(contract.Address()).Cmp(value) < 0 {
		fmt.Printf("Not enough balance: wanted to use %v, got %v\n", value, in.StateDB.GetBalance(addr))
		in.contract.Gas += GasCostCallStipend
		return ErrEEICallFailure
	}

	// TODO tracing
	// TODO check that EIP-150 is respected

	// Add amount to recipient
	in.evm.Transfer(in.StateDB, contract.Address(), addr, value)

	// Load the contract code in a new VM structure
	targetContract := NewContract(contract, AccountRef(addr), value, uint64(gas))
	code := in.StateDB.GetCode(addr)
	targetContract.SetCallCode(&addr, in.StateDB.GetCodeHash(addr), code)

	return callCommon(in, contract, targetContract, input, value, snapshot, gas, false)
}

func callDataCopy(p *exec.Process, in *InterpreterEWASM, resultOffset int32, dataOffset int32, length int32) {
	in.gasAccounting(GasCostVeryLow + GasCostCopy*(uint64(length+31)>>5))
	p.WriteAt(in.contract.Input[dataOffset:dataOffset+length], int64(resultOffset))
}

func getCallDataSize(p *exec.Process, in *InterpreterEWASM) int32 {
	in.gasAccounting(GasCostBase)
	return int32(len(in.contract.Input))
}

func callCode(p *exec.Process, in *InterpreterEWASM, gas int64, addressOffset int32, valueOffset int32, dataOffset int32, dataLength int32) int32 {
	in.gasAccounting(GasCostCall)

	contract := in.contract

	// Get the address of the contract to call
	addr := common.BytesToAddress(readSize(p, addressOffset, common.AddressLength))

	// Get the value. The [spec](https://github.com/ewasm/design/blob/master/eth_interface.md#call)
	// requires this operation to be U128, which is incompatible with the EVM version that expects
	// a u256.
	value := big.NewInt(0).SetBytes(readSize(p, valueOffset, u128Len))

	if value.Cmp(big.NewInt(0)) != 0 {
		in.gasAccounting(GasCostCallValue)
		gas += GasCostCallStipend
	}

	// Get the arguments.
	// TODO check the need for callvalue (seems not, a lot of that stuff is
	// already accounted for in the functions that I already called - need to
	// refactor all that)
	input := readSize(p, dataOffset, int(dataLength))

	snapshot := in.StateDB.Snapshot()

	// Check that there is enough balance to transfer the value
	if in.StateDB.GetBalance(addr).Cmp(value) < 0 {
		fmt.Printf("Not enough balance: wanted to use %v, got %v\n", value, in.StateDB.GetBalance(addr))
		return ErrEEICallFailure
	}

	// TODO tracing
	// TODO check that EIP-150 is respected

	// Load the contract code in a new VM structure
	targetContract := NewContract(contract.caller, AccountRef(contract.Address()), value, uint64(gas))
	code := in.StateDB.GetCode(addr)
	targetContract.SetCallCode(&addr, in.StateDB.GetCodeHash(addr), code)

	return callCommon(in, contract, targetContract, input, value, snapshot, gas, false)
}

func callDelegate(p *exec.Process, in *InterpreterEWASM, gas int64, addressOffset int32, dataOffset int32, dataLength int32) int32 {
	in.gasAccounting(GasCostCall)

	contract := in.contract

	// Get the address of the contract to call
	addr := common.BytesToAddress(readSize(p, addressOffset, common.AddressLength))

	// Get the value. The [spec](https://github.com/ewasm/design/blob/master/eth_interface.md#call)
	// requires this operation to be U128, which is incompatible with the EVM version that expects
	// a u256.
	value := contract.value

	if value.Cmp(big.NewInt(0)) != 0 {
		in.gasAccounting(GasCostCallValue)
		gas += GasCostCallStipend
	}

	// Get the arguments.
	// TODO check the need for callvalue (seems not, a lot of that stuff is
	// already accounted for in the functions that I already called - need to
	// refactor all that)
	input := readSize(p, dataOffset, int(dataLength))

	snapshot := in.StateDB.Snapshot()

	// Check that there is enough balance to transfer the value
	if in.StateDB.GetBalance(addr).Cmp(value) < 0 {
		fmt.Printf("Not enough balance: wanted to use %v, got %v\n", value, in.StateDB.GetBalance(addr))
		return ErrEEICallFailure
	}

	// TODO tracing
	// TODO check that EIP-150 is respected

	// Load the contract code in a new VM structure
	targetContract := NewContract(AccountRef(contract.Address()), AccountRef(contract.Address()), value, uint64(gas))
	code := in.StateDB.GetCode(addr)
	caddr := contract.Address()
	targetContract.SetCallCode(&caddr, in.StateDB.GetCodeHash(addr), code)

	return callCommon(in, contract, targetContract, input, value, snapshot, gas, false)
}

func callStatic(p *exec.Process, in *InterpreterEWASM, gas int64, addressOffset int32, dataOffset int32, dataLength int32) int32 {
	in.gasAccounting(GasCostCall)

	contract := in.contract

	// Get the address of the contract to call
	addr := common.BytesToAddress(readSize(p, addressOffset, common.AddressLength))

	value := big.NewInt(0)

	// Get the arguments.
	// TODO check the need for callvalue (seems not, a lot of that stuff is
	// already accounted for in the functions that I already called - need to
	// refactor all that)
	input := readSize(p, dataOffset, int(dataLength))

	snapshot := in.StateDB.Snapshot()

	// Check that the contract exists
	if !in.StateDB.Exist(addr) {
		// TODO check that no new account creation stuff is required
		in.StateDB.CreateAccount(addr)
	}

	// Check that there is enough balance to transfer the value
	if in.StateDB.GetBalance(addr).Cmp(value) < 0 {
		fmt.Printf("Not enough balance: wanted to use %v, got %v\n", value, in.StateDB.GetBalance(addr))
		in.contract.Gas += GasCostCallStipend
		return ErrEEICallFailure
	}

	// TODO tracing
	// TODO check that EIP-150 is respected

	// Add amount to recipient
	in.evm.Transfer(in.StateDB, contract.Address(), addr, value)

	// Load the contract code in a new VM structure
	targetContract := NewContract(contract, AccountRef(addr), value, uint64(gas))
	code := in.StateDB.GetCode(addr)
	targetContract.SetCallCode(&addr, in.StateDB.GetCodeHash(addr), code)

	saveStatic := in.staticMode
	in.staticMode = true
	defer func() { in.staticMode = saveStatic }()

	return callCommon(in, contract, targetContract, input, value, snapshot, gas, true)
}

func storageStore(p *exec.Process, interpreter *InterpreterEWASM, pathOffset int32, valueOffset int32) {
	if interpreter.staticMode == true {
		panic("Static mode violation in storageStore")
	}

	loc := common.BytesToHash(readSize(p, pathOffset, u256Len))
	val := common.BytesToHash(readSize(p, valueOffset, u256Len))

	fmt.Println(val, loc)
	nonZeroBytes := 0
	for _, b := range val.Bytes() {
		if b != 0 {
			nonZeroBytes++
		}
	}

	oldValue := interpreter.StateDB.GetState(interpreter.contract.Address(), loc)
	oldNonZeroBytes := 0
	for _, b := range oldValue.Bytes() {
		if b != 0 {
			oldNonZeroBytes++
		}
	}

	if (nonZeroBytes > 0 && oldNonZeroBytes != nonZeroBytes) || (oldNonZeroBytes != 0 && nonZeroBytes == 0) {
		interpreter.gasAccounting(GasCostSSet)
	} else {
		// Refund for setting one value to 0 or if the "zeroness" remains
		// unchanged.
		interpreter.gasAccounting(GasCostSReset)
	}

	interpreter.StateDB.SetState(interpreter.contract.Address(), loc, val)
}

func storageLoad(p *exec.Process, interpreter *InterpreterEWASM, pathOffset int32, resultOffset int32) {
	interpreter.gasAccounting(interpreter.gasTable.SLoad)
	loc := common.BytesToHash(readSize(p, pathOffset, u256Len))
	valBytes := interpreter.StateDB.GetState(interpreter.contract.Address(), loc).Bytes()
	p.WriteAt(valBytes, int64(resultOffset))
}

func getCaller(p *exec.Process, in *InterpreterEWASM, resultOffset int32) {
	callerAddress := in.contract.CallerAddress
	in.gasAccounting(GasCostBase)
	p.WriteAt(callerAddress.Bytes(), int64(resultOffset))
}

func getCallValue(p *exec.Process, in *InterpreterEWASM, resultOffset int32) {
	in.gasAccounting(GasCostBase)
	p.WriteAt(swapEndian(in.contract.Value().Bytes()), int64(resultOffset))
}

func codeCopy(p *exec.Process, in *InterpreterEWASM, resultOffset int32, codeOffset int32, length int32) {
	in.gasAccounting(GasCostVeryLow + GasCostCopy*(uint64(length+31)>>5))
	code := in.contract.Code
	p.WriteAt(code[codeOffset:codeOffset+length], int64(resultOffset))
}

func getCodeSize(p *exec.Process, in *InterpreterEWASM) int32 {
	in.gasAccounting(GasCostBase)
	code := in.StateDB.GetCode(*in.contract.CodeAddr)
	return int32(len(code))
}

func getBlockCoinbase(p *exec.Process, in *InterpreterEWASM, resultOffset int32) {
	in.gasAccounting(GasCostBase)
	p.WriteAt(in.evm.Coinbase.Bytes(), int64(resultOffset))
}

func sentinel(in *InterpreterEWASM, input []byte) ([]byte, error) {
	savedContract := in.contract
	savedVM := in.vm
	defer func() {
		in.contract = savedContract
		in.vm = savedVM
	}()
	meteringContractAddress := common.HexToAddress("0x000000000000000000000000000000000000000a")
	meteringCode := in.StateDB.GetCode(meteringContractAddress)
	in.contract = NewContract(in.contract, AccountRef(meteringContractAddress), &big.Int{}, 0)
	in.contract = in.meteringContract
	in.contract.SetCallCode(&meteringContractAddress, crypto.Keccak256Hash(meteringCode), meteringCode)
	in.vm = in.meteringVM
	meteredCode, err := in.meteringVM.ExecCode(0)
	return meteredCode.([]byte), err
}

func create(p *exec.Process, in *InterpreterEWASM, valueOffset uint32, codeOffset uint32, length uint32, resultOffset uint32) int32 {
	in.gasAccounting(GasCostCreate)
	savedVM := in.vm
	savedContract := in.contract
	defer func() {
		in.vm = savedVM
		in.contract = savedContract
	}()
	in.terminationType = TerminateInvalid

	if int(codeOffset)+int(length) > len(in.vm.Memory()) {
		return ErrEEICallFailure
	}
	input := readSize(p, int32(codeOffset), int(length))

	if (int(valueOffset) + u128Len) > len(in.vm.Memory()) {
		return ErrEEICallFailure
	}
	value := swapEndian(readSize(p, int32(valueOffset), u128Len))

	in.terminationType = TerminateFinish

	// EIP150 says that the calling contract should keep 1/64th of the
	// leftover gas.
	gas := in.contract.Gas - in.contract.Gas/64
	in.gasAccounting(gas)

	/* Meter the contract code if metering is enabled */
	if in.metering {
		/* It seems that hera doesn't handle errors */
		input, _ = sentinel(in, input)
	}

	_, addr, gasLeft, _ := in.evm.Create(in.contract, input, gas, big.NewInt(0).SetBytes(value))

	switch in.terminationType {
	case TerminateFinish:
		savedContract.Gas += gasLeft
		p.WriteAt(addr.Bytes(), int64(resultOffset))
		return EEICallSuccess
	case TerminateRevert:
		savedContract.Gas += gas
		return ErrEEICallRevert
	default:
		savedContract.Gas += gasLeft
		return ErrEEICallFailure
	}
}

func getBlockDifficulty(p *exec.Process, in *InterpreterEWASM, resultOffset int32) {
	in.gasAccounting(GasCostBase)
	p.WriteAt(swapEndian(in.evm.Difficulty.Bytes()), int64(resultOffset))
}

func externalCodeCopy(p *exec.Process, in *InterpreterEWASM, addressOffset int32, resultOffset int32, codeOffset int32, length int32) {
	in.gasAccounting(in.gasTable.ExtcodeCopy + GasCostCopy*(uint64(length+31)>>5))
	addr := common.BytesToAddress(readSize(p, addressOffset, common.AddressLength))
	code := in.StateDB.GetCode(addr)
	p.WriteAt(code[codeOffset:codeOffset+length], int64(resultOffset))
}

func getExternalCodeSize(p *exec.Process, in *InterpreterEWASM, addressOffset int32) int32 {
	in.gasAccounting(in.gasTable.ExtcodeSize)
	addr := common.BytesToAddress(readSize(p, addressOffset, common.AddressLength))
	code := in.StateDB.GetCode(addr)
	return int32(len(code))
}

func getGasLeft(p *exec.Process, in *InterpreterEWASM) int64 {
	in.gasAccounting(GasCostBase)
	return int64(in.contract.Gas)
}

func getBlockGasLimit(p *exec.Process, in *InterpreterEWASM) int64 {
	in.gasAccounting(GasCostBase)
	return int64(in.evm.GasLimit)
}

func getTxGasPrice(p *exec.Process, in *InterpreterEWASM, valueOffset int32) {
	in.gasAccounting(GasCostBase)
	p.WriteAt(in.evm.GasPrice.Bytes(), int64(valueOffset))
}

// It would be nice to be able to use variadic functions to pass the number of topics,
// however this imposes a change in wagon because the number of arguments is being
// checked when calling a function.
func log(p *exec.Process, in *InterpreterEWASM, dataOffset int32, length int32, numberOfTopics int32, topic1 int32, topic2 int32, topic3 int32, topic4 int32) {
	in.gasAccounting(GasCostLog + GasCostLogData*uint64(length) + uint64(numberOfTopics)*GasCostLogTopic)

	// TODO need to add some info about the memory boundary on wagon
	if uint64(len(in.vm.Memory())) <= uint64(length)+uint64(dataOffset) {
		panic("out of memory")
	}
	data := readSize(p, dataOffset, int(uint32(length)))
	topics := make([]common.Hash, numberOfTopics)

	if numberOfTopics > 4 || numberOfTopics < 0 {
		in.terminationType = TerminateInvalid
		p.Terminate()
	}

	// Variadic functions FTW
	if numberOfTopics > 0 {
		if uint64(len(in.vm.Memory())) <= uint64(topic1) {
			panic("out of memory")
		}
		topics[0] = common.BigToHash(big.NewInt(0).SetBytes(readSize(p, topic1, u256Len)))
	}
	if numberOfTopics > 1 {
		if uint64(len(in.vm.Memory())) <= uint64(topic2) {
			panic("out of memory")
		}
		topics[1] = common.BigToHash(big.NewInt(0).SetBytes(readSize(p, topic2, u256Len)))
	}
	if numberOfTopics > 2 {
		if uint64(len(in.vm.Memory())) <= uint64(topic3) {
			panic("out of memory")
		}
		topics[2] = common.BigToHash(big.NewInt(0).SetBytes(readSize(p, topic3, u256Len)))
	}
	if numberOfTopics > 3 {
		if uint64(len(in.vm.Memory())) <= uint64(topic3) {
			panic("out of memory")
		}
		topics[3] = common.BigToHash(big.NewInt(0).SetBytes(readSize(p, topic4, u256Len)))
	}

	in.StateDB.AddLog(&types.Log{
		Address:     in.contract.Address(),
		Topics:      topics,
		Data:        data,
		BlockNumber: in.evm.BlockNumber.Uint64(),
	})
}

func getBlockNumber(p *exec.Process, in *InterpreterEWASM) int64 {
	in.gasAccounting(GasCostBase)
	return in.evm.BlockNumber.Int64()
}

func getTxOrigin(p *exec.Process, in *InterpreterEWASM, resultOffset int32) {
	in.gasAccounting(GasCostBase)
	p.WriteAt(in.evm.Origin.Big().Bytes(), int64(resultOffset))
}

func unWindContract(p *exec.Process, in *InterpreterEWASM, dataOffset int32, length int32) {
	in.returnData = make([]byte, length)
	p.ReadAt(in.returnData, int64(dataOffset))
}

func finish(p *exec.Process, in *InterpreterEWASM, dataOffset int32, length int32) {
	unWindContract(p, in, dataOffset, length)

	in.terminationType = TerminateFinish
	p.Terminate()
}

func revert(p *exec.Process, in *InterpreterEWASM, dataOffset int32, length int32) {
	unWindContract(p, in, dataOffset, length)

	in.terminationType = TerminateRevert
	p.Terminate()
}

func getReturnDataSize(p *exec.Process, in *InterpreterEWASM) int32 {
	in.gasAccounting(GasCostBase)
	return int32(len(in.returnData))
}

func returnDataCopy(p *exec.Process, in *InterpreterEWASM, resultOffset int32, dataOffset int32, length int32) {
	in.gasAccounting(GasCostVeryLow + GasCostCopy*(uint64(length+31)>>5))
	p.WriteAt(in.returnData[dataOffset:dataOffset+length], int64(resultOffset))
}

func selfDestruct(p *exec.Process, in *InterpreterEWASM, addressOffset int32) {
	contract := in.contract
	mem := in.vm.Memory()

	balance := in.StateDB.GetBalance(contract.Address())

	addr := common.BytesToAddress(mem[addressOffset : addressOffset+common.AddressLength])
	in.StateDB.AddBalance(addr, balance)

	totalGas := in.gasTable.Suicide
	// If the destination address doesn't exist, add the account creation costs
	if in.StateDB.Empty(addr) && balance.Sign() != 0 {
		totalGas += in.gasTable.CreateBySuicide
	}
	in.gasAccounting(totalGas)

	in.StateDB.Suicide(contract.Address())

	// Same as for `revert` and `return`, I need to forcefully terminate
	// the execution of the contract.
	in.terminationType = TerminateSuicide
	p.Terminate()
}

func getBlockTimestamp(p *exec.Process, in *InterpreterEWASM) int64 {
	in.gasAccounting(GasCostBase)
	return in.evm.Time.Int64()
}
