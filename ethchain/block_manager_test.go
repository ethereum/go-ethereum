package ethchain

import (
	_ "fmt"
	"github.com/ethereum/eth-go/ethdb"
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
	"testing"
)

func TestVm(t *testing.T) {
	InitFees()
	ethutil.ReadConfig("")

	db, _ := ethdb.NewMemDatabase()
	ethutil.Config.Db = db
	bm := NewBlockManager(nil)

	block := bm.bc.genesisBlock
	script := Compile([]string{
		"PUSH",
		"1",
		"PUSH",
		"2",
	})
	tx := NewTransaction(ContractAddr, big.NewInt(200000000), script)
	addr := tx.Hash()[12:]
	bm.ApplyTransactions(block, []*Transaction{tx})

	tx2 := NewTransaction(addr, big.NewInt(1e17), nil)
	tx2.Sign([]byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	bm.ApplyTransactions(block, []*Transaction{tx2})
}
