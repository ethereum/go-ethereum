package types

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/rlp"
)

func init() {
	ethutil.ReadConfig(".ethtest", "/tmp/ethtest", "")
	ethutil.Config.Db, _ = ethdb.NewMemDatabase()
}

func TestNewBlock(t *testing.T) {
	block := GenesisBlock()
	data := ethutil.Encode(block)

	var genesis Block
	err := rlp.Decode(bytes.NewReader(data), &genesis)
}
