package pathdb

import (
	"sync"
)

func (db *Database) GetLock() *sync.RWMutex {
	return &db.lock
}
