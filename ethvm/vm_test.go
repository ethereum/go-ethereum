package ethvm

import (
	"fmt"
	"github.com/ethereum/eth-go/ethdb"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethutil"
	"log"
	"math/big"
	"os"
	"testing"
)

type TestEnv struct {
}

func (self TestEnv) GetObject() Object               { return nil }
func (self TestEnv) Origin() []byte                  { return nil }
func (self TestEnv) BlockNumber() *big.Int           { return nil }
func (self TestEnv) PrevHash() []byte                { return nil }
func (self TestEnv) Coinbase() []byte                { return nil }
func (self TestEnv) Time() int64                     { return 0 }
func (self TestEnv) Difficulty() *big.Int            { return nil }
func (self TestEnv) Data() []string                  { return nil }
func (self TestEnv) Value() *big.Int                 { return nil }
func (self TestEnv) GetBalance(addr []byte) *big.Int { return nil }
func (self TestEnv) State() *ethstate.State          { return nil }

func TestVm(t *testing.T) {
	ethlog.AddLogSystem(ethlog.NewStdLogSystem(os.Stdout, log.LstdFlags, ethlog.LogLevel(4)))

	db, _ := ethdb.NewMemDatabase()
	ethutil.ReadConfig(".ethtest", "/tmp/ethtest", "")
	ethutil.Config.Db = db

	stateObject := ethstate.NewStateObject([]byte{'j', 'e', 'f', 'f'})
	callerClosure := NewClosure(stateObject, stateObject, []byte{0x60, 0x01}, big.NewInt(1000000), big.NewInt(0))

	vm := New(TestEnv{})
	vm.Verbose = true

	ret, _, e := callerClosure.Call(vm, nil)
	if e != nil {
		fmt.Println("error", e)
	}
	fmt.Println(ret)
}
