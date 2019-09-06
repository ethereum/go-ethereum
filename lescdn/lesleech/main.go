package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"strconv"

	"github.com/bsipos/thist"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	target  = flag.Int("target", 16, "Target number of nodes to pack into a tile")
	limit   = flag.Int("limit", 256, "Maximum number of nodes to pack into a tile")
	barrier = flag.Int("barrier", 2, "Trie depth barrier to start tiles at")
)

func main() {
	flag.Parse()

	rpc, err := ethclient.Dial("http://127.0.0.1:8545")
	if err != nil {
		panic(err)
	}
	number, err := strconv.Atoi(flag.Args()[0])
	if err != nil {
		panic(err)
	}
	previous := make(map[common.Hash]common.StorageSize)
	for ; ; number++ {
		// Crawl the entire state trie
		header, err := rpc.HeaderByNumber(context.Background(), big.NewInt(int64(number)))
		if err != nil {
			panic(err)
		}
		queue := prque.New(nil)
		queue.Push(header.Root, 0)

		var (
			tiles   int
			storage common.StorageSize
		)
		depths := make(map[common.Hash]int)

		tileHists := make(map[int]*thist.Hist)
		tileCounts := make(map[int]int)
		tileStorage := make(map[int]common.StorageSize)

		current := make(map[common.Hash]common.StorageSize)
		for !queue.Empty() {
			hash, prio := queue.Pop()
			root, depth := hash.(common.Hash), -prio

			// Read the next tile and dump some statistics
			nodes, size := fetchTile(root)
			current[root] = common.StorageSize(size)

			storage += common.StorageSize(size)
			tiles++

			tileStorage[int(depth)] += common.StorageSize(size)
			tileCounts[int(depth)]++
			if _, ok := tileHists[int(depth)]; !ok {
				tileHists[int(depth)] = thist.NewHist(nil, "", "auto", -1, true)
			}
			tileHists[int(depth)].Update(float64(size))

			log.Printf("D=%d R=0x%x: %d nodes, %v (%d tiles, %v total)", depth, root, len(nodes), common.StorageSize(size), tiles, storage)

			// Decode the tile and continue expansion
			pulled := make(map[common.Hash]struct{})
			for _, node := range nodes {
				pulled[crypto.Keccak256Hash(node)] = struct{}{}
			}
			for _, node := range nodes {
				trie.IterateRefs(node, func(path []byte, child common.Hash) error {
					depths[child] = depths[crypto.Keccak256Hash(node)] + len(path)
					if _, ok := pulled[child]; !ok {
						queue.Push(child, -int64(depths[child]))
					}
					return nil
				})
			}
		}
		// Report any tile stats
		fmt.Println("\nTile stats:")
		for i := 0; i < 128; i++ {
			if tileCounts[i] > 0 {
				fmt.Printf("  Depth %d: %d tiles, %v\n", i, tileCounts[i], tileStorage[i])
				fmt.Println(tileHists[i].Draw())
			}
		}
		// Compare the current tileset with the previous one and report the diff
		var (
			addSize, dupSize, delSize    common.StorageSize
			addCount, dupCount, delCount int
		)
		for hash, size := range current {
			if _, ok := previous[hash]; ok {
				dupSize += size
				dupCount++
			} else {
				addSize += size
				addCount++
			}
		}
		for hash, size := range previous {
			if _, ok := current[hash]; !ok {
				delSize += size
				delCount++
			}
		}
		previous = current

		fmt.Printf("Block %d: Added %d(%v), removed %d(%v), retained %d(%v)\n", number, addCount, addSize, delCount, delSize, dupCount, dupSize)
	}
}

// fetchTile retrieves a tile rooted at a certain trie hash node, also returning
// the number of bytes the transfered raw data consisted of.
func fetchTile(root common.Hash) ([][]byte, int) {
	res, err := http.Get(fmt.Sprintf("http://127.0.0.1:8548/state/0x%x?target=%d&limit=%d&barrier=%d", root, *target, *limit, *barrier))
	if err != nil {
		panic(err)
	}
	blob, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	res.Body.Close()

	var nodes [][]byte
	if err := rlp.DecodeBytes(blob, &nodes); err != nil {
		panic(err)
	}
	return nodes, len(blob)
}
