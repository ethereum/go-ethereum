package state

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
)

// A Witness encompasses a block and all state necessary to compute the
// post-state root.
type Witness struct {
	Block       *types.Block
	blockHashes map[uint64]common.Hash
	codes       map[common.Hash][]byte
	root        common.Hash
	tries       map[common.Hash]map[string][]byte
	triesLock   sync.Mutex
}

// BlockHash returns the block hash corresponding to an ancestor block between 1-256 blocks old.
func (w *Witness) BlockHash(num uint64) common.Hash {
	return w.blockHashes[num]
}

// Root returns the post-state root of the witness if it has been computed or 0x00..0 if not.
func (w *Witness) Root() common.Hash {
	return w.root
}

// rlpWitness is the encoding structure for a Witness
type rlpWitness struct {
	EncBlock    []byte
	Root        common.Hash
	Owners      []common.Hash
	TriesPaths  [][]string
	TriesNodes  [][][]byte
	BlockNums   []uint64
	BlockHashes []common.Hash
	Codes       [][]byte
}

func (e *rlpWitness) toWitness() (*Witness, error) {
	res := NewWitness(e.Root)
	if err := rlp.DecodeBytes(e.EncBlock, &res.Block); err != nil {
		return nil, err
	}
	for _, code := range e.Codes {
		codeHash := crypto.Keccak256Hash(code)
		if _, ok := res.codes[codeHash]; ok {
			return nil, errors.New("duplicate code in witness")
		}
		res.codes[codeHash] = code
	}
	for i, owner := range e.Owners {
		trieNodes := make(map[string][]byte)
		for j := 0; j < len(e.TriesPaths[i]); j++ {
			trieNodes[e.TriesPaths[i][j]] = e.TriesNodes[i][j]
		}
		res.tries[owner] = trieNodes
	}
	for i, blockNum := range e.BlockNums {
		res.blockHashes[blockNum] = e.BlockHashes[i]
	}
	return res, nil
}

// DecodeWitnessRLP decodes a byte slice into a witness object.
func DecodeWitnessRLP(b []byte) (*Witness, error) {
	var res rlpWitness
	if err := rlp.DecodeBytes(b, &res); err != nil {
		return nil, err
	}
	return res.toWitness()
}

// EncodeRLP encodes a witness object into bytes.  The Witness' state root is
// zeroed before the encoding.  The encoding is not deterministic (the result
// can differ for the same Witness)
func (w *Witness) EncodeRLP() ([]byte, error) {
	var encWit rlpWitness
	var encBlock bytes.Buffer
	if err := w.Block.EncodeRLPWithZeroRoot(&encBlock); err != nil {
		return nil, err
	}
	encWit.EncBlock = encBlock.Bytes()

	for owner, trie := range w.tries {
		encWit.Owners = append(encWit.Owners, owner)
		var ownerPaths []string
		var ownerNodes [][]byte

		for path, node := range trie {
			ownerPaths = append(ownerPaths, path)
			ownerNodes = append(ownerNodes, node)
		}
		encWit.TriesPaths = append(encWit.TriesPaths, ownerPaths)
		encWit.TriesNodes = append(encWit.TriesNodes, ownerNodes)
	}

	for _, code := range w.codes {
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

// addAccessList associates a map of RLP-encoded trie nodes keyed by path to
// an owner in the witness.  the witness takes ownership of the passed map. It
// is safe to call this method concurrently.
func (w *Witness) addAccessList(owner common.Hash, newTrieNodes map[string][]byte) {
	var trie map[string][]byte

	if len(newTrieNodes) == 0 {
		return
	}
	w.triesLock.Lock()
	defer w.triesLock.Unlock()

	trie, ok := w.tries[owner]
	if !ok {
		trie = make(map[string][]byte)
		w.tries[owner] = trie
	}
	for path, node := range newTrieNodes {
		trie[path] = node
	}
}

// AddBlockHash adds a block hash/number to the witness
func (w *Witness) AddBlockHash(hash common.Hash, num uint64) {
	w.blockHashes[num] = hash
}

// AddCode associates a hash with code in the Witness.
// The Witness takes ownership over the passed code slice.
func (w *Witness) AddCode(hash common.Hash, code []byte) {
	if hash == types.EmptyCodeHash || hash == (common.Hash{}) || len(code) == 0 {
		return
	}
	w.codes[hash] = code
}

// Summary prints a human-readable summary containing the total size of the
// witness and the sizes of the underlying components
func (w *Witness) Summary() string {
	b := new(bytes.Buffer)
	xx, err := rlp.EncodeToBytes(w.Block)
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
	for _, ownerPaths := range w.tries {
		for path, node := range ownerPaths {
			nodePathCount++
			totNodes += len(node)
			totPaths += len(path)
		}
	}

	fmt.Fprintf(b, "%4d hashes: %v\n", len(w.blockHashes), common.StorageSize(len(w.blockHashes)*32))
	fmt.Fprintf(b, "%4d owners: %v\n", len(w.tries), common.StorageSize(len(w.tries)*32))
	fmt.Fprintf(b, "%4d nodes:  %v\n", nodePathCount, common.StorageSize(totNodes))
	fmt.Fprintf(b, "%4d paths:  %v\n", nodePathCount, common.StorageSize(totPaths))
	fmt.Fprintf(b, "%4d codes:  %v\n", len(w.codes), common.StorageSize(totCode))
	fmt.Fprintf(b, "%4d codeHashes: %v\n", len(w.codes), common.StorageSize(len(w.codes)*32))
	fmt.Fprintf(b, "block (%4d txs): %v\n", len(w.Block.Transactions()), common.StorageSize(totBlock))
	fmt.Fprintf(b, "Total size: %v\n ", common.StorageSize(totWit))
	return b.String()
}

// Copy deep-copies the witness object.  Witness.Block isn't deep-copied as it
// is never mutated by Witness
func (w *Witness) Copy() *Witness {
	var res Witness
	res.Block = w.Block

	for blockNr, blockHash := range w.blockHashes {
		res.blockHashes[blockNr] = blockHash
	}
	for codeHash, code := range w.codes {
		cpy := make([]byte, len(code))
		copy(cpy, code)
		res.codes[codeHash] = cpy
	}
	res.root = w.root
	for owner, owned := range w.tries {
		res.tries[owner] = make(map[string][]byte)
		for path, node := range owned {
			cpy := make([]byte, len(node))
			copy(cpy, node)
			res.tries[owner][path] = cpy
		}
	}
	return &res
}

// sortedWitness encodes returns an rlpWitness where hash-map items are sorted lexicographically by key
// in the encoder object to ensure that the encoded bytes are always the same for a given witness.
func (w *Witness) sortedWitness() *rlpWitness {
	var (
		sortedCodeHashes []common.Hash
		owners           []common.Hash
		ownersPaths      [][]string
		ownersNodes      [][][]byte
		blockNrs         []uint64
		blockHashes      []common.Hash
		codeHashes       []common.Hash
		codes            [][]byte
	)
	for key := range w.codes {
		sortedCodeHashes = append(sortedCodeHashes, key)
	}
	sort.Slice(sortedCodeHashes, func(i, j int) bool {
		return bytes.Compare(sortedCodeHashes[i][:], sortedCodeHashes[j][:]) > 0
	})

	// sort the list of owners
	for owner := range w.tries {
		owners = append(owners, owner)
	}
	sort.Slice(owners, func(i, j int) bool {
		return bytes.Compare(owners[i][:], owners[j][:]) > 0
	})

	// sort the trie nodes of each trie by path
	for _, owner := range owners {
		nodes := w.tries[owner]
		var ownerPaths []string
		for path := range nodes {
			ownerPaths = append(ownerPaths, path)
		}
		sort.Strings(ownerPaths)

		var ownerNodes [][]byte
		for _, path := range ownerPaths {
			ownerNodes = append(ownerNodes, nodes[path])
		}
		ownersPaths = append(ownersPaths, ownerPaths)
		ownersNodes = append(ownersNodes, ownerNodes)
	}

	for blockNr, blockHash := range w.blockHashes {
		blockNrs = append(blockNrs, blockNr)
		blockHashes = append(blockHashes, blockHash)
	}

	for codeHash := range w.codes {
		codeHashes = append(codeHashes, codeHash)
	}
	sort.Slice(codeHashes, func(i, j int) bool {
		return bytes.Compare(codeHashes[i][:], codeHashes[j][:]) > 0
	})

	for _, codeHash := range codeHashes {
		codes = append(codes, w.codes[codeHash])
	}

	encBlock, _ := rlp.EncodeToBytes(w.Block)
	return &rlpWitness{
		EncBlock:    encBlock,
		Root:        common.Hash{},
		Owners:      owners,
		TriesPaths:  ownersPaths,
		TriesNodes:  ownersNodes,
		BlockNums:   blockNrs,
		BlockHashes: blockHashes,
		Codes:       codes,
	}
}

// PrettyPrint displays the contents of a witness object in a human-readable format to standard output.
func (w *Witness) PrettyPrint() string {
	sorted := w.sortedWitness()
	b := new(bytes.Buffer)
	fmt.Fprintf(b, "block: %+v\n", w.Block)
	fmt.Fprintf(b, "root: %x\n", sorted.Root)
	fmt.Fprint(b, "owners:\n")
	for i, owner := range sorted.Owners {
		if owner == (common.Hash{}) {
			fmt.Fprintf(b, "\troot:\n")
		} else {
			fmt.Fprintf(b, "\t%x:\n", owner)
		}
		ownerPaths := sorted.TriesPaths[i]
		ownerNodes := sorted.TriesNodes[i]
		for j, path := range ownerPaths {
			fmt.Fprintf(b, "\t\t%x:%x\n", []byte(path), ownerNodes[j])
		}
	}
	fmt.Fprintf(b, "block hashes:\n")
	for i, blockNum := range sorted.BlockNums {
		blockHash := sorted.BlockHashes[i]
		fmt.Fprintf(b, "\t%d:%x\n", blockNum, blockHash)
	}
	fmt.Fprintf(b, "codes:\n")
	for _, code := range sorted.Codes {
		hash := crypto.Keccak256Hash(code)
		fmt.Fprintf(b, "\t%x:%x\n", hash, code)
	}
	return b.String()
}

// Hash returns the sha256 hash of a Witness
func (w *Witness) Hash() common.Hash {
	res, err := rlp.EncodeToBytes(w.sortedWitness())
	if err != nil {
		panic(err)
	}

	return sha256.Sum256(res[:])
}

// NewWitness returns a new Witness object.
func NewWitness(root common.Hash) *Witness {
	return &Witness{
		Block:       nil,
		blockHashes: make(map[uint64]common.Hash),
		codes:       make(map[common.Hash][]byte),
		root:        root,
		tries:       make(map[common.Hash]map[string][]byte),
	}
}

// PopulateDB imports tries,codes and block hashes from the witness
// into the specified path-based backing db.
func (w *Witness) PopulateDB(db ethdb.Database) error {
	batch := db.NewBatch()
	for owner, nodes := range w.tries {
		for path, node := range nodes {
			if owner == (common.Hash{}) {
				rawdb.WriteAccountTrieNode(batch, []byte(path), node)
			} else {
				rawdb.WriteStorageTrieNode(batch, owner, []byte(path), node)
			}
		}
	}
	for codeHash, code := range w.codes {
		rawdb.WriteCode(batch, codeHash, code)
	}
	if err := batch.Write(); err != nil {
		return err
	}
	return nil
}
