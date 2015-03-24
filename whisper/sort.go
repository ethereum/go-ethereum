package whisper

import (
	"sort"

	"github.com/ethereum/go-ethereum/common"
)

type sortedKeys struct {
	k []int32
}

func (self *sortedKeys) Len() int           { return len(self.k) }
func (self *sortedKeys) Less(i, j int) bool { return self.k[i] < self.k[j] }
func (self *sortedKeys) Swap(i, j int)      { self.k[i], self.k[j] = self.k[j], self.k[i] }

func sortKeys(m map[int32]common.Hash) []int32 {
	sorted := new(sortedKeys)
	sorted.k = make([]int32, len(m))
	i := 0
	for key, _ := range m {
		sorted.k[i] = key
		i++
	}

	sort.Sort(sorted)

	return sorted.k
}
