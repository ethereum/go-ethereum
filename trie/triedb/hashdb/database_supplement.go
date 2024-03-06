package hashdb

import (
	"sync"

	"github.com/VictoriaMetrics/fastcache"
)

func (db *Database) GetLock() *sync.RWMutex {
	return &db.lock
}

func (db *Database) GetCleans() *fastcache.Cache {
	return db.cleans
}
