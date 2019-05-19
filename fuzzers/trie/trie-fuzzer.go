package trie

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/trie"
)

// randTest performs random trie operations.
// Instances of this test are created by Generate.
type randTest []randTestStep

type randTestStep struct {
	op    int
	key   []byte // for opUpdate, opDelete, opGet
	value []byte // for opUpdate
	err   error  // for debugging
}

type proofDb struct{}

func (proofDb) Put(key []byte, value []byte) error {
	return nil
}

func (proofDb) Delete(key []byte) error {
	return nil
}

const (
	opUpdate = iota
	opDelete
	opGet
	opCommit
	opHash
	opReset
	opItercheckhash
	opProve
	opMax // boundary value, not an actual op
)

type dataSource struct {
	input  []byte
	reader *bytes.Reader
}

func newDataSource(input []byte) *dataSource {
	return &dataSource{
		input, bytes.NewReader(input),
	}
}
func (ds *dataSource) ReadByte() byte {
	if b, err := ds.reader.ReadByte(); err != nil {
		return 0
	} else {
		return b
	}
}
func (ds *dataSource) Read(buf []byte) (int, error) {
	return ds.reader.Read(buf)
}
func (ds *dataSource) Ended() bool {
	return ds.reader.Len() == 0
}

func Generate(input []byte) randTest {

	var allKeys [][]byte
	r := newDataSource(input)
	genKey := func() []byte {

		if len(allKeys) < 2 || r.ReadByte() < 0x0f {
			// new key
			key := make([]byte, r.ReadByte()%50)
			r.Read(key)
			allKeys = append(allKeys, key)
			return key
		}
		// use existing key
		return allKeys[int(r.ReadByte())%len(allKeys)]
	}

	var steps randTest

	for i := 0; !r.Ended(); i++ {

		step := randTestStep{op: int(r.ReadByte()) % opMax}
		switch step.op {
		case opUpdate:
			step.key = genKey()
			step.value = make([]byte, 8)
			binary.BigEndian.PutUint64(step.value, uint64(i))
		case opGet, opDelete, opProve:
			step.key = genKey()
		}
		steps = append(steps, step)
		if len(steps) > 500 {
			break
		}
	}

	//fmt.Printf("steps len %d input len %d\n", len(steps), len(input))
	return steps
}

func Fuzz(input []byte) int {
	program := Generate(input)
	if len(program) == 0{
		return -1
	}
	if err := runRandTest(program); err != nil {
		panic(err)
	}
	return 0
}

func runRandTest(rt randTest) error {

	triedb := trie.NewDatabase(memorydb.New())

	tr, _ := trie.New(common.Hash{}, triedb)
	values := make(map[string]string) // tracks content of the trie

	for i, step := range rt {
		switch step.op {
		case opUpdate:
			tr.Update(step.key, step.value)
			values[string(step.key)] = string(step.value)
		case opDelete:
			tr.Delete(step.key)
			delete(values, string(step.key))
		case opGet:
			v := tr.Get(step.key)
			want := values[string(step.key)]
			if string(v) != want {
				rt[i].err = fmt.Errorf("mismatch for key 0x%x, got 0x%x want 0x%x", step.key, v, want)
			}
		case opCommit:
			_, rt[i].err = tr.Commit(nil)
		case opHash:
			tr.Hash()
		case opReset:
			hash, err := tr.Commit(nil)
			if err != nil {
				return err
			}
			newtr, err := trie.New(hash, triedb)
			if err != nil {
				return err
			}
			tr = newtr
		case opItercheckhash:
			checktr, _ := trie.New(common.Hash{}, triedb)
			it := trie.NewIterator(tr.NodeIterator(nil))
			for it.Next() {
				checktr.Update(it.Key, it.Value)
			}
			if tr.Hash() != checktr.Hash() {
				return fmt.Errorf("hash mismatch in opItercheckhash")
			}
		case opProve:
			rt[i].err = tr.Prove(step.key, 0, proofDb{})
		}
		// Abort the test on error.
		if rt[i].err != nil {
			return rt[i].err
		}
	}
	return nil
}
