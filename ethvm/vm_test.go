package ethvm

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"testing"

	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethutil"
)

type TestEnv struct {
}

func (self TestEnv) Origin() []byte         { return nil }
func (self TestEnv) BlockNumber() *big.Int  { return nil }
func (self TestEnv) BlockHash() []byte      { return nil }
func (self TestEnv) PrevHash() []byte       { return nil }
func (self TestEnv) Coinbase() []byte       { return nil }
func (self TestEnv) Time() int64            { return 0 }
func (self TestEnv) Difficulty() *big.Int   { return nil }
func (self TestEnv) Value() *big.Int        { return nil }
func (self TestEnv) State() *ethstate.State { return nil }

const mutcode = `
var x = 0;
for i := 0; i < 10; i++ {
	x = i
}

return x`

func setup(level int, typ Type) (*Closure, VirtualMachine) {
	code, err := ethutil.Compile(mutcode, true)
	if err != nil {
		log.Fatal(err)
	}

	// Pipe output to /dev/null
	ethlog.AddLogSystem(ethlog.NewStdLogSystem(ioutil.Discard, log.LstdFlags, ethlog.LogLevel(level)))

	ethutil.ReadConfig(".ethtest", "/tmp/ethtest", "")

	stateObject := ethstate.NewStateObject([]byte{'j', 'e', 'f', 'f'})
	callerClosure := NewClosure(nil, stateObject, stateObject, code, big.NewInt(1000000), big.NewInt(0))

	return callerClosure, New(TestEnv{}, typ)
}

func TestDebugVm(t *testing.T) {
	closure, vm := setup(4, DebugVmTy)
	ret, _, e := closure.Call(vm, nil)
	if e != nil {
		fmt.Println("error", e)
	}

	if ret[len(ret)-1] != 9 {
		t.Errorf("Expected VM to return 1, got", ret, "instead.")
	}
}

func TestVm(t *testing.T) {
	closure, vm := setup(4, StandardVmTy)
	ret, _, e := closure.Call(vm, nil)
	if e != nil {
		fmt.Println("error", e)
	}

	if ret[len(ret)-1] != 9 {
		t.Errorf("Expected VM to return 1, got", ret, "instead.")
	}
}

func BenchmarkDebugVm(b *testing.B) {
	closure, vm := setup(3, DebugVmTy)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		closure.Call(vm, nil)
	}
}

func BenchmarkVm(b *testing.B) {
	closure, vm := setup(3, StandardVmTy)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		closure.Call(vm, nil)
	}
}
