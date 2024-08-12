package rip7560

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/status-im/keycard-go/hexutils"
	"math/big"
	"testing"
)

const DEFAULT_SENDER = "0x1111111111222222222233333333334444444444"
const DEFAULT_BALANCE = 1 << 62

type testContext struct {
	genesisAlloc types.GenesisAlloc
	t            *testing.T
	chainContext *ethapi.ChainContext
	gaspool      *core.GasPool
	genesis      *core.Genesis
	genesisBlock *types.Block
}

func newTestContext(t *testing.T) *testContext {
	return newTestContextBuilder(t).build()
}

type testContextBuilder struct {
	t            *testing.T
	genesisAlloc types.GenesisAlloc
}

func newTestContextBuilder(t *testing.T) *testContextBuilder {
	genesisAlloc := types.GenesisAlloc{}

	return &testContextBuilder{
		t:            t,
		genesisAlloc: genesisAlloc,
	}
}

// return a contract code that will deploy the given code
func create2_contract(deployedCode []byte) []byte {

	return returnWithData(deployedCode)
}

// return the generated address when deploying the given code
func create2_addr(deployer common.Address, deployedCode []byte) common.Address {

	contractCode := create2_contract(deployedCode)
	data := createCode(0xff, deployer.Bytes(), common.Hash{}, crypto.Keccak256(contractCode))
	return common.BytesToAddress(crypto.Keccak256(data))
}

// generate code to call create2
// note: parameter is the deployed code, not the full contract code
// always use zero value and zero salt.
func create2(deployedCode []byte) []byte {
	contractCode := create2_contract(deployedCode)
	return createCode(
		copyToMemory(contractCode, 0),
		push(0), push(len(contractCode)), push(0), push(0), vm.CREATE2,
	)
}

func (tb *testContextBuilder) build() *testContext {
	genesis := core.DeveloperGenesisBlock(10_000_000, &common.Address{})
	genesis.Timestamp = 100
	genesisBlock := genesis.ToBlock()
	gaspool := new(core.GasPool).AddGas(genesisBlock.GasLimit())

	//TODO: fill some mock backend...
	var backend ethapi.Backend

	return &testContext{
		t:            tb.t,
		genesisAlloc: tb.genesisAlloc,
		chainContext: ethapi.NewChainContext(context.TODO(), backend),
		genesis:      genesis,
		genesisBlock: genesisBlock,
		gaspool:      gaspool,
	}
}

// add EOA account with balance
func (tt *testContextBuilder) withAccount(addr string, balance int64) *testContextBuilder {
	tt.genesisAlloc[common.HexToAddress(addr)] = types.Account{Balance: big.NewInt(balance)}
	return tt
}
func (tt *testContextBuilder) withCode(addr string, code []byte, balance int64) *testContextBuilder {
	if len(code) == 0 {
		tt.genesisAlloc[common.HexToAddress(addr)] = types.Account{
			Balance: big.NewInt(balance),
		}
	} else {
		tt.genesisAlloc[common.HexToAddress(addr)] = types.Account{
			Code:    code,
			Balance: big.NewInt(balance),
		}
	}
	return tt
}

// generate a push opcode and its following constant value
func push(n int) []byte {
	if n < 0 {
		panic("attempt to push negative")
	}
	if n < 256 {
		return createCode(vm.PUSH1, byte(n))
	}
	if n < 65536 {
		return createCode(vm.PUSH2, byte(n>>8), byte(n))
	}
	if n < 1<<32 {
		return createCode(vm.PUSH4, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	}
	panic("larger number")
}

// create code to copy data into memory at the given offset
// NOTE: if data is not in 32-byte multiples, it will override the next bytes
// used by RETURN/REVERT
func copyToMemory(data []byte, offset uint) []byte {
	ret := []byte{}
	for len(data) > 32 {
		ret = append(ret, createCode(vm.PUSH32, data[0:32], vm.PUSH2, uint16(offset), vm.MSTORE)...)
		data = data[32:]
		offset = offset + 32
	}

	if len(data) > 0 {
		//push data up, as EVM is big-endian
		v := common.RightPadBytes(data, 32)
		ret = append(ret, createCode(vm.PUSH32, v, vm.PUSH2, uint16(offset), vm.MSTORE)...)
	}
	return ret
}

// revert with given data
func revertWithData(data []byte) []byte {
	ret := append(copyToMemory(data, 0), createCode(vm.PUSH2, uint16(len(data)), vm.PUSH0, vm.REVERT)...)
	return ret
}

// generate the code to return the given byte array (up to 32 bytes)
func returnWithData(data []byte) []byte {
	ret := append(copyToMemory(data, 0), createCode(vm.PUSH2, uint16(len(data)), vm.PUSH0, vm.RETURN)...)
	return ret
}

func createAccountCode() []byte {
	return nil
}

// create EVM code from OpCode, byte and []bytes
func createCode(items ...interface{}) []byte {
	var buffer bytes.Buffer

	for _, item := range items {
		switch v := item.(type) {
		case string:
			buffer.Write(hexutils.HexToBytes(v))
		case vm.OpCode:
			buffer.WriteByte(byte(v))
		case byte:
			buffer.WriteByte(v)
		case []byte:
			buffer.Write(v)
		case int8:
			buffer.WriteByte(byte(v))
		case uint16:
			buffer.Write([]byte{byte(v >> 8), byte(v)})
		case uint32:
			buffer.Write([]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)})
		case int:
			if v >= 256 {
				panic(fmt.Errorf("int defaults to int8 (byte). use int16, etc: %v", v))
			}
			buffer.WriteByte(byte(v))
		case common.Hash:
			buffer.Write(v.Bytes())
		case common.Address:
			buffer.Write(v.Bytes())
		default:
			// should be a compile-time error...
			panic(fmt.Errorf("unsupported type: %T", v))
		}
	}

	return buffer.Bytes()
}

func asBytes32(a int) []byte {
	return common.LeftPadBytes(big.NewInt(int64(a)).Bytes(), 32)
}
