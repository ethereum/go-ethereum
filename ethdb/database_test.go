package ethdb

import (
	"os"

	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
)

func newDb() *LDBDatabase {
	file := filepath.Join("/", "tmp", "ldbtesttmpfile")
	if common.FileExist(file) {
		os.RemoveAll(file)
	}

	db, _ := NewLDBDatabase(file)

	return db
}
