package core

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rlp"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	ethutil.ReadConfig("/tmp/ethtest", "/tmp/ethtest", "ETH")
}

func reset() {
	db, err := ethdb.NewMemDatabase()
	if err != nil {
		panic("Could not create mem-db, failing")
	}
	ethutil.Config.Db = db
}

func loadChain(fn string, t *testing.T) (types.Blocks, error) {
	fh, err := os.OpenFile(path.Join(os.Getenv("GOPATH"), "src", "github.com", "ethereum", "go-ethereum", "_data", fn), os.O_RDONLY, os.ModePerm)
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
	reset()

	chain1, err := loadChain("valid1", t)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	chain2, err := loadChain("valid2", t)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	var eventMux event.TypeMux
	chainMan := NewChainManager(&eventMux)
	txPool := NewTxPool(&eventMux)
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

func TestChainMultipleInsertions(t *testing.T) {
	reset()

	const max = 4
	chains := make([]types.Blocks, max)
	var longest int
	for i := 0; i < max; i++ {
		var err error
		name := "valid" + strconv.Itoa(i+1)
		chains[i], err = loadChain(name, t)
		if len(chains[i]) >= len(chains[longest]) {
			longest = i
		}
		fmt.Println("loaded", name, "with a length of", len(chains[i]))
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
	}
	var eventMux event.TypeMux
	chainMan := NewChainManager(&eventMux)
	txPool := NewTxPool(&eventMux)
	blockMan := NewBlockManager(txPool, chainMan, &eventMux)
	chainMan.SetProcessor(blockMan)
	done := make(chan bool, max)
	for i, chain := range chains {
		// XXX the go routine would otherwise reference the same (chain[3]) variable and fail
		i := i
		chain := chain
		go func() {
			insertChain(done, chainMan, chain, t)
			fmt.Println(i, "done")
		}()
	}

	for i := 0; i < max; i++ {
		<-done
	}

	if !reflect.DeepEqual(chains[longest][len(chains[longest])-1], chainMan.CurrentBlock()) {
		t.Error("Invalid canonical chain")
	}
}
