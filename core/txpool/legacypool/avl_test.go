package legacypool

import (
	"math/big"
	"math/rand"
	"sort"
	"testing"
)

const (
	opAdd = iota
	opRemove
	opSearch
)

const maxKey = 100
const nops = 100000

func TestTree(t *testing.T) {
	for j := 0; j < 100; j++ {
		//t.Logf("------------------Test %d--------------------", j)
		rand.Seed(int64(j))
		tree := &AVLTree{}
		m := make(map[uint64]*big.Int)

		for i := 0; i < nops; i++ {
			op := rand.Intn(3)
			k := uint64(rand.Intn(maxKey))

			switch op {
			case opAdd:
				v := int64(rand.Int())
				// t.Logf("Insert %d: %d", k, v)
				tree.Add(uint64(k), big.NewInt(v))
				m[k] = big.NewInt(v)
			case opRemove:
				tree.Remove(k)
				// t.Logf("Remove %d", k)
				delete(m, k)

			case opSearch:
				tv := big.NewInt(0)
				// t.Logf("Search %d", k)
				node, sum := tree.Search(k)
				tok := node != nil
				if tok {
					tv = node.value
				}

				mv := m[k]
				if mv == nil {
					mv = big.NewInt(0)
				}
				if tv.Cmp(mv) != 0 {
					t.Fatalf("Incorrect value for key %d, want: %d, got: %d", k, mv, tv)
				}

				var msum = big.NewInt(0)

				for key, value := range m {
					if key <= k {
						msum.Add(msum, value)
					}
				}

				if sum.Cmp(msum) != 0 {
					t.Fatalf("Incorrect sum for key %d, want: %d, got: %d", k, msum, sum)
				}
			}
		}

		nodes := tree.Flatten()
		keys := make([]uint64, 0)
		for key := range m {
			keys = append(keys, uint64(key))
		}
		sort.Slice(keys, func(i, j2 int) bool {
			return keys[i] < keys[j2]
		})

		if len(keys) != len(nodes) {
			t.Fatalf("Incorrect number of nodes, want: %d, got: %d", len(keys), len(nodes))
		}

		for i := 0; i < len(keys); i++ {
			if keys[i] != nodes[i].key {
				t.Fatalf("Incorrect key, want: %d, got: %d", keys[i], nodes[i].key)
			}
		}
	}
	// tree := &AVLTree{}
	// tree.Add(4, big.NewInt(7))
	// tree.Add(8, big.NewInt(8))
	// tree.Search(8)
}
