package core

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"runtime"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rlp"
	//logpkg "github.com/ethereum/go-ethereum/logger"
)

//var Logger logpkg.LogSystem
//var Log = logpkg.NewLogger("TEST")

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	//Logger = logpkg.NewStdLogSystem(os.Stdout, log.LstdFlags, logpkg.InfoLevel)
	//logpkg.AddLogSystem(Logger)

	ethutil.ReadConfig("/tmp/ethtest", "/tmp/ethtest", "ETH")

	db, err := ethdb.NewMemDatabase()
	if err != nil {
		panic("Could not create mem-db, failing")
	}
	ethutil.Config.Db = db
}

func loadChain(fn string, t *testing.T) (types.Blocks, error) {
	fh, err := os.OpenFile(path.Join("..", "_data", fn), os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	var chain types.Blocks
	if err := rlp.Decode(fh, &chain); err != nil {
		return nil, err
	}

	return chain, nil
}

func insertChain(done chan bool, chainMan *ChainManager, chain types.Blocks, t *testing.T) {
	err := chainMan.InsertChain(chain)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	done <- true
}

func TestChainInsertions(t *testing.T) {
	chain1, err := loadChain("chain1", t)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	chain2, err := loadChain("chain2", t)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	var eventMux event.TypeMux
	chainMan := NewChainManager(&eventMux)
	txPool := NewTxPool(chainMan, &eventMux)
	blockMan := NewBlockManager(txPool, chainMan, &eventMux)
	chainMan.SetProcessor(blockMan)

	const max = 2
	done := make(chan bool, max)

	go insertChain(done, chainMan, chain1, t)
	go insertChain(done, chainMan, chain2, t)

	for i := 0; i < max; i++ {
		<-done
	}

	if reflect.DeepEqual(chain2[len(chain2)-1], chainMan.CurrentBlock()) {
		t.Error("chain2 is canonical and shouldn't be")
	}

	if !reflect.DeepEqual(chain1[len(chain1)-1], chainMan.CurrentBlock()) {
		t.Error("chain1 isn't canonical and should be")
	}
}
