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
	ctrct := NewTransaction(ContractAddr, big.NewInt(200000000), []string{
		"PUSH",
		"1",
		"PUSH",
		"2",

		"STOP",
	})
	bm.ApplyTransactions(block, []*Transaction{ctrct})
}
