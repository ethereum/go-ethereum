package core

import (
	"fmt"
	"path"
	"runtime"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/event"
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

func loadChain(fn string, t *testing.T) types.Blocks {
	c1, err := ethutil.ReadAllFile(path.Join("..", "_data", fn))
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	value := ethutil.NewValueFromBytes([]byte(c1))
	blocks := make(types.Blocks, value.Len())
	it := value.NewIterator()
	for it.Next() {
		blocks[it.Idx()] = types.NewBlockFromRlpValue(it.Value())
	}

	return blocks
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
	chain1 := loadChain("chain1", t)
	chain2 := loadChain("chain2", t)
	var eventMux event.TypeMux
	chainMan := NewChainManager(&eventMux)
	txPool := NewTxPool(chainMan, nil, &eventMux)
	blockMan := NewBlockManager(txPool, chainMan, &eventMux)
	chainMan.SetProcessor(blockMan)

	const max = 2
	done := make(chan bool, max)

	go insertChain(done, chainMan, chain1, t)
	go insertChain(done, chainMan, chain2, t)

	for i := 0; i < max; i++ {
		<-done
	}
	fmt.Println(chainMan.CurrentBlock())
}
