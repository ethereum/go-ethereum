package ethdb

import (
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/compression/rle"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/rcrowley/go-metrics"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

var OpenFileLimit = 64

type LDBDatabase struct {
	fn string      // filename for reporting
	db *leveldb.DB // LevelDB instance

	GetTimer       metrics.Timer // Timer for measuring the database get request counts and latencies
	PutTimer       metrics.Timer // Timer for measuring the database put request counts and latencies
	DelTimer       metrics.Timer // Timer for measuring the database delete request counts and latencies
	MissMeter      metrics.Meter // Meter for measuring the missed database get requests
	ReadMeter      metrics.Meter // Meter for measuring the database get request data usage
	WriteMeter     metrics.Meter // Meter for measuring the database put request data usage
	CompTimeMeter  metrics.Meter // Meter for measuring the total time spent in database compaction
	CompReadMeter  metrics.Meter // Meter for measuring the data read during compaction
	CompWriteMeter metrics.Meter // Meter for measuring the data written during compaction
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
		fn: file,
		db: db,
	}
	go database.meter(3 * time.Second)

	return database, nil
}

// Put puts the given key / value to the queue
func (self *LDBDatabase) Put(key []byte, value []byte) error {
	// Measure the database put latency, if requested
	if self.PutTimer != nil {
		defer self.PutTimer.UpdateSince(time.Now())
	}
	// Generate the data to write to disk, update the meter and write
	dat := rle.Compress(value)

	if self.WriteMeter != nil {
		self.WriteMeter.Mark(int64(len(dat)))
	}
	return self.db.Put(key, dat, nil)
}

// Get returns the given key if it's present.
func (self *LDBDatabase) Get(key []byte) ([]byte, error) {
	// Measure the database get latency, if requested
	if self.GetTimer != nil {
		defer self.GetTimer.UpdateSince(time.Now())
	}
	// Retrieve the key and increment the miss counter if not found
	dat, err := self.db.Get(key, nil)
	if err != nil {
		if self.MissMeter != nil {
			self.MissMeter.Mark(1)
		}
		return nil, err
	}
	// Otherwise update the actually retrieved amount of data
	if self.ReadMeter != nil {
		self.ReadMeter.Mark(int64(len(dat)))
	}
	return rle.Decompress(dat)
}

// Delete deletes the key from the queue and database
func (self *LDBDatabase) Delete(key []byte) error {
	// Measure the database delete latency, if requested
	if self.DelTimer != nil {
		defer self.DelTimer.UpdateSince(time.Now())
	}
	// Execute the actual operation
	return self.db.Delete(key, nil)
}

func (self *LDBDatabase) NewIterator() iterator.Iterator {
	return self.db.NewIterator(nil, nil)
}

// Flush flushes out the queue to leveldb
func (self *LDBDatabase) Flush() error {
	return nil
}

func (self *LDBDatabase) Close() {
	if err := self.Flush(); err != nil {
		glog.V(logger.Error).Infof("error: flush '%s': %v\n", self.fn, err)
	}
	self.db.Close()
	glog.V(logger.Error).Infoln("flushed and closed db:", self.fn)
}

func (self *LDBDatabase) LDB() *leveldb.DB {
	return self.db
}

// meter periodically retrieves internal leveldb counters and reports them to
// the metrics subsystem.
//
// This is how a stats table look like (currently):
//   Compactions
//    Level |   Tables   |    Size(MB)   |    Time(sec)  |    Read(MB)   |   Write(MB)
//   -------+------------+---------------+---------------+---------------+---------------
//      0   |          0 |       0.00000 |       1.27969 |       0.00000 |      12.31098
//      1   |         85 |     109.27913 |      28.09293 |     213.92493 |     214.26294
//      2   |        523 |    1000.37159 |       7.26059 |      66.86342 |      66.77884
//      3   |        570 |    1113.18458 |       0.00000 |       0.00000 |       0.00000
func (self *LDBDatabase) meter(refresh time.Duration) {
	// Create the counters to store current and previous values
	counters := make([][]float64, 2)
	for i := 0; i < 2; i++ {
		counters[i] = make([]float64, 3)
	}
	// Iterate ad infinitum and collect the stats
	for i := 1; ; i++ {
		// Retrieve the database stats
		stats, err := self.db.GetProperty("leveldb.stats")
		if err != nil {
			glog.V(logger.Error).Infof("failed to read database stats: %v", err)
			return
		}
		// Find the compaction table, skip the header
		lines := strings.Split(stats, "\n")
		for len(lines) > 0 && strings.TrimSpace(lines[0]) != "Compactions" {
			lines = lines[1:]
		}
		if len(lines) <= 3 {
			glog.V(logger.Error).Infof("compaction table not found")
			return
		}
		lines = lines[3:]

		// Iterate over all the table rows, and accumulate the entries
		for j := 0; j < len(counters[i%2]); j++ {
			counters[i%2][j] = 0
		}
		for _, line := range lines {
			parts := strings.Split(line, "|")
			if len(parts) != 6 {
				break
			}
			for idx, counter := range parts[3:] {
				if value, err := strconv.ParseFloat(strings.TrimSpace(counter), 64); err != nil {
					glog.V(logger.Error).Infof("compaction entry parsing failed: %v", err)
					return
				} else {
					counters[i%2][idx] += value
				}
			}
		}
		// Update all the requested meters
		if self.CompTimeMeter != nil {
			self.CompTimeMeter.Mark(int64((counters[i%2][0] - counters[(i-1)%2][0]) * 1000 * 1000 * 1000))
		}
		if self.CompReadMeter != nil {
			self.CompReadMeter.Mark(int64((counters[i%2][1] - counters[(i-1)%2][1]) * 1024 * 1024))
		}
		if self.CompWriteMeter != nil {
			self.CompWriteMeter.Mark(int64((counters[i%2][2] - counters[(i-1)%2][2]) * 1024 * 1024))
		}
		// Sleep a bit, then repeat the stats collection
		time.Sleep(refresh)
	}
}
