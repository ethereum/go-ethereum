package core

import (
	"fmt"
	"path"
	"testing"

	"github.com/ethereum/go-ethereum/ethutil"
)

func TestChainInsertions(t *testing.T) {
	c1, err := ethutil.ReadAllFile(path.Join("..", "_data", "chain1"))
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	data1, _ := ethutil.Decode([]byte(c1), 0)
	fmt.Println(data1)
}
