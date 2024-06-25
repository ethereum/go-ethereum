package rip7560

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/status-im/keycard-go/hexutils"
	"math/big"
	"testing"
)

const DEFAULT_SENDER = "0x1111111111222222222233333333334444444444"

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

func (tb *testContextBuilder) build() *testContext {
	genesis := core.DeveloperGenesisBlock(10_000_000, &common.Address{})
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

// generate the code to return the given byte array (up to 32 bytes)
func returnData(data []byte) []byte {
	//couldn't get geth to support PUSH0 ...
	datalen := len(data)
	if datalen == 0 {
		data = []byte{0}
	}
	if datalen > 32 {
		panic(fmt.Errorf("data length is too big %v", data))
	}

	PUSHn := byte(int(vm.PUSH0) + datalen)
	ret := createCode(PUSHn, data, vm.PUSH1, 0, vm.MSTORE, vm.PUSH1, 32, vm.PUSH1, 0, vm.RETURN)
	return ret
}

// create bytecode for account
func createAccountCode() []byte {
	magic := big.NewInt(0xbf45c166)
	magic.Lsh(magic, 256-32)

	return returnData(magic.Bytes())
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
		case int:
			if v >= 256 {
				panic(fmt.Errorf("int defaults to int8 (byte). int16, etc: %v", v))
			}
			buffer.WriteByte(byte(v))
		default:
			// should be a compile-time error...
			panic(fmt.Errorf("unsupported type: %T", v))
		}
	}

	return buffer.Bytes()
}
