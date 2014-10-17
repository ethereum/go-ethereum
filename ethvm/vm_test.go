package ethvm

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethtrie"
	"github.com/ethereum/eth-go/ethutil"
)

type TestEnv struct {
}

func (self TestEnv) Origin() []byte        { return nil }
func (self TestEnv) BlockNumber() *big.Int { return nil }
func (self TestEnv) BlockHash() []byte     { return nil }
func (self TestEnv) PrevHash() []byte      { return nil }
func (self TestEnv) Coinbase() []byte      { return nil }
func (self TestEnv) Time() int64           { return 0 }
func (self TestEnv) Difficulty() *big.Int  { return nil }
func (self TestEnv) Value() *big.Int       { return nil }

// This is likely to fail if anything ever gets looked up in the state trie :-)
func (self TestEnv) State() *ethstate.State { return ethstate.New(ethtrie.New(nil, "")) }

const mutcode = `
var x = 0;
for i := 0; i < 10; i++ {
	x = i
}

return x`

func setup(level ethlog.LogLevel, typ Type) (*Closure, VirtualMachine) {
	code, err := ethutil.Compile(mutcode, true)
	if err != nil {
		log.Fatal(err)
	}

	// Pipe output to /dev/null
	ethlog.AddLogSystem(ethlog.NewStdLogSystem(ioutil.Discard, log.LstdFlags, level))

	ethutil.ReadConfig(".ethtest", "/tmp/ethtest", "")

	stateObject := ethstate.NewStateObject([]byte{'j', 'e', 'f', 'f'})
	callerClosure := NewClosure(nil, stateObject, stateObject, code, big.NewInt(1000000), big.NewInt(0))

	return callerClosure, New(TestEnv{}, typ)
}

func TestDebugVm(t *testing.T) {
	closure, vm := setup(ethlog.DebugLevel, DebugVmTy)
	ret, _, e := closure.Call(vm, nil)
	if e != nil {
		fmt.Println("error", e)
	}

	if ret[len(ret)-1] != 9 {
		t.Errorf("Expected VM to return 9, got", ret, "instead.")
	}
}

func TestVm(t *testing.T) {
	closure, vm := setup(ethlog.DebugLevel, StandardVmTy)
	ret, _, e := closure.Call(vm, nil)
	if e != nil {
		fmt.Println("error", e)
	}

	if ret[len(ret)-1] != 9 {
		t.Errorf("Expected VM to return 9, got", ret, "instead.")
	}
}

func BenchmarkDebugVm(b *testing.B) {
	closure, vm := setup(ethlog.InfoLevel, DebugVmTy)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		closure.Call(vm, nil)
	}
}

func BenchmarkVm(b *testing.B) {
	closure, vm := setup(ethlog.InfoLevel, StandardVmTy)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		closure.Call(vm, nil)
	}
}

func RunCode(mutCode string, typ Type) []byte {
	code, err := ethutil.Compile(mutCode, true)
	if err != nil {
		log.Fatal(err)
	}

	ethlog.AddLogSystem(ethlog.NewStdLogSystem(os.Stdout, log.LstdFlags, ethlog.InfoLevel))

	ethutil.ReadConfig(".ethtest", "/tmp/ethtest", "")

	stateObject := ethstate.NewStateObject([]byte{'j', 'e', 'f', 'f'})
	closure := NewClosure(nil, stateObject, stateObject, code, big.NewInt(1000000), big.NewInt(0))

	vm := New(TestEnv{}, typ)
	ret, _, e := closure.Call(vm, nil)
	if e != nil {
		fmt.Println(e)
	}

	return ret
}

func TestBuildInSha256(t *testing.T) {
	ret := RunCode(`
	var in = 42
	var out = 0

	call(0x2, 0, 10000, in, out)

	return out
	`, DebugVmTy)

	exp := ethcrypto.Sha256(ethutil.LeftPadBytes([]byte{42}, 32))
	if bytes.Compare(ret, exp) != 0 {
		t.Errorf("Expected %x, got %x", exp, ret)
	}
}

func TestBuildInRipemd(t *testing.T) {
	ret := RunCode(`
	var in = 42
	var out = 0

	call(0x3, 0, 10000, in, out)

	return out
	`, DebugVmTy)

	exp := ethutil.RightPadBytes(ethcrypto.Ripemd160(ethutil.LeftPadBytes([]byte{42}, 32)), 32)
	if bytes.Compare(ret, exp) != 0 {
		t.Errorf("Expected %x, got %x", exp, ret)
	}
}
