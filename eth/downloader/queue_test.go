package downloader

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"gopkg.in/fatih/set.v0"
)

func createHashSet(hashes []common.Hash) *set.Set {
	hset := set.New()

	for _, hash := range hashes {
		hset.Add(hash)
	}

	return hset
}

func createBlocksFromHashSet(hashes *set.Set) []*types.Block {
	blocks := make([]*types.Block, hashes.Size())

	var i int
	hashes.Each(func(v interface{}) bool {
		blocks[i] = createBlock(i, common.Hash{}, v.(common.Hash))
		i++
		return true
	})

	return blocks
}
