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
	"time"

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

type tileInfo struct {
	Depth int
	Size  common.StorageSize
	Refs  []common.Hash
}

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
	previous := make(map[common.Hash]*tileInfo)
	for ; ; number++ {
		// Crawl the entire state trie
		header, err := rpc.HeaderByNumber(context.Background(), big.NewInt(int64(number)))
		if err != nil {
			if err.Error() == "not found" { // Super ugly, good enough for a tester
				time.Sleep(15 * time.Second)
				number--
				continue
			}
			panic(err)
		}
		queue := prque.New(nil)
		queue.Push(header.Root, 0)

		var (
			tiles   int
			storage common.StorageSize
		)
		depths := make(map[common.Hash]int)

		tileCounts := make(map[int]int)
		tileStorage := make(map[int]common.StorageSize)

		current := make(map[common.Hash]*tileInfo)
		for !queue.Empty() {
			hash, prio := queue.Pop()
			root, depth := hash.(common.Hash), -prio

			// If the node is already crawled in the previous run, short circuit
			if _, crawled := previous[root]; crawled {
				// Iterate over the entire subtrie (ugly queue, don't care) and visit
				for skipset := []common.Hash{root}; len(skipset) > 0; skipset = skipset[1:] {
					if _, ok := current[skipset[0]]; ok {
						continue
					}
					current[skipset[0]] = previous[skipset[0]]
					for _, ref := range current[skipset[0]].Refs {
						skipset = append(skipset, ref)
					}
					storage += current[skipset[0]].Size
					tiles++

					tileStorage[current[skipset[0]].Depth] += current[skipset[0]].Size
					tileCounts[current[skipset[0]].Depth]++
				}
				// Skip recrawling this subtrie
				continue
			}
			// Read the next tile and dump some statistics
			nodes, size := fetchTile(root)
			current[root] = &tileInfo{
				Depth: int(depth),
				Size:  common.StorageSize(size),
			}
			storage += current[root].Size
			tiles++

			tileStorage[current[root].Depth] += current[root].Size
			tileCounts[current[root].Depth]++

			// Decode the tile and continue expansion
			pulled := make(map[common.Hash]struct{})
			for _, node := range nodes {
				pulled[crypto.Keccak256Hash(node)] = struct{}{}
			}
			for _, node := range nodes {
				trie.IterateRefs(node, func(path []byte, child common.Hash) error {
					depths[child] = depths[crypto.Keccak256Hash(node)] + len(path)
					if _, ok := pulled[child]; !ok {
						current[root].Refs = append(current[root].Refs, child)
						queue.Push(child, -int64(depths[child]))
					}
					return nil
				})
			}
			log.Printf("D=%d R=0x%x: %d nodes, %d refs, %v (%d tiles, %v total)", depth, root, len(nodes), len(current[root].Refs), current[root].Size, tiles, storage)
		}
		// Compare the current tileset with the previous one and report the diff
		var (
			addSize, dupSize, delSize    common.StorageSize
			addCount, dupCount, delCount int

			addSubSizes  = make(map[int]common.StorageSize)
			dupSubSizes  = make(map[int]common.StorageSize)
			delSubSizes  = make(map[int]common.StorageSize)
			addSubCounts = make(map[int]int)
			dupSubCounts = make(map[int]int)
			delSubCounts = make(map[int]int)
		)
		for hash, info := range current {
			if _, ok := previous[hash]; ok {
				dupSubSizes[info.Depth] += info.Size
				dupSubCounts[info.Depth]++

				dupSize += info.Size
				dupCount++
			} else {
				addSubSizes[info.Depth] += info.Size
				addSubCounts[info.Depth]++

				addSize += info.Size
				addCount++
			}
		}
		for hash, info := range previous {
			if _, ok := current[hash]; !ok {
				delSubSizes[info.Depth] += info.Size
				delSubCounts[info.Depth]++

				delSize += info.Size
				delCount++
			}
		}
		previous = current

		log.Printf("Block %d: Added %d(%v), removed %d(%v), retained %d(%v)", number, addCount, addSize, delCount, delSize, dupCount, dupSize)
		for i := 0; i < 128; i++ {
			if addSubCounts[i] > 0 || delSubCounts[i] > 0 || dupSubCounts[i] > 0 || tileCounts[i] > 0 {
				log.Printf("  Depth %d: Added %d(%v), removed %d(%v), retained %d(%v), total %d(%v)", i, addSubCounts[i], addSubSizes[i], delSubCounts[i], delSubSizes[i], dupSubCounts[i], dupSubSizes[i], tileCounts[i], tileStorage[i])
			}
		}
		log.Println()
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
