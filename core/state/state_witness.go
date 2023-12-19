package state

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"os"
	"path/filepath"
)

type Witness struct {
	block       *types.Block
	blockHashes map[uint64]common.Hash
	codes       map[common.Hash]Code
	root        common.Hash
	lists       map[common.Hash]map[string][]byte
}

func (w *Witness) GetBlockHash(num uint64) common.Hash {
	return w.blockHashes[num]
}

func (w *Witness) Root() common.Hash {
	return w.root
}

type rlpWitness struct {
	Block       *types.Block
	Root        common.Hash
	Owners      []common.Hash
	AllPaths    [][]string
	AllNodes    [][][]byte
	BlockNums   []uint64
	BlockHashes []common.Hash
	Codes       []Code
	CodeHashes  []common.Hash
}

func (e *rlpWitness) ToWitness() *Witness {
	res := NewWitness()
	res.root = e.Root
	for i := 0; i < len(e.Codes); i++ {
		res.codes[e.CodeHashes[i]] = e.Codes[i]
	}
	for i, owner := range e.Owners {
		pathMap := make(map[string][]byte)
		for j := 0; j < len(e.AllPaths[i]); j++ {
			pathMap[e.AllPaths[i][j]] = e.AllNodes[i][j]
		}
		res.lists[owner] = pathMap
	}
	for i, blockNum := range e.BlockNums {
		res.blockHashes[blockNum] = e.BlockHashes[i]
	}
	return res
}

func DecodeWitnessRLP(b []byte) (*Witness, error) {
	var res Witness
	if err := rlp.DecodeBytes(b, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (w *Witness) EncodeRLP() ([]byte, error) {
	var encWit rlpWitness
	encWit.Block = w.block

	for owner, nodeMap := range w.lists {
		encWit.Owners = append(encWit.Owners, owner)
		var ownerPaths []string
		var ownerNodes [][]byte

		for path, node := range nodeMap {
			ownerPaths = append(ownerPaths, path)
			ownerNodes = append(ownerNodes, node)
		}
		encWit.AllPaths = append(encWit.AllPaths, ownerPaths)
		encWit.AllNodes = append(encWit.AllNodes, ownerNodes)
	}

	for codeHash, code := range w.codes {
		encWit.CodeHashes = append(encWit.CodeHashes, codeHash)
		encWit.Codes = append(encWit.Codes, code)
	}

	for blockNum, blockHash := range w.blockHashes {
		encWit.BlockNums = append(encWit.BlockNums, blockNum)
		encWit.BlockHashes = append(encWit.BlockHashes, blockHash)
	}
	res, err := rlp.EncodeToBytes(&encWit)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (w *Witness) addAccessList(owner common.Hash, list map[string][]byte) {
	if len(list) > 0 {
		w.lists[owner] = list
	}
}

func (w *Witness) AddBlockHash(hash common.Hash, num uint64) {
	w.blockHashes[num] = hash
}

// TODO: don't include the code hash in the witness if not necessary
func (w *Witness) AddCode(hash common.Hash, code Code) {
	if code, ok := w.codes[hash]; ok && len(code) > 0 {
		return
	}
	w.codes[hash] = code
}

func (w *Witness) AddCodeHash(hash common.Hash) {
	if _, ok := w.codes[hash]; ok {
		return
	}
	w.codes[hash] = []byte{}
}

func (w Witness) Copy() Witness {
	panic("not implemented")
}

func (w *Witness) LogSizeWithBlock(b *types.Block) {
	enc, _ := w.EncodeRLP()
	fmt.Printf("block %d witness+block size: %d\n", b.Number(), len(enc))
}

func (w *Witness) Summary() string {
	b := new(bytes.Buffer)
	xx, err := rlp.EncodeToBytes(w.block)
	if err != nil {
		panic(err)
	}
	totBlock := len(xx)

	yy, _ := w.EncodeRLP()

	totWit := len(yy)
	totCode := 0
	for _, c := range w.codes {
		totCode += len(c)
	}
	totNodes := 0
	totPaths := 0
	nodePathCount := 0
	for _, ownerPaths := range w.lists {
		for path, node := range ownerPaths {
			nodePathCount++
			totNodes += len(node)
			totPaths += len(path)
		}
	}

	fmt.Fprintf(b, "%4d hashes: %v\n", len(w.blockHashes), common.StorageSize(len(w.blockHashes)*32))
	fmt.Fprintf(b, "%4d owners: %v\n", len(w.lists), common.StorageSize(len(w.lists)*32))
	fmt.Fprintf(b, "%4d nodes:  %v\n", nodePathCount, common.StorageSize(totNodes))
	fmt.Fprintf(b, "%4d paths:  %v\n", nodePathCount, common.StorageSize(totPaths))
	fmt.Fprintf(b, "%4d codes:  %v\n", len(w.codes), common.StorageSize(totCode))
	fmt.Fprintf(b, "%4d codeHashes: %v\n", len(w.codes), common.StorageSize(len(w.codes)*32))
	fmt.Fprintf(b, "block (%4d txs): %v\n", len(w.block.Transactions()), common.StorageSize(totBlock))
	fmt.Fprintf(b, "Total size: %v\n ", common.StorageSize(totWit))
	return b.String()
}

func (w *Witness) PopulateMemoryDB() ethdb.Database {
	db := rawdb.NewMemoryDatabase()
	for codeHash, code := range w.codes {
		rawdb.WriteCode(db, codeHash, code)
	}

	for owner, owned := range w.lists {
		for path, node := range owned {
			rawdb.WriteTrieNode(db, owner, []byte(path), common.Hash{}, node, rawdb.PathScheme)
		}
	}

	return db
}

func (w *Witness) SetBlock(b *types.Block) {
	w.block = b
}

func NewWitness() *Witness {
	return &Witness{
		block:       nil,
		blockHashes: make(map[uint64]common.Hash),
		codes:       make(map[common.Hash]Code),
		root:        common.Hash{},
		lists:       make(map[common.Hash]map[string][]byte),
	}
}

func DumpBlockWitnessToFile(w *Witness, path string) error {
	enc, _ := w.EncodeRLP()

	blockHash := w.block.Hash()
	outputFName := fmt.Sprintf("%d-%x.rlp", w.block.NumberU64(), blockHash[0:8])
	path = filepath.Join(path, outputFName)
	err := os.WriteFile(path, enc, 0644)
	if err != nil {
		return err
	}
	return nil
}
