package ethdb

import (
	"fmt"

	"github.com/ethereum/go-ethereum/compression/rle"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
)

type LDBDatabase struct {
	db   *leveldb.DB
	comp bool
}

func NewLDBDatabase(file string) (*LDBDatabase, error) {
	// Open the db
	db, err := leveldb.OpenFile(file, nil)
	if err != nil {
		return nil, err
	}
	database := &LDBDatabase{db: db, comp: true}
	return database, nil
}

func (self *LDBDatabase) Put(key []byte, value []byte) {
	if self.comp {
		value = rle.Compress(value)
	}

	err := self.db.Put(key, value, nil)
	if err != nil {
		fmt.Println("Error put", err)
	}
}

func (self *LDBDatabase) Get(key []byte) ([]byte, error) {
	dat, err := self.db.Get(key, nil)
	if err != nil {
		return nil, err
	}

	if self.comp {
		return rle.Decompress(dat)
	}

	return dat, nil
}

func (self *LDBDatabase) Delete(key []byte) error {
	return self.db.Delete(key, nil)
}

func (self *LDBDatabase) LastKnownTD() []byte {
	data, _ := self.Get([]byte("LTD"))

	if len(data) == 0 {
		data = []byte{0x0}
	}

	return data
}

func (self *LDBDatabase) NewIterator() iterator.Iterator {
	return self.db.NewIterator(nil, nil)
}

func (self *LDBDatabase) Write(batch *leveldb.Batch) error {
	return self.db.Write(batch, nil)
}

func (self *LDBDatabase) Close() {
	// Close the leveldb database
	self.db.Close()
}

func (self *LDBDatabase) Print() {
	iter := self.db.NewIterator(nil, nil)
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()

		fmt.Printf("%x(%d): ", key, len(key))
		node := ethutil.NewValueFromBytes(value)
		fmt.Printf("%v\n", node)
	}
}
