package vm

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/trie"
	checker "gopkg.in/check.v1"
	// "github.com/obscuren/mutan"
)

type VmSuite struct{}

var _ = checker.Suite(&VmSuite{})
var big9 = ethutil.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000009")

const mutcode = `
var x = 0;
for i := 0; i < 10; i++ {
	x = i
}

return x`

type TestEnv struct{}

func (TestEnv) Origin() []byte        { return nil }
func (TestEnv) BlockNumber() *big.Int { return nil }
func (TestEnv) BlockHash() []byte     { return nil }
func (TestEnv) PrevHash() []byte      { return nil }
func (TestEnv) Coinbase() []byte      { return nil }
func (TestEnv) Time() int64           { return 0 }
func (TestEnv) GasLimit() *big.Int    { return nil }
func (TestEnv) Difficulty() *big.Int  { return nil }
func (TestEnv) Value() *big.Int       { return nil }
func (TestEnv) AddLog(*state.Log)     {}
func (TestEnv) Transfer(from, to Account, amount *big.Int) error {
	return nil
}

// This is likely to fail if anything ever gets looked up in the state trie :-)
func (TestEnv) State() *state.State {
	return state.New(trie.New(nil, ""))
}

func setup(level logger.LogLevel, typ Type) (*Closure, VirtualMachine) {
	code, err := ethutil.Compile(mutcode, true)
	if err != nil {
		log.Fatal(err)
	}

	// Pipe output to /dev/null
	logger.AddLogSystem(logger.NewStdLogSystem(ioutil.Discard, log.LstdFlags, level))

	ethutil.ReadConfig(".ethtest", "/tmp/ethtest", "")

	stateObject := state.NewStateObject([]byte{'j', 'e', 'f', 'f'})
	callerClosure := NewClosure(nil, stateObject, stateObject, code, big.NewInt(1000000), big.NewInt(0))

	return callerClosure, New(TestEnv{}, typ)
}

func (s *VmSuite) TestDebugVm(c *checker.C) {
	// if mutan.Version < "0.6" {
	// 	t.Skip("skipping for mutan version", mutan.Version, " < 0.6")
	// }
	closure, vm := setup(logger.DebugLevel, DebugVmTy)
	ret, _, e := closure.Call(vm, nil)
	c.Assert(e, checker.NotNil)
	c.Skip("Depends on mutan")
	c.Assert(ret, checker.DeepEquals, big9)
}

func (s *VmSuite) TestVm(c *checker.C) {
	// if mutan.Version < "0.6" {
	// 	t.Skip("skipping for mutan version", mutan.Version, " < 0.6")
	// }
	closure, vm := setup(logger.DebugLevel, StandardVmTy)
	ret, _, e := closure.Call(vm, nil)
	c.Assert(e, checker.NotNil)
	c.Skip("Depends on mutan")
	c.Assert(ret, checker.DeepEquals, big9)
}

func (s *VmSuite) BenchmarkDebugVm(c *checker.C) {
	closure, vm := setup(logger.InfoLevel, StandardVmTy)

	c.ResetTimer()

	for i := 0; i < c.N; i++ {
		closure.Call(vm, nil)
	}
}

func (s *VmSuite) BenchmarkVm(c *checker.C) {
	closure, vm := setup(logger.InfoLevel, DebugVmTy)

	c.ResetTimer()

	for i := 0; i < c.N; i++ {
		closure.Call(vm, nil)
	}
}

func RunCode(mutCode string, typ Type) []byte {
	code, err := ethutil.Compile(mutCode, true)
	if err != nil {
		log.Fatal(err)
	}

	logger.AddLogSystem(logger.NewStdLogSystem(os.Stdout, log.LstdFlags, logger.InfoLevel))

	ethutil.ReadConfig(".ethtest", "/tmp/ethtest", "")

	stateObject := state.NewStateObject([]byte{'j', 'e', 'f', 'f'})
	closure := NewClosure(nil, stateObject, stateObject, code, big.NewInt(1000000), big.NewInt(0))

	vm := New(TestEnv{}, typ)
	ret, _, e := closure.Call(vm, nil)
	if e != nil {
		fmt.Println(e)
	}

	return ret
}

func (s *VmSuite) TestBuildInSha256(c *checker.C) {
	ret := RunCode(`
	var in = 42
	var out = 0

	call(0x2, 0, 10000, in, out)

	return out
	`, DebugVmTy)

	exp := crypto.Sha256(ethutil.LeftPadBytes([]byte{42}, 32))
	c.Skip("Depends on mutan")
	c.Assert(ret, checker.DeepEquals, exp)
}

func (s *VmSuite) TestBuildInRipemd(c *checker.C) {
	ret := RunCode(`
	var in = 42
	var out = 0

	call(0x3, 0, 10000, in, out)

	return out
	`, DebugVmTy)

	exp := ethutil.RightPadBytes(crypto.Ripemd160(ethutil.LeftPadBytes([]byte{42}, 32)), 32)
	c.Skip("Depends on mutan")
	c.Assert(ret, checker.DeepEquals, exp)
}

func (s *VmSuite) TestOog(c *checker.C) {
	// This tests takes a long time and will eventually run out of gas
	// t.Skip()
	c.Skip("This tests takes a long time and will eventually run out of gas")

	logger.AddLogSystem(logger.NewStdLogSystem(os.Stdout, log.LstdFlags, logger.InfoLevel))

	ethutil.ReadConfig(".ethtest", "/tmp/ethtest", "")

	stateObject := state.NewStateObject([]byte{'j', 'e', 'f', 'f'})
	closure := NewClosure(nil, stateObject, stateObject, ethutil.Hex2Bytes("60ff60ff600057"), big.NewInt(1000000), big.NewInt(0))

	vm := New(TestEnv{}, DebugVmTy)
	_, _, e := closure.Call(vm, nil)
	if e != nil {
		fmt.Println(e)
	}
}
