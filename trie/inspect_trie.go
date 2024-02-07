package trie

import (
	"bytes"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/sync/semaphore"
)

type Inspector struct {
	trie           *Trie // traverse trie
	db             *Database
	stateRootHash  common.Hash
	blocknum       uint64
	root           node // root of triedb
	totalNum       uint64
	wg             sync.WaitGroup
	statLock       sync.RWMutex
	result         map[string]*trieTreeStat
	sem            *semaphore.Weighted
	eoaAccountNums uint64
}

type trieTreeStat struct {
	isAccountTrie      bool
	theNodeStatByLevel [15]nodeStat
	totalNodeStat      nodeStat
}

type nodeStat struct {
	ShortNodeCnt atomic.Uint64
	FullNodeCnt  atomic.Uint64
	ValueNodeCnt atomic.Uint64
}

func (ns *nodeStat) IsEmpty() bool {
	if ns.FullNodeCnt.Load() == 0 && ns.ShortNodeCnt.Load() == 0 && ns.ValueNodeCnt.Load() == 0 {
		return true
	}
	return false
}

func (trieStat *trieTreeStat) AtomicAdd(theNode node, height uint32) {
	switch (theNode).(type) {
	case *shortNode:
		trieStat.totalNodeStat.ShortNodeCnt.Add(1)
		trieStat.theNodeStatByLevel[height].ShortNodeCnt.Add(1)
	case *fullNode:
		trieStat.totalNodeStat.FullNodeCnt.Add(1)
		trieStat.theNodeStatByLevel[height].FullNodeCnt.Add(1)
	case valueNode:
		trieStat.totalNodeStat.ValueNodeCnt.Add(1)
		trieStat.theNodeStatByLevel[height].ValueNodeCnt.Add(1)
	default:
		panic(errors.New("invalid node type for statistics"))
	}
}

func (trieStat *trieTreeStat) Display(ownerAddress string, treeType string) string {
	sw := new(strings.Builder)
	table := tablewriter.NewWriter(sw)
	table.SetHeader([]string{"-", "Level", "ShortNodeCnt", "FullNodeCnt", "ValueNodeCnt"})
	if ownerAddress == "" {
		table.SetCaption(true, fmt.Sprintf("%v", treeType))
	} else {
		table.SetCaption(true, fmt.Sprintf("%v-%v", treeType, ownerAddress))
	}
	table.SetAlignment(1)
	for i := 0; i < len(trieStat.theNodeStatByLevel); i++ {
		ns := &trieStat.theNodeStatByLevel[i]
		if ns.IsEmpty() {
			break

		}
		table.AppendBulk([][]string{
			{"-", fmt.Sprintf("%d", i), fmt.Sprintf("%d", ns.ShortNodeCnt.Load()), fmt.Sprintf("%d", ns.FullNodeCnt.Load()), fmt.Sprintf("%d", ns.ValueNodeCnt.Load())},
		})
	}
	table.AppendBulk([][]string{
		{"Total", "-", fmt.Sprintf("%d", trieStat.totalNodeStat.ShortNodeCnt.Load()), fmt.Sprintf("%d", trieStat.totalNodeStat.FullNodeCnt.Load()), fmt.Sprintf("%d", trieStat.totalNodeStat.ValueNodeCnt.Load())},
	})
	table.Render()
	return sw.String()
}

// NewInspector return an inspector obj
func NewInspector(tr *Trie, db *Database, stateRootHash common.Hash, blocknum uint64, jobnum uint64) (*Inspector, error) {
	if tr == nil {
		return nil, errors.New("trie is nil")
	}

	if tr.root == nil {
		return nil, errors.New("trie root is nil")
	}

	ins := &Inspector{
		trie:           tr,
		db:             db,
		stateRootHash:  stateRootHash,
		blocknum:       blocknum,
		root:           tr.root,
		result:         make(map[string]*trieTreeStat),
		totalNum:       (uint64)(0),
		wg:             sync.WaitGroup{},
		sem:            semaphore.NewWeighted(int64(jobnum)),
		eoaAccountNums: 0,
	}

	return ins, nil
}

// Run statistics, external call
func (inspect *Inspector) Run() {
	accountTrieStat := &trieTreeStat{
		isAccountTrie: true,
	}

	if _, ok := inspect.result[""]; !ok {
		inspect.result[""] = accountTrieStat
	}
	log.Info("Find Account Trie Tree", "rootHash: ", inspect.trie.Hash().String(), "BlockNum: ", inspect.blocknum)

	inspect.concurrentTraversal(inspect.trie, accountTrieStat, inspect.root, 0, []byte{})
	inspect.wg.Wait()
}

func (inspect *Inspector) concurrentTraversal(theTrie *Trie, theTrieTreeStat *trieTreeStat, theNode node, height uint32, path []byte) {
	// print process progress
	totalNum := atomic.AddUint64(&inspect.totalNum, 1)
	if totalNum%100000 == 0 {
		fmt.Printf("Complete progress: %v, go routines Num: %v\n", totalNum, runtime.NumGoroutine())
	}

	// nil node
	if theNode == nil {
		return
	}

	switch current := (theNode).(type) {
	case *shortNode:
		inspect.concurrentTraversal(theTrie, theTrieTreeStat, current.Val, height, append(path, current.Key...))
	case *fullNode:
		for idx, child := range current.Children {
			if child == nil {
				continue
			}
			childPath := append(path, byte(idx))
			if inspect.sem.TryAcquire(1) {
				inspect.wg.Add(1)
				dst := make([]byte, len(childPath))
				copy(dst, childPath)
				go func() {
					inspect.concurrentTraversal(theTrie, theTrieTreeStat, theNode, height, path)
					inspect.wg.Done()
				}()
			} else {
				inspect.concurrentTraversal(theTrie, theTrieTreeStat, child, height+1, childPath)
			}
		}
	case hashNode:
		n, err := theTrie.resolveWithoutTrack(current, path)
		if err != nil {
			fmt.Printf("Resolve HashNode error: %v, TrieRoot: %v, Height: %v, Path: %v\n", err, theTrie.Hash().String(), height+1, path)
			return
		}
		inspect.concurrentTraversal(theTrie, theTrieTreeStat, n, height, path)
		return
	case valueNode:
		if !hasTerm(path) {
			break
		}
		var account types.StateAccount
		if err := rlp.Decode(bytes.NewReader(current), &account); err != nil {
			break
		}
		if common.BytesToHash(account.CodeHash) == types.EmptyCodeHash {
			inspect.eoaAccountNums++
		}
		if account.Root == (common.Hash{}) || account.Root == types.EmptyRootHash {
			break
		}
		ownerAddress := common.BytesToHash(hexToCompact(path))
		contractTrie, err := New(StorageTrieID(inspect.stateRootHash, ownerAddress, account.Root), inspect.db)
		if err != nil {
			fmt.Printf("New contract trie node: %v, error: %v, Height: %v, Path: %v\n", theNode, err, height, path)
			break
		}
		contractTrie.tracer.reset()
		trieStat := &trieTreeStat{
			isAccountTrie: false,
		}

		inspect.statLock.Lock()
		if _, ok := inspect.result[ownerAddress.String()]; !ok {
			inspect.result[ownerAddress.String()] = trieStat
		}
		inspect.statLock.Unlock()

		// log.Info("Find Contract Trie Tree, rootHash: ", contractTrie.Hash().String(), "")
		inspect.wg.Add(1)
		go func() {
			inspect.concurrentTraversal(contractTrie, trieStat, contractTrie.root, 0, []byte{})
			inspect.wg.Done()
		}()
	default:
		panic(errors.New("invalid node type for traverse"))
	}
	theTrieTreeStat.AtomicAdd(theNode, height)
}

func (inspect *Inspector) DisplayResult() {
	// display root hash
	if _, ok := inspect.result[""]; !ok {
		log.Info("Display result error", "missing account trie")
		return
	}
	fmt.Printf(inspect.result[""].Display("", "AccountTrie"))

	type sortedTrie struct {
		totalNum     uint64
		ownerAddress string
	}
	// display contract trie
	var sortedTriesByNums []sortedTrie
	var totalContactsNodeStat nodeStat
	var contractTrieCnt uint64 = 0

	for ownerAddress, stat := range inspect.result {
		if ownerAddress == "" {
			continue
		}
		contractTrieCnt++
		totalContactsNodeStat.ShortNodeCnt.Add(stat.totalNodeStat.ShortNodeCnt.Load())
		totalContactsNodeStat.FullNodeCnt.Add(stat.totalNodeStat.FullNodeCnt.Load())
		totalContactsNodeStat.ValueNodeCnt.Add(stat.totalNodeStat.ValueNodeCnt.Load())
		totalNodeCnt := stat.totalNodeStat.ShortNodeCnt.Load() + stat.totalNodeStat.ValueNodeCnt.Load() + stat.totalNodeStat.FullNodeCnt.Load()
		sortedTriesByNums = append(sortedTriesByNums, sortedTrie{totalNum: totalNodeCnt, ownerAddress: ownerAddress})
	}
	sort.Slice(sortedTriesByNums, func(i, j int) bool {
		return sortedTriesByNums[i].totalNum > sortedTriesByNums[j].totalNum
	})
	fmt.Println("EOA accounts num: ", inspect.eoaAccountNums)
	// only display top 5
	for i, t := range sortedTriesByNums {
		if i > 5 {
			break
		}
		if stat, ok := inspect.result[t.ownerAddress]; !ok {
			log.Error("Storage trie stat not found", "ownerAddress", t.ownerAddress)
		} else {
			fmt.Printf(stat.Display(t.ownerAddress, "ContractTrie"))
		}
	}
	fmt.Printf("Contract Trie, total trie num: %v, ShortNodeCnt: %v, FullNodeCnt: %v, ValueNodeCnt: %v\n",
		contractTrieCnt, totalContactsNodeStat.ShortNodeCnt.Load(), totalContactsNodeStat.FullNodeCnt.Load(), totalContactsNodeStat.ValueNodeCnt.Load())
}
