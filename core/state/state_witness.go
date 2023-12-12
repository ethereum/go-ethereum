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
)

type Witness struct {
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

type encodedWitness struct {
	block       types.Block
	root        common.Hash
	owners      []common.Hash
	allPaths    [][]string
	allNodes    [][][]byte
	blockNums   []uint64
	blockHashes []common.Hash
	codes       []Code
	codeHashes  []common.Hash
}

func (e *encodedWitness) ToWitness() *Witness {
	var res Witness
	res.root = e.root
	for i := 0; i < len(e.codes); i++ {
		res.codes[e.codeHashes[i]] = e.codes[i]
	}
	for i, owner := range e.owners {
		pathMap := make(map[string][]byte)
		for j := 0; j < len(e.allPaths[i]); j++ {
			pathMap[e.allPaths[i][j]] = e.allNodes[i][j]
		}
		res.lists[owner] = pathMap
	}
	for i, blockNum := range e.blockNums {
		res.blockHashes[blockNum] = e.blockHashes[i]
	}
	return &res
}

func DecodeWitnessRLP(b []byte) (*types.Block, *Witness, error) {
	var res encodedWitness
	if err := rlp.DecodeBytes(b, &res); err != nil {
		return nil, nil, err
	}
	return &res.block, res.ToWitness(), nil
}

func (w *Witness) EncodeRLP(b *types.Block) []byte {
	buf := new(bytes.Buffer)
	eb := rlp.NewEncoderBuffer(buf)

	var root common.Hash
	var owners []common.Hash
	var allPaths [][]string
	var allNodes [][][]byte
	var blockNums []uint64
	var blockHashes []common.Hash
	var codes []Code
	var codeHashes []common.Hash

	for owner, nodeMap := range w.lists {
		owners = append(owners, owner)
		var ownerPaths []string
		var ownerNodes [][]byte

		for path, node := range nodeMap {
			ownerPaths = append(ownerPaths, path)
			ownerNodes = append(ownerNodes, node)
		}
		allPaths = append(allPaths, ownerPaths)
		allNodes = append(allNodes, ownerNodes)
	}

	for codeHash, code := range w.codes {
		codeHashes = append(codeHashes, codeHash)
		codes = append(codes, code)
	}

	for blockNum, blockHash := range w.blockHashes {
		blockNums = append(blockNums, blockNum)
		blockHashes = append(blockHashes, blockHash)
	}
	l := eb.List()
	b.EncodeRLP(eb)
	if err := rlp.Encode(eb, root); err != nil {
		panic(err)
	}
	if err := rlp.Encode(eb, owners); err != nil {
		panic(err)
	}
	if err := rlp.Encode(eb, allPaths); err != nil {
		panic(err)
	}
	if err := rlp.Encode(eb, allNodes); err != nil {
		panic(err)
	}
	if err := rlp.Encode(eb, blockNums); err != nil {
		panic(err)
	}
	if err := rlp.Encode(eb, blockHashes); err != nil {
		panic(err)
	}
	if err := rlp.Encode(eb, codeHashes); err != nil {
		panic(err)
	}
	if err := rlp.Encode(eb, codes); err != nil {
		panic(err)
	}
	eb.ListEnd(l)
	eb.Flush()
	return buf.Bytes()
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

func (w *Witness) Dump() {
	for owner, al := range w.lists {
		fmt.Printf("owner %x:\n", owner)
		for path, node := range al {
			fmt.Printf("%x: %x\n", []byte(path), node)
		}
	}
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

func NewWitness() *Witness {
	return &Witness{
		make(map[uint64]common.Hash),
		make(map[common.Hash]Code),
		common.Hash{},
		make(map[common.Hash]map[string][]byte),
	}
}
func DumpBlockWithWitnessToFile(w *Witness, b *types.Block) {
	enc := w.EncodeRLP(b)
	path, _ := os.Getwd() //"/datadrive/"
	err := os.MkdirAll(fmt.Sprintf("%s/block-dump", path), 0755)
	if err != nil {
		panic("shite2")
	}
	outputFName := fmt.Sprintf("%d-%x.rlp", b.NumberU64(), b.Hash())
	err = os.WriteFile(path+"/block-dump/"+outputFName, enc, 0644)
	if err != nil {
		panic("shite 3")
	}
}
