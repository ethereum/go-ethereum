package ethdb

import (
	"os"
	"path"

	"github.com/ethereum/go-ethereum/common"
)

func newDb() *LDBDatabase {
	file := path.Join("/", "tmp", "ldbtesttmpfile")
	if common.FileExist(file) {
		os.RemoveAll(file)
	}

	db, _ := NewLDBDatabase(file)

	return db
}
