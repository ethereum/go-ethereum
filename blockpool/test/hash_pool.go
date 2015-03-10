package test

import (
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
)

// test helpers
// TODO: move into common test helper package (see p2p/crypto etc.)

func NewHashPool() *TestHashPool {
	return &TestHashPool{intToHash: make(intToHash), hashToInt: make(hashToInt)}
}

type intToHash map[int][]byte

type hashToInt map[string]int

// hashPool is a test helper, that allows random hashes to be referred to by integers
type TestHashPool struct {
	intToHash
	hashToInt
	lock sync.Mutex
}

func newHash(i int) []byte {
	return crypto.Sha3([]byte(string(i)))
}

func (self *TestHashPool) IndexesToHashes(indexes []int) (hashes [][]byte) {
	self.lock.Lock()
	defer self.lock.Unlock()
	for _, i := range indexes {
		hash, found := self.intToHash[i]
		if !found {
			hash = newHash(i)
			self.intToHash[i] = hash
			self.hashToInt[string(hash)] = i
		}
		hashes = append(hashes, hash)
	}
	return
}

func (self *TestHashPool) HashesToIndexes(hashes [][]byte) (indexes []int) {
	self.lock.Lock()
	defer self.lock.Unlock()
	for _, hash := range hashes {
		i, found := self.hashToInt[string(hash)]
		if !found {
			i = -1
		}
		indexes = append(indexes, i)
	}
	return
}
