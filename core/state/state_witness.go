package state

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"os"
)

type Witness struct {
	blockHashes map[uint64]common.Hash
	codes       map[common.Hash]Code
	root        common.Hash
	lists       map[common.Hash]map[string][]byte
}

func (w *Witness) EncodeRLP(b *types.Block) []byte {
	buf := new(bytes.Buffer)
	eb := rlp.NewEncoderBuffer(buf)

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
	path := "/datadrive/"
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
