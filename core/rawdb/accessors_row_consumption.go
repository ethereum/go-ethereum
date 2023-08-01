package rawdb

import (
	"bytes"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rlp"
)

// WriteBlockRowConsumption writes a RowConsumption of the block to the database.
func WriteBlockRowConsumption(db ethdb.KeyValueWriter, l2BlockHash common.Hash, rc *types.RowConsumption) {
	if rc == nil {
		return
	}

	bytes, err := rlp.EncodeToBytes(&rc)
	if err != nil {
		log.Crit("Failed to RLP encode RowConsumption ", "err", err)
	}
	if err := db.Put(rowConsumptionKey(l2BlockHash), bytes); err != nil {
		log.Crit("Failed to store RowConsumption ", "err", err)
	}
}

// ReadBlockRowConsumption retrieves the RowConsumption corresponding to the block hash.
func ReadBlockRowConsumption(db ethdb.Reader, l2BlockHash common.Hash) *types.RowConsumption {
	data := ReadBlockRowConsumptionRLP(db, l2BlockHash)
	if len(data) == 0 {
		return nil
	}
	rc := new(types.RowConsumption)
	if err := rlp.Decode(bytes.NewReader(data), rc); err != nil {
		log.Crit("Invalid RowConsumption message RLP", "l2BlockHash", l2BlockHash.String(), "data", data, "err", err)
	}
	return rc
}

// ReadBlockRowConsumption retrieves the RowConsumption in its raw RLP database encoding.
func ReadBlockRowConsumptionRLP(db ethdb.Reader, l2BlockHash common.Hash) rlp.RawValue {
	data, err := db.Get(rowConsumptionKey(l2BlockHash))
	if err != nil && isNotFoundErr(err) {
		return nil
	}
	if err != nil {
		log.Crit("Failed to load RowConsumption", "l2BlockHash", l2BlockHash.String(), "err", err)
	}
	return data
}
