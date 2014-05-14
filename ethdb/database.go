package ethdb

import (
	"fmt"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/syndtr/goleveldb/leveldb"
	"path"
)

type LDBDatabase struct {
	db *leveldb.DB
}

func NewLDBDatabase(name string) (*LDBDatabase, error) {
	dbPath := path.Join(ethutil.Config.ExecPath, name)

	// Open the db
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, err
	}

	database := &LDBDatabase{db: db}

	return database, nil
}

func (db *LDBDatabase) Put(key []byte, value []byte) {
	err := db.db.Put(key, value, nil)
	if err != nil {
		fmt.Println("Error put", err)
	}
}

func (db *LDBDatabase) Get(key []byte) ([]byte, error) {
	return db.db.Get(key, nil)
}

func (db *LDBDatabase) Delete(key []byte) error {
	return db.db.Delete(key, nil)
}

func (db *LDBDatabase) Db() *leveldb.DB {
	return db.db
}

func (db *LDBDatabase) LastKnownTD() []byte {
	data, _ := db.db.Get([]byte("LastKnownTotalDifficulty"), nil)

	if len(data) == 0 {
		data = []byte{0x0}
	}

	return data
}

/*
func (db *LDBDatabase) GetKeys() []*ethutil.Key {
	data, _ := db.Get([]byte("KeyRing"))

	return []*ethutil.Key{ethutil.NewKeyFromBytes(data)}
}
*/

func (db *LDBDatabase) Close() {
	// Close the leveldb database
	db.db.Close()
}

func (db *LDBDatabase) Print() {
	iter := db.db.NewIterator(nil, nil)
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()

		fmt.Printf("%x(%d): ", key, len(key))
		node := ethutil.NewValueFromBytes(value)
		fmt.Printf("%v\n", node)
	}
}
