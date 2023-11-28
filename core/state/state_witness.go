package state

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type witness struct {
	root   common.Hash
	lists  []map[string][]byte
	owners []common.Hash
}

func newWitness(originalRoot common.Hash) *witness {
	return &witness{root: originalRoot}
}

func (w *witness) addAccessList(owner common.Hash, list map[string][]byte) {
	//fmt.Printf("Adding owner %x len %d\n", owner, len(list))
	if len(list) > 0 {
		w.lists = append(w.lists, list)
		w.owners = append(w.owners, owner)
	}
}

func (w *witness) Dump() {
	fmt.Printf("[witness] Root %x\n", w.root)
	for i, list := range w.lists {
		owner := w.owners[i]
		fmt.Printf("[witness] Owner %#x, %d entries: \n", owner, len(list))
		for path, v := range list {
			fmt.Printf("[witness] - '%#x': %#x\n", path, v)
		}
	}
}
