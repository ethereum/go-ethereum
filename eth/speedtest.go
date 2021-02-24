package eth

import (
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	mrand "math/rand"
	"time"
)

func SpeedTest(db ethdb.Database) {
	var (
		count = uint64(0)
		key   = make([]byte, 32)
		t0    = time.Now()
	)
	for time.Since(t0) < 2*time.Second {
		mrand.Read(key)
		db.Get(key)
		count++
	}
	duration := uint64(time.Now().Sub(t0))
	log.Info("Speed test performed", "r/s", float64(uint64(time.Second)*count)/float64(duration))
}
