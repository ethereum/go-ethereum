package ethdb

import (
	"fmt"

	"github.com/ethereum/go-ethereum/compression/rle"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
)

type LDBDatabase struct {
	DB   *leveldb.DB
	Comp bool
}

func NewLDBDatabase(dbPath string) (*LDBDatabase, error) {
	// Open the db
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, err
	}

	database := &LDBDatabase{DB: db, Comp: false}

	return database, nil
}

func (self *LDBDatabase) Put(key []byte, value []byte) {
	if self.Comp {
		value = rle.Compress(value)
	}

	err := self.DB.Put(key, value, nil)
	if err != nil {
		fmt.Println("Error put", err)
	}
}

func (self *LDBDatabase) Get(key []byte) ([]byte, error) {
	dat, err := self.DB.Get(key, nil)
	if err != nil {
		return nil, err
	}

	if self.Comp {
		return rle.Decompress(dat)
	}

	return dat, nil
}

func (self *LDBDatabase) Delete(key []byte) error {
	return self.DB.Delete(key, nil)
}

func (self *LDBDatabase) LastKnownTD() []byte {
	data, _ := self.Get([]byte("LTD"))

	if len(data) == 0 {
		data = []byte{0x0}
	}

	return data
}

func (self *LDBDatabase) NewIterator() iterator.Iterator {
	return self.DB.NewIterator(nil, nil)
}

func (self *LDBDatabase) Write(batch *leveldb.Batch) error {
	return self.DB.Write(batch, nil)
}

func (self *LDBDatabase) Close() {
	// Close the leveldb database
	self.DB.Close()
}

func (self *LDBDatabase) Print() {
	iter := self.DB.NewIterator(nil, nil)
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()

		fmt.Printf("%x(%d): ", key, len(key))
		node := ethutil.NewValueFromBytes(value)
		fmt.Printf("%v\n", node)
	}
}
