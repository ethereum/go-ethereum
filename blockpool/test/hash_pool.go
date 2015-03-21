package test

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// hashPool is a test helper, that allows random hashes to be referred to by integers
type TestHashPool struct {
	intToHash
	hashToInt
	lock sync.Mutex
}

func NewHashPool() *TestHashPool {
	return &TestHashPool{intToHash: make(intToHash), hashToInt: make(hashToInt)}
}

type intToHash map[int]common.Hash

type hashToInt map[common.Hash]int

func newHash(i int) common.Hash {
	return common.BytesToHash(crypto.Sha3([]byte(string(i))))
}

func (self *TestHashPool) IndexesToHashes(indexes []int) (hashes []common.Hash) {
	self.lock.Lock()
	defer self.lock.Unlock()
	for _, i := range indexes {
		hash, found := self.intToHash[i]
		if !found {
			hash = newHash(i)
			self.intToHash[i] = hash
			self.hashToInt[hash] = i
		}
		hashes = append(hashes, hash)
	}
	return
}

func (self *TestHashPool) HashesToIndexes(hashes []common.Hash) (indexes []int) {
	self.lock.Lock()
	defer self.lock.Unlock()
	for _, hash := range hashes {
		i, found := self.hashToInt[hash]
		if !found {
			i = -1
		}
		indexes = append(indexes, i)
	}
	return
}
