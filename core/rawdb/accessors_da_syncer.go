package rawdb

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
)

// WriteDASyncedL1BlockNumber writes the highest synced L1 block number to the database.
func WriteDASyncedL1BlockNumber(db ethdb.KeyValueWriter, L1BlockNumber uint64) {
	value := big.NewInt(0).SetUint64(L1BlockNumber).Bytes()

	if err := db.Put(daSyncedL1BlockNumberKey, value); err != nil {
		log.Crit("Failed to update DA synced L1 block number", "err", err)
	}
}

// ReadDASyncedL1BlockNumber retrieves the highest synced L1 block number.
func ReadDASyncedL1BlockNumber(db ethdb.Reader) *uint64 {
	data, err := db.Get(daSyncedL1BlockNumberKey)
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("Failed to read DA synced L1 block number from database", "err", err)
	}
	if len(data) == 0 {
		return nil
	}

	number := new(big.Int).SetBytes(data)
	if !number.IsUint64() {
		log.Crit("Unexpected DA synced L1 block number in database", "number", number)
	}

	value := number.Uint64()
	return &value
}
