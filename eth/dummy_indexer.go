package eth

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// TODO only for testing; should be removed in the final version
type dummyIndexer struct {
	head, needBlocksFrom uint64
}

func (d *dummyIndexer) AddBlockData(headers []*types.Header, receipts []types.Receipts) (bool, common.Range[uint64]) {
	fmt.Print("*** AddBlockData")
	for _, header := range headers {
		num := header.Number.Uint64()
		d.head = max(d.head, num)
		fmt.Print(" ", num)
		if d.needBlocksFrom == 0 {
			d.needBlocksFrom = num - 100000
		}
		if d.needBlocksFrom == num {
			d.needBlocksFrom++
		}
	}
	fmt.Println()
	if len(receipts) > 0 && len(receipts[0]) > 0 {
		fmt.Println("    receipt:", receipts[0][0])
	}
	return d.status()
}

func (d *dummyIndexer) Revert(blockNumber uint64) {
	d.head = blockNumber
	fmt.Println("*** Revert", blockNumber)
}

func (d *dummyIndexer) Status() (bool, common.Range[uint64]) {
	fmt.Println("*** Status")
	return d.status()
}

func (d *dummyIndexer) status() (ready bool, needBlocks common.Range[uint64]) {
	ready = (time.Now().Unix()/5)&1 == 1
	if d.head >= d.needBlocksFrom {
		needBlocks = common.NewRange[uint64](d.needBlocksFrom, d.head+1-d.needBlocksFrom)
	}
	fmt.Println("    ready", ready, "needBlocks", needBlocks)
	return
}

func (*dummyIndexer) MissingBlocks(missing common.Range[uint64]) {
	fmt.Println("*** MissingBlocks", missing)
}

func (*dummyIndexer) Stop() {
	fmt.Println("*** Stop")
}
