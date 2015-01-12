// db-backed storage layer for chunks

package bzz

import (
	"bytes"
	// "fmt"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/syndtr/goleveldb/leveldb"
	"path"
)

type dpaDBStorage struct {
	db *ethdb.LDBDatabase
}

func (s *dpaDBStorage) Put(entry *Chunk) error {

	data := ethutil.Encode([]interface{}{entry.Data, entry.Size})

	s.db.Put(entry.Key, data)

	return nil
}

func (s *dpaDBStorage) Get(hash Key) (chunk *Chunk, err error) {

	data, err := s.db.Get(hash)
	if err != nil {
		panic("hi")
	}

	dec := rlp.NewListStream(bytes.NewReader(data), uint64(len(data)))
	dec.Decode(&chunk)

	return

}

func (s *dpaDBStorage) Init() {

	dbPath := path.Join(".", "bzz")

	// Open the db
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return
	}

	s.db = &ethdb.LDBDatabase{DB: db, Comp: false}

}
