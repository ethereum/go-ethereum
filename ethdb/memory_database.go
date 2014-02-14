package ethdb

import (
	"fmt"
	"github.com/ethereum/eth-go/ethutil"
)

/*
 * This is a test memory database. Do not use for any production it does not get persisted
 */
type MemDatabase struct {
	db map[string][]byte
}

func NewMemDatabase() (*MemDatabase, error) {
	db := &MemDatabase{db: make(map[string][]byte)}

	return db, nil
}

func (db *MemDatabase) Put(key []byte, value []byte) {
	db.db[string(key)] = value
}

func (db *MemDatabase) Get(key []byte) ([]byte, error) {
	return db.db[string(key)], nil
}

func (db *MemDatabase) Print() {
	for key, val := range db.db {
		fmt.Printf("%x(%d): ", key, len(key))
		dec, _ := ethutil.Decode(val, 0)
		node := ethutil.Conv(dec)
		fmt.Printf("%q\n", node.AsRaw())
	}
}

func (db *MemDatabase) Close() {
}

func (db *MemDatabase) LastKnownTD() []byte {
	data, _ := db.Get([]byte("LastKnownTotalDifficulty"))

	if len(data) == 0 || data == nil {
		data = []byte{0x0}
	}

	return data
}
