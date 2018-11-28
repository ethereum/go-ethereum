package statediff

import (
	"encoding/json"
	"math/big"
	"sort"
	"strings"

	// "github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

type StateDiffBuilder struct {
	chainDb ethdb.Database
}

type StateDiff struct {
	BlockNumber     big.Int                                   `json:"blockNumber"      gencodec:"required"`
	BlockHash       common.Hash                               `json:"blockHash"        gencodec:"required"`
	CreatedAccounts map[common.Address]AccountDiffEventual    `json:"createdAccounts"  gencodec:"required"`
	DeletedAccounts map[common.Address]AccountDiffEventual    `json:"deletedAccounts" gencodec:"required"`
	UpdatedAccounts map[common.Address]AccountDiffIncremental `json:"updatedAccounts"  gencodec:"required"`

	encoded []byte
	err     error
}

func (self *StateDiff) ensureEncoded() {
	if self.encoded == nil && self.err == nil {
		self.encoded, self.err = json.Marshal(self)
	}
}

// Implement Encoder interface for StateDiff
func (self *StateDiff) Length() int {
	self.ensureEncoded()
	return len(self.encoded)
}

// Implement Encoder interface for StateDiff
func (self *StateDiff) Encode() ([]byte, error) {
	self.ensureEncoded()
	return self.encoded, self.err
}

type AccountDiffEventual struct {
	Nonce        diffUint64            `json:"nonce"         gencodec:"required"`
	Balance      diffBigInt            `json:"balance"       gencodec:"required"`
	Code         string                `json:"code"          gencodec:"required"`
	CodeHash     string                `json:"codeHash"      gencodec:"required"`
	ContractRoot diffString            `json:"contractRoot"  gencodec:"required"`
	Storage      map[string]diffString `json:"storage"       gencodec:"required"`
}

type AccountDiffIncremental struct {
	Nonce        diffUint64            `json:"nonce"         gencodec:"required"`
	Balance      diffBigInt            `json:"balance"       gencodec:"required"`
	CodeHash     string                `json:"codeHash"      gencodec:"required"`
	ContractRoot diffString            `json:"contractRoot"  gencodec:"required"`
	Storage      map[string]diffString `json:"storage"       gencodec:"required"`
}

type diffString struct {
	NewValue *string `json:"newValue"  gencodec:"optional"`
	OldValue *string `json:"oldValue"  gencodec:"optional"`
}
type diffUint64 struct {
	NewValue *uint64 `json:"newValue"  gencodec:"optional"`
	OldValue *uint64 `json:"oldValue"  gencodec:"optional"`
}
type diffBigInt struct {
	NewValue *big.Int `json:"newValue"  gencodec:"optional"`
	OldValue *big.Int `json:"oldValue"  gencodec:"optional"`
}

func NewStateDiffBuilder(db ethdb.Database) (*StateDiffBuilder, error) {
	return &StateDiffBuilder{
		chainDb: db,
	}, nil
}

func newTrieDb(chainDb ethdb.Database) *trie.Database {
	return trie.NewDatabase(chainDb)
}

func (self *StateDiffBuilder) CreateStateDiff(oldStateRoot common.Hash, newStateRoot common.Hash, blockNumber big.Int, blockHash common.Hash) (*StateDiff, error) {
	//they're building a trie based on the old hash
	//and then a trie based on the new hash
	//but the tries that they're creating require a leveldb connection - not sure how to do this
	//yet without interacting directly with level db
	//what is being used in the trie struct pkg
	//maybe it can just be something/anything that implements that
	//db interface, like even a connection to VDB?
	log.Debug("Creating StateDiff", "oldRoot", oldStateRoot.Hex(), "newRoot", newStateRoot.Hex())
	trieDb := newTrieDb(self.chainDb)
	oldTrie, errOld := trie.New(oldStateRoot, trieDb)
	if errOld != nil {
		log.Error("Error in creating Trie", "stateroot", oldStateRoot.Hex())
		return nil, errOld
	}

	newTrie, errNew := trie.New(newStateRoot, trieDb)
	if errNew != nil {
		log.Error("Error in creating Trie", "stateroot", newStateRoot.Hex())
		return nil, errNew
	}

	oldIt := oldTrie.NodeIterator(make([]byte, 0))
	newIt := newTrie.NodeIterator(make([]byte, 0))
	creations, errCreate := self.collectDiffNodes(oldIt, newIt)
	log.Debug("Creation Accounts", "creations", creations)
	if errCreate != nil {
		log.Error("Error in finding Created statediffs", "error", errCreate)
		return nil, errCreate
	}

	oldIt = oldTrie.NodeIterator(make([]byte, 0))
	newIt = newTrie.NodeIterator(make([]byte, 0))
	deletions, errDel := self.collectDiffNodes(newIt, oldIt)
	if errDel != nil {
		log.Error("Error in finding deleted statediffs", "error", errDel)
		return nil, errDel
	}

	createKeys := sortKeys(&creations)
	deleteKeys := sortKeys(&deletions)
	updatedKeys := findIntersection(createKeys, deleteKeys)

	// build statediff
	updatedAccounts, errUDiffs := self.buildDiffIncremental(&creations, &deletions, &updatedKeys)
	if errUDiffs != nil {
		log.Error("Error in finding updated statediffs", "error", errDel)
		return nil, errUDiffs
	}
	log.Debug("updated accounts", "updated", updatedAccounts)

	createdAccounts, errCDiffs := self.buildDiffEventual(&creations, true)
	if errCDiffs != nil {
		return nil, errCDiffs
	}
	log.Debug("created accounts", "created", createdAccounts)

	deletedAccounts, errDDiffs := self.buildDiffEventual(&deletions, false)
	if errDDiffs != nil {
		return nil, errDDiffs
	}
	log.Debug("deleted accounts", "deleted", deletedAccounts)

	return &StateDiff{
		BlockNumber:     blockNumber,
		BlockHash:       blockHash,
		CreatedAccounts: createdAccounts,
		DeletedAccounts: deletedAccounts,
		UpdatedAccounts: updatedAccounts,
	}, nil
}

func (self *StateDiffBuilder) buildDiffEventual(accounts *map[common.Address]*state.Account, created bool) (map[common.Address]AccountDiffEventual, error) {
	accountDiffs := make(map[common.Address]AccountDiffEventual)
	for addr, val := range *accounts {
		sr := val.Root
		if storageDiffs, err := self.buildStorageDiffsEventual(sr, created); err != nil {
			log.Error("Failed building eventual storage diffs", "Address", val, "error", err)
			return nil, err
		} else {
			code := ""
			codeBytes, errGetCode := self.chainDb.Get(val.CodeHash)
			if errGetCode == nil && len(codeBytes) != 0 {
				code = common.ToHex(codeBytes)
			} else {
				log.Debug("No code field.", "codehash", val.CodeHash, "Address", val, "error", errGetCode)
			}
			codeHash := common.ToHex(val.CodeHash)
			if created {
				nonce := diffUint64{
					NewValue: &val.Nonce,
				}

				balance := diffBigInt{
					NewValue: val.Balance,
				}

				hexRoot := val.Root.Hex()
				contractRoot := diffString{
					NewValue: &hexRoot,
				}
				accountDiffs[addr] = AccountDiffEventual{
					Nonce:        nonce,
					Balance:      balance,
					CodeHash:     codeHash,
					Code:         code,
					ContractRoot: contractRoot,
					Storage:      storageDiffs,
				}
			} else {
				nonce := diffUint64{
					OldValue: &val.Nonce,
				}
				balance := diffBigInt{
					OldValue: val.Balance,
				}
				hexRoot := val.Root.Hex()
				contractRoot := diffString{
					OldValue: &hexRoot,
				}
				accountDiffs[addr] = AccountDiffEventual{
					Nonce:        nonce,
					Balance:      balance,
					CodeHash:     codeHash,
					ContractRoot: contractRoot,
					Storage:      storageDiffs,
				}
			}
		}
	}
	return accountDiffs, nil
}

func (self *StateDiffBuilder) buildDiffIncremental(creations *map[common.Address]*state.Account, deletions *map[common.Address]*state.Account, updatedKeys *[]string) (map[common.Address]AccountDiffIncremental, error) {
	updatedAccounts := make(map[common.Address]AccountDiffIncremental)
	for _, val := range *updatedKeys {
		createdAcc := (*creations)[common.HexToAddress(val)]
		deletedAcc := (*deletions)[common.HexToAddress(val)]
		oldSR := deletedAcc.Root
		newSR := createdAcc.Root
		if storageDiffs, err := self.buildStorageDiffsIncremental(oldSR, newSR); err != nil {
			log.Error("Failed building storage diffs", "Address", val, "error", err)
			return nil, err
		} else {
			nonce := diffUint64{
				NewValue: &createdAcc.Nonce,
				OldValue: &deletedAcc.Nonce,
			}

			balance := diffBigInt{
				NewValue: createdAcc.Balance,
				OldValue: deletedAcc.Balance,
			}
			codeHash := common.ToHex(createdAcc.CodeHash)

			nHexRoot := createdAcc.Root.Hex()
			oHexRoot := deletedAcc.Root.Hex()
			contractRoot := diffString{
				NewValue: &nHexRoot,
				OldValue: &oHexRoot,
			}

			updatedAccounts[common.HexToAddress(val)] = AccountDiffIncremental{
				Nonce:        nonce,
				Balance:      balance,
				CodeHash:     codeHash,
				ContractRoot: contractRoot,
				Storage:      storageDiffs,
			}
			delete(*creations, common.HexToAddress(val))
			delete(*deletions, common.HexToAddress(val))
		}
	}
	return updatedAccounts, nil
}

func (self *StateDiffBuilder) buildStorageDiffsEventual(sr common.Hash, creation bool) (map[string]diffString, error) {
	log.Debug("Storage Root For Eventual Diff", "root", sr.Hex())
	trieDb := newTrieDb(self.chainDb)
	sTrie, errSt := trie.New(sr, trieDb)
	if errSt != nil {
		return nil, errSt
	}
	it := sTrie.NodeIterator(make([]byte, 0))
	storageDiffs := make(map[string]diffString)
	for {
		log.Debug("Iterating over state at path ", "path", pathToStr(it))
		if it.Leaf() {
			log.Debug("Found leaf in storage", "path", pathToStr(it))
			path := pathToStr(it)
			value := common.ToHex(it.LeafBlob())
			if creation {
				storageDiffs[path] = diffString{NewValue: &value}
			} else {
				storageDiffs[path] = diffString{OldValue: &value}
			}
		}
		cont := it.Next(true)
		if !cont {
			break
		}
	}
	return storageDiffs, nil
}

func (self *StateDiffBuilder) buildStorageDiffsIncremental(oldSR common.Hash, newSR common.Hash) (map[string]diffString, error) {
	log.Debug("Storage Roots for Incremental Diff", "old", oldSR.Hex(), "new", newSR.Hex())
	trieDb := newTrieDb(self.chainDb)
	oldTrie, errStOld := trie.New(oldSR, trieDb)
	if errStOld != nil {
		return nil, errStOld
	}
	newTrie, errStNew := trie.New(newSR, trieDb)
	if errStNew != nil {
		return nil, errStNew
	}

	oldIt := oldTrie.NodeIterator(make([]byte, 0))
	newIt := newTrie.NodeIterator(make([]byte, 0))
	it, _ := trie.NewDifferenceIterator(oldIt, newIt)
	storageDiffs := make(map[string]diffString)
	for {
		if it.Leaf() {
			log.Debug("Found leaf in storage", "path", pathToStr(it))
			path := pathToStr(it)
			value := common.ToHex(it.LeafBlob())
			if oldVal, err := oldTrie.TryGet(it.LeafKey()); err != nil {
				log.Error("Failed to look up value in oldTrie", "path", path, "error", err)
			} else {
				hexOldVal := common.ToHex(oldVal)
				storageDiffs[path] = diffString{OldValue: &hexOldVal, NewValue: &value}
			}
		}

		cont := it.Next(true)
		if !cont {
			break
		}
	}
	return storageDiffs, nil
}

func (self *StateDiffBuilder) collectDiffNodes(a, b trie.NodeIterator) (map[common.Address]*state.Account, error) {
	var diffAccounts = make(map[common.Address]*state.Account)
	it, _ := trie.NewDifferenceIterator(a, b)
	for {
		log.Debug("Current Path and Hash", "path", pathToStr(it), "hashold", common.Hash(it.Hash()))
		if it.Leaf() {

			// lookup address
			path := make([]byte, len(it.Path())-1)
			copy(path, it.Path())
			addr, errLookUp := self.addressByPath(path)
			if errLookUp != nil {
				log.Error("Error looking up address via path", "path", path, "error", errLookUp)
				return nil, errLookUp
			}

			// lookup account state
			var account state.Account
			if errAcc := rlp.DecodeBytes(it.LeafBlob(), &account); errAcc != nil {
				log.Error("Error looking up account via address", "address", addr, "error", errAcc)
				return nil, errAcc
			}

			// record account to creations
			log.Debug("Account lookup successful", "address", addr, "account", account)
			diffAccounts[*addr] = &account
		}
		cont := it.Next(true)
		if !cont {
			break
		}
	}
	return diffAccounts, nil
}

func (self *StateDiffBuilder) addressByPath(path []byte) (*common.Address, error) {
	// db := core.PreimageTable(self.chainDb)
	log.Debug("Looking up address from path", "path", common.ToHex(append([]byte("secure-key-"), path...)))
	// if addrBytes,err := db.Get(path); err != nil {
	if addrBytes, err := self.chainDb.Get(append([]byte("secure-key-"), hexToKeybytes(path)...)); err != nil {
		log.Error("Error looking up address via path", "path", common.ToHex(append([]byte("secure-key-"), path...)), "error", err)
		return nil, err
	} else {
		addr := common.BytesToAddress(addrBytes)
		log.Debug("Address found", "Address", addr)
		return &addr, nil
	}

}

func sortKeys(data *map[common.Address]*state.Account) []string {
	var keys []string
	for key, _ := range *data {
		keys = append(keys, key.Hex())
	}
	sort.Strings(keys)
	return keys
}

func findIntersection(a, b []string) []string {
	length_a := len(a)
	length_b := len(b)
	i_a, i_b := 0, 0
	updates := make([]string, 0)
	if i_a >= length_a || i_b >= length_b {
		return updates
	}
	for {
		switch strings.Compare(a[i_a], b[i_b]) {
		// a[i_a] < b[i_b]
		case -1:
			i_a++
			if i_a >= length_a {
				return updates
			}
			break
			// a[i_a] == b[i_b]
		case 0:
			updates = append(updates, a[i_a])
			i_a++
			i_b++
			if i_a >= length_a || i_b >= length_b {
				return updates
			}
			break
			// a[i_a] > b[i_b]
		case 1:
			i_b++
			if i_b >= length_b {
				return updates
			}
			break
		}
	}
}

func pathToStr(it trie.NodeIterator) string {
	path := it.Path()
	if hasTerm(path) {
		path = path[:len(path)-1]
	}
	nibblePath := ""
	for i, v := range common.ToHex(path) {
		if i%2 == 0 && i > 1 {
			continue
		}
		nibblePath = nibblePath + string(v)
	}
	return nibblePath
}

// Duplicated from trie/encoding.go

func hexToKeybytes(hex []byte) []byte {
	if hasTerm(hex) {
		hex = hex[:len(hex)-1]
	}
	if len(hex)&1 != 0 {
		panic("can't convert hex key of odd length")
	}
	key := make([]byte, (len(hex)+1)/2)
	decodeNibbles(hex, key)
	return key
}

func decodeNibbles(nibbles []byte, bytes []byte) {
	for bi, ni := 0, 0; ni < len(nibbles); bi, ni = bi+1, ni+2 {
		bytes[bi] = nibbles[ni]<<4 | nibbles[ni+1]
	}
}

// prefixLen returns the length of the common prefix of a and b.
func prefixLen(a, b []byte) int {
	var i, length = 0, len(a)
	if len(b) < length {
		length = len(b)
	}
	for ; i < length; i++ {
		if a[i] != b[i] {
			break
		}
	}
	return i
}

// hasTerm returns whether a hex key has the terminator flag.
func hasTerm(s []byte) bool {
	return len(s) > 0 && s[len(s)-1] == 16
}