// +build evmjit

package vm

import "math/big"
import "github.com/ethereum/go-ethereum/crypto"
import "github.com/ethereum/go-ethereum/state"

/*
#include "../evmjit/libevmjit/interface.h"

#cgo LDFLAGS: -L/home/chfast/go/src/github.com/ethereum/go-ethereum/evmjit/build/libevmjit -levmjit
*/
import "C"

import "unsafe"
import "fmt"
import "reflect"
import "errors"

type JitVm struct {
	env Environment
	me  ContextRef
}

type i256 [32]byte

const (
	Gas = iota
	address
	Caller
	Origin
	CallValue
	CallDataSize
	GasPrice
	CoinBase
	TimeStamp
	Number
	Difficulty
	GasLimit
	CodeSize

	_size

	ReturnDataOffset   = CallValue // Reuse 2 fields for return data reference
	ReturnDataSize     = CallDataSize
	SuicideDestAddress = address ///< Suicide balance destination address
)

type RuntimeData struct {
	elems    [_size]i256
	callData *byte
	code     *byte
}

func hash2llvm(h []byte) i256 {
	var m i256
	copy(m[len(m)-len(h):], h) // right aligned copy
	return m
}

func llvm2hash(m *i256) []byte { //TODO: It should copy data
	hdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(m)),
		Len:  int(len(m)),
		Cap:  int(len(m)),
	}
	return *(*[]byte)(unsafe.Pointer(&hdr))
}

func bswap(m *i256) *i256 {
	for i, l := 0, len(m); i < l/2; i++ {
		m[i], m[l-i-1] = m[l-i-1], m[i]
	}
	return m
}

func trim(m []byte) []byte {
	skip := 0
	for i := 0; i < len(m); i++ {
		if m[i] == 0 {
			skip++
		} else {
			break
		}
	}
	return m[skip:]
}

func big2llvm(n *big.Int) i256 {
	m := hash2llvm(n.Bytes())
	for i, l := 0, len(m); i < l/2; i++ {
		m[i], m[l-i-1] = m[l-i-1], m[i]
	}
	return m
}

func llvm2big(m *i256) *big.Int {
	n := big.NewInt(0)
	for i := 0; i < len(m); i++ {
		b := big.NewInt(int64(m[i]))
		b.Lsh(b, uint(i)*8)
		n.Add(n, b)
	}
	return n
}

func llvm2bytes(data *byte, length uint64) []byte {
	hdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(data)),
		Len:  int(length),
		Cap:  int(length),
	}
	return *(*[]byte)(unsafe.Pointer(&hdr))
}

func NewJitVm(env Environment) *JitVm {
	return &JitVm{env: env}
}

func (self *JitVm) Run(me, caller ContextRef, code []byte, value, gas, price *big.Int, callData []byte) (ret []byte, err error) {
	self.me = me // FIXME: Make sure Run() is not used more than once

	var data RuntimeData
	data.elems[Gas] = big2llvm(gas)
	data.elems[address] = hash2llvm(self.me.Address())
	data.elems[Caller] = hash2llvm(caller.Address())
	data.elems[Origin] = hash2llvm(self.env.Origin())
	data.elems[CallValue] = big2llvm(value)
	data.elems[CallDataSize] = big2llvm(big.NewInt(int64(len(callData)))) // TODO: Keep call data size as i64
	data.elems[CoinBase] = hash2llvm(self.env.Coinbase())
	data.elems[TimeStamp] = big2llvm(big.NewInt(self.env.Time())) // TODO: Keep timestamp as i64
	data.elems[Number] = big2llvm(self.env.BlockNumber())
	data.elems[Difficulty] = big2llvm(self.env.Difficulty())
	data.elems[GasLimit] = big2llvm(self.env.GasLimit())
	data.elems[CodeSize] = big2llvm(big.NewInt(int64(len(code)))) // TODO: Keep code size as i64
	if len(callData) > 0 {
		data.callData = &callData[0]
	}
	if len(code) > 0 {
		data.code = &code[0]
	}

	r := C.evmjit_run(unsafe.Pointer(&data), unsafe.Pointer(self))
	fmt.Printf("JIT result: %d\n", r)

	gasLeft := llvm2big(&data.elems[Gas]) // TODO: Set value directly to gas instance
	gas.Set(gasLeft)

	if r >= 100 {
		err = errors.New("OOG from JIT")
	}

	return ret, err
}

func (self *JitVm) Printf(format string, v ...interface{}) VirtualMachine {
	return self
}

func (self *JitVm) Endl() VirtualMachine {
	return self
}

func (self *JitVm) Env() Environment {
	return self.env
}

//export env_sha3
func env_sha3(dataPtr *byte, length uint64, hashPtr unsafe.Pointer) {
	data := llvm2bytes(dataPtr, length)
	hash := crypto.Sha3(data)

	hashHdr := reflect.SliceHeader{
		Data: uintptr(hashPtr),
		Len:  32,
		Cap:  32,
	}
	oHash := *(*[]byte)(unsafe.Pointer(&hashHdr))

	copy(oHash, hash)
}

//export env_sstore
func env_sstore(vmPtr unsafe.Pointer, indexPtr unsafe.Pointer, valuePtr unsafe.Pointer) {
	vm := (*JitVm)(vmPtr)
	index := llvm2hash(bswap((*i256)(indexPtr)))
	value := llvm2hash(bswap((*i256)(valuePtr)))
	value = trim(value)
	vm.env.State().SetState(vm.me.Address(), index, value)
}

//export env_sload
func env_sload(vmPtr unsafe.Pointer, indexPtr unsafe.Pointer, resultPtr unsafe.Pointer) {
	vm := (*JitVm)(vmPtr)
	index := llvm2hash(bswap((*i256)(indexPtr)))
	value := vm.env.State().GetState(vm.me.Address(), index)
	result := (*i256)(resultPtr)
	*result = hash2llvm(value)
	bswap(result)
}

//export env_balance
func env_balance(_vm unsafe.Pointer, _addr unsafe.Pointer, _result unsafe.Pointer) {
	vm := (*JitVm)(_vm)
	addr := llvm2hash((*i256)(_addr))
	balance := vm.Env().State().GetBalance(addr)
	result := (*i256)(_result)
	*result = big2llvm(balance)
}

//export env_blockhash
func env_blockhash(_vm unsafe.Pointer, _number unsafe.Pointer, _result unsafe.Pointer) {
	vm := (*JitVm)(_vm)
	number := llvm2big((*i256)(_number))
	hash := vm.Env().GetHash(uint64(number.Int64()))
	result := (*i256)(_result)
	*result = hash2llvm(hash)
}

//export env_call
func env_call(_vm unsafe.Pointer, _gas unsafe.Pointer, _receiveAddr unsafe.Pointer, _value unsafe.Pointer, inDataPtr *byte, inDataLen uint64, outDataPtr *byte, outDataLen uint64, _codeAddr unsafe.Pointer) bool {
	vm := (*JitVm)(_vm)

	balance := vm.Env().State().GetBalance(vm.me.Address())
	value := llvm2big((*i256)(_value))

	if vm.env.Depth() < 1024 && balance.Cmp(value) >= 0 {
		vm.env.State().AddBalance(vm.me.Address(), value.Neg(value))
		receiveAddr := llvm2hash((*i256)(_receiveAddr))
		inData := llvm2bytes(inDataPtr, inDataLen)
		outData := llvm2bytes(outDataPtr, outDataLen)
		//codeAddr := llvm2hash((*i256)(_codeAddr)) //TODO: Handle CallCode
		llvmGas := (*i256)(_gas)
		gas := llvm2big(llvmGas)
		price := big.NewInt(0) // TODO
		out, err := vm.env.Call(vm.me, receiveAddr, inData, gas, price, value)
		*llvmGas = big2llvm(gas)
		if err == nil {
			copy(outData, out)
			return true
		}
	}

	return false
}

//export env_create
func env_create(_vm unsafe.Pointer, _gas unsafe.Pointer, _value unsafe.Pointer, initDataPtr *byte, initDataLen uint64, _result unsafe.Pointer) {
	vm := (*JitVm)(_vm)

	balance := vm.Env().State().GetBalance(vm.me.Address())
	value := llvm2big((*i256)(_value))
	result := (*i256)(_result)

	if vm.env.Depth() < 1024 && balance.Cmp(value) >= 0 {
		vm.env.State().AddBalance(vm.me.Address(), value.Neg(value))
		initData := llvm2bytes(initDataPtr, initDataLen)
		llvmGas := (*i256)(_gas)
		gas := llvm2big(llvmGas)
		price := big.NewInt(0) // TODO
		addr, _, _ := vm.Env().Create(vm.me, vm.me.Address(), initData, gas, price, value)
		*llvmGas = big2llvm(gas)
		*result = hash2llvm(addr)
	} else {
		*result = i256{}
	}
}

//export env_log
func env_log(_vm unsafe.Pointer, dataPtr *byte, dataLen uint64, _topic1 unsafe.Pointer, _topic2 unsafe.Pointer, _topic3 unsafe.Pointer, _topic4 unsafe.Pointer) {
	vm := (*JitVm)(_vm)

	dataRef := llvm2bytes(dataPtr, dataLen)
	data := make([]byte, len(dataRef))
	copy(data, dataRef)

	topics := make([][]byte, 0, 4)
	if _topic1 != nil {
		topics = append(topics, llvm2hash((*i256)(_topic1)))
	}
	if _topic2 != nil {
		topics = append(topics, llvm2hash((*i256)(_topic2)))
	}
	if _topic3 != nil {
		topics = append(topics, llvm2hash((*i256)(_topic3)))
	}
	if _topic4 != nil {
		topics = append(topics, llvm2hash((*i256)(_topic4)))
	}

	vm.Env().AddLog(state.NewLog(vm.me.Address(), topics, data))
}
