// +build evmjit

package vm

/*
#include <stdint.h>
#include <stdlib.h>

struct evmjit_result
{
	int32_t  returnCode;
	uint64_t returnDataSize;
	void*    returnData;
};

struct evmjit_result evmjit_run(void* _data, void* _env);

// Shared library evmjit (e.g. libevmjit.so) is expected to be installed in /usr/local/lib
// More: https://github.com/ethereum/evmjit
#cgo LDFLAGS: -levmjit
*/
import "C"

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/state"
	"math/big"
	"unsafe"
)

type JitVm struct {
	env        Environment
	me         ContextRef
	callerAddr []byte
	price      *big.Int
	data       RuntimeData
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

func llvm2hash(m *i256) []byte {
	return C.GoBytes(unsafe.Pointer(m), C.int(len(m)))
}

func llvm2hashRef(m *i256) []byte {
	return (*[1 << 30]byte)(unsafe.Pointer(m))[:len(m):len(m)]
}

func address2llvm(addr []byte) i256 {
	n := hash2llvm(addr)
	bswap(&n)
	return n
}

// bswap swap bytes of the 256-bit integer on LLVM side
// TODO: Do not change memory on LLVM side, that can conflict with memory access optimizations
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

func getDataPtr(m []byte) *byte {
	var p *byte
	if len(m) > 0 {
		p = &m[0]
	}
	return p
}

func big2llvm(n *big.Int) i256 {
	m := hash2llvm(n.Bytes())
	bswap(&m)
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

// llvm2bytesRef creates a []byte slice that references byte buffer on LLVM side (as of that not controller by GC)
// User must asure that referenced memory is available to Go until the data is copied or not needed any more
func llvm2bytesRef(data *byte, length uint64) []byte {
	if length == 0 {
		return nil
	}
	if data == nil {
		panic("Unexpected nil data pointer")
	}
	return (*[1 << 30]byte)(unsafe.Pointer(data))[:length:length]
}

func untested(condition bool, message string) {
	if condition {
		panic("Condition `" + message + "` tested. Remove assert.")
	}
}

func assert(condition bool, message string) {
	if !condition {
		panic("Assert `" + message + "` failed!")
	}
}

func NewJitVm(env Environment) *JitVm {
	return &JitVm{env: env}
}

func (self *JitVm) Run(me, caller ContextRef, code []byte, value, gas, price *big.Int, callData []byte) (ret []byte, err error) {
	// TODO: depth is increased but never checked by VM. VM should not know about it at all.
	self.env.SetDepth(self.env.Depth() + 1)

	// TODO: Move it to Env.Call() or sth
	if Precompiled[string(me.Address())] != nil {
		// if it's address of precopiled contract
		// fallback to standard VM
		stdVm := New(self.env)
		return stdVm.Run(me, caller, code, value, gas, price, callData)
	}

	if self.me != nil {
		panic("JitVm.Run() can be called only once per JitVm instance")
	}

	self.me = me
	self.callerAddr = caller.Address()
	self.price = price

	self.data.elems[Gas] = big2llvm(gas)
	self.data.elems[address] = address2llvm(self.me.Address())
	self.data.elems[Caller] = address2llvm(caller.Address())
	self.data.elems[Origin] = address2llvm(self.env.Origin())
	self.data.elems[CallValue] = big2llvm(value)
	self.data.elems[CallDataSize] = big2llvm(big.NewInt(int64(len(callData)))) // TODO: Keep call data size as i64
	self.data.elems[GasPrice] = big2llvm(price)
	self.data.elems[CoinBase] = address2llvm(self.env.Coinbase())
	self.data.elems[TimeStamp] = big2llvm(big.NewInt(self.env.Time())) // TODO: Keep timestamp as i64
	self.data.elems[Number] = big2llvm(self.env.BlockNumber())
	self.data.elems[Difficulty] = big2llvm(self.env.Difficulty())
	self.data.elems[GasLimit] = big2llvm(self.env.GasLimit())
	self.data.elems[CodeSize] = big2llvm(big.NewInt(int64(len(code)))) // TODO: Keep code size as i64
	self.data.callData = getDataPtr(callData)
	self.data.code = getDataPtr(code)

	result := C.evmjit_run(unsafe.Pointer(&self.data), unsafe.Pointer(self))

	if result.returnCode >= 100 {
		err = errors.New("OOG from JIT")
		gas.SetInt64(0) // Set gas to 0, JIT does not bother
	} else {
		gasLeft := llvm2big(&self.data.elems[Gas]) // TODO: Set value directly to gas instance
		gas.Set(gasLeft)
		if result.returnCode == 1 { // RETURN
			ret = C.GoBytes(result.returnData, C.int(result.returnDataSize))
			C.free(result.returnData)
		} else if result.returnCode == 2 { // SUICIDE
			// TODO: Suicide support logic should be moved to Env to be shared by VM implementations
			state := self.Env().State()
			receiverAddr := llvm2hashRef(bswap(&self.data.elems[address]))
			receiver := state.GetOrNewStateObject(receiverAddr)
			balance := state.GetBalance(me.Address())
			receiver.AddAmount(balance)
			state.Delete(me.Address())
		}
	}

	return
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
func env_sha3(dataPtr *byte, length uint64, resultPtr unsafe.Pointer) {
	data := llvm2bytesRef(dataPtr, length)
	hash := crypto.Sha3(data)
	result := (*i256)(resultPtr)
	*result = hash2llvm(hash)
}

//export env_sstore
func env_sstore(vmPtr unsafe.Pointer, indexPtr unsafe.Pointer, valuePtr unsafe.Pointer) {
	vm := (*JitVm)(vmPtr)
	index := llvm2hash(bswap((*i256)(indexPtr)))
	value := llvm2hash(bswap((*i256)(valuePtr)))
	value = trim(value)
	if len(value) == 0 {
		prevValue := vm.env.State().GetState(vm.me.Address(), index)
		if len(prevValue) != 0 {
			vm.Env().State().Refund(vm.callerAddr, GasSStoreRefund)
		}
	}

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
	result := (*i256)(_result)

	currNumber := vm.Env().BlockNumber()
	limit := big.NewInt(0).Sub(currNumber, big.NewInt(256))
	if number.Cmp(limit) >= 0 && number.Cmp(currNumber) < 0 {
		hash := vm.Env().GetHash(uint64(number.Int64()))
		*result = hash2llvm(hash)
	} else {
		*result = i256{}
	}
}

//export env_call
func env_call(_vm unsafe.Pointer, _gas unsafe.Pointer, _receiveAddr unsafe.Pointer, _value unsafe.Pointer, inDataPtr unsafe.Pointer, inDataLen uint64, outDataPtr *byte, outDataLen uint64, _codeAddr unsafe.Pointer) bool {
	vm := (*JitVm)(_vm)

	//fmt.Printf("env_call (depth %d)\n", vm.Env().Depth())

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered in env_call (depth %d, out %p %d): %s\n", vm.Env().Depth(), outDataPtr, outDataLen, r)
		}
	}()

	balance := vm.Env().State().GetBalance(vm.me.Address())
	value := llvm2big((*i256)(_value))

	if balance.Cmp(value) >= 0 {
		receiveAddr := llvm2hash((*i256)(_receiveAddr))
		inData := C.GoBytes(inDataPtr, C.int(inDataLen))
		outData := llvm2bytesRef(outDataPtr, outDataLen)
		codeAddr := llvm2hash((*i256)(_codeAddr))
		llvmGas := (*i256)(_gas)
		gas := llvm2big(llvmGas)
		var out []byte
		var err error
		if bytes.Equal(codeAddr, receiveAddr) {
			out, err = vm.env.Call(vm.me, codeAddr, inData, gas, vm.price, value)
		} else {
			out, err = vm.env.CallCode(vm.me, codeAddr, inData, gas, vm.price, value)
		}
		*llvmGas = big2llvm(gas)
		if err == nil {
			copy(outData, out)
			return true
		}
	}

	return false
}

//export env_create
func env_create(_vm unsafe.Pointer, _gas unsafe.Pointer, _value unsafe.Pointer, initDataPtr unsafe.Pointer, initDataLen uint64, _result unsafe.Pointer) {
	vm := (*JitVm)(_vm)

	value := llvm2big((*i256)(_value))
	initData := C.GoBytes(initDataPtr, C.int(initDataLen)) // TODO: Unnecessary if low balance
	result := (*i256)(_result)
	*result = i256{}

	llvmGas := (*i256)(_gas)
	gas := llvm2big(llvmGas)

	ret, suberr, ref := vm.env.Create(vm.me, nil, initData, gas, vm.price, value)
	if suberr == nil {
		dataGas := big.NewInt(int64(len(ret))) // TODO: Nto the best design. env.Create can do it, it has the reference to gas counter
		dataGas.Mul(dataGas, GasCreateByte)
		gas.Sub(gas, dataGas)
		*result = hash2llvm(ref.Address())
	}
	*llvmGas = big2llvm(gas)
}

//export env_log
func env_log(_vm unsafe.Pointer, dataPtr unsafe.Pointer, dataLen uint64, _topic1 unsafe.Pointer, _topic2 unsafe.Pointer, _topic3 unsafe.Pointer, _topic4 unsafe.Pointer) {
	vm := (*JitVm)(_vm)

	data := C.GoBytes(dataPtr, C.int(dataLen))

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

//export env_extcode
func env_extcode(_vm unsafe.Pointer, _addr unsafe.Pointer, o_size *uint64) *byte {
	vm := (*JitVm)(_vm)
	addr := llvm2hash((*i256)(_addr))
	code := vm.Env().State().GetCode(addr)
	*o_size = uint64(len(code))
	return getDataPtr(code)
}
