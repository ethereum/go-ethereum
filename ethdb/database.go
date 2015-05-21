package ethdb

import (
	"sync"

	"github.com/ethereum/go-ethereum/compression/rle"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

var OpenFileLimit = 64

type LDBDatabase struct {
	fn string

	mu sync.Mutex
	db *leveldb.DB

	queue map[string][]byte

	quit chan struct{}
}

// NewLDBDatabase returns a LevelDB wrapped object. LDBDatabase does not persist data by
// it self but requires a background poller which syncs every X. `Flush` should be called
// when data needs to be stored and written to disk.
func NewLDBDatabase(file string) (*LDBDatabase, error) {
	// Open the db
	db, err := leveldb.OpenFile(file, &opt.Options{OpenFilesCacheCapacity: OpenFileLimit})
	// check for curruption and attempt to recover
	if _, iscorrupted := err.(*errors.ErrCorrupted); iscorrupted {
		db, err = leveldb.RecoverFile(file, nil)
	}
	// (re) check for errors and abort if opening of the db failed
	if err != nil {
		return nil, err
	}
	database := &LDBDatabase{
		fn:   file,
		db:   db,
		quit: make(chan struct{}),
	}
	database.makeQueue()

	return database, nil
}

func (self *LDBDatabase) makeQueue() {
	self.queue = make(map[string][]byte)
}

// Put puts the given key / value to the queue
func (self *LDBDatabase) Put(key []byte, value []byte) {
	self.mu.Lock()
	defer self.mu.Unlock()

	self.queue[string(key)] = value
}

// Get returns the given key if it's present.
func (self *LDBDatabase) Get(key []byte) ([]byte, error) {
	self.mu.Lock()
	defer self.mu.Unlock()

	// Check queue first
	if dat, ok := self.queue[string(key)]; ok {
		return dat, nil
	}

	dat, err := self.db.Get(key, nil)
	if err != nil {
		return nil, err
	}

	return rle.Decompress(dat)
}

// Delete deletes the key from the queue and database
func (self *LDBDatabase) Delete(key []byte) error {
	self.mu.Lock()
	defer self.mu.Unlock()

	// make sure it's not in the queue
	delete(self.queue, string(key))

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

// Flush flushes out the queue to leveldb
func (self *LDBDatabase) Flush() error {
	self.mu.Lock()
	defer self.mu.Unlock()

	batch := new(leveldb.Batch)

	for key, value := range self.queue {
		batch.Put([]byte(key), rle.Compress(value))
	}
	self.makeQueue() // reset the queue

	glog.V(logger.Detail).Infoln("Flush database: ", self.fn)

	return self.db.Write(batch, nil)
}

func (self *LDBDatabase) Close() {
	if err := self.Flush(); err != nil {
		glog.V(logger.Error).Infof("error: flush '%s': %v\n", self.fn, err)
	}

	self.db.Close()
	glog.V(logger.Error).Infoln("flushed and closed db:", self.fn)
}
