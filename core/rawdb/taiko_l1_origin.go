package rawdb

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	// Database key prefix for L2 block's L1Origin.
	l1OriginPrefix  = []byte("TKO:L1O")
	headL1OriginKey = []byte("TKO:LastL1O")
)

// l1OriginKey calculates the L1Origin key.
// l1OriginPrefix + l2HeaderHash -> l1OriginKey
func l1OriginKey(blockID *big.Int) []byte {
	data, _ := (*math.HexOrDecimal256)(blockID).MarshalText()
	return append(l1OriginPrefix, data...)
}

//go:generate go run github.com/fjl/gencodec -type L1Origin -field-override l1OriginMarshaling -out gen_taiko_l1_origin.go

// L1Origin represents a L1Origin of a L2 block.
type L1Origin struct {
	BlockID       *big.Int       `json:"blockID" gencodec:"required"`
	L2BlockHash   common.Hash    `json:"l2BlockHash"`
	L1BlockHeight *big.Int       `json:"l1BlockHeight"`
	L1BlockHash   common.Hash    `json:"l1BlockHash"`
	BatchID       *big.Int       `json:"batchID" rlp:"optional"`
	EndOfBlock    bool           `json:"endOfBlock" rlp:"optional"`
	EndOfPreconf  bool           `json:"endOfPreconf" rlp:"optional"`
	Preconfer     common.Address `json:"preconfer" rlp:"optional"`
}

// L1OriginLegacy represents the legacy L1Origin structure of a L2 block.
type L1OriginLegacy struct {
	BlockID       *big.Int    `json:"blockID" gencodec:"required"`
	L2BlockHash   common.Hash `json:"l2BlockHash"`
	L1BlockHeight *big.Int    `json:"l1BlockHeight"`
	L1BlockHash   common.Hash `json:"l1BlockHash"`
}

type l1OriginMarshaling struct {
	BlockID       *math.HexOrDecimal256
	L1BlockHeight *math.HexOrDecimal256
}

// IsSoftBlock returns true if the L1Origin is a softblock.
func (l *L1Origin) IsSoftBlock() bool {
	return l.BatchID != nil
}

// WriteL1Origin stores a L1Origin into the database.
func WriteL1Origin(db ethdb.KeyValueWriter, blockID *big.Int, l1Origin *L1Origin) {
	data, err := rlp.EncodeToBytes(l1Origin)
	if err != nil {
		log.Crit("Failed to encode L1Origin", "err", err)
	}

	if err := db.Put(l1OriginKey(blockID), data); err != nil {
		log.Crit("Failed to store L1Origin", "err", err)
	}
}

// DeleteL1Origin removes the L1Origin.
func DeleteL1Origin(db ethdb.KeyValueWriter, blockID *big.Int) {
	_ = db.Delete(l1OriginKey(blockID))
}

// ReadL1Origin retrieves the L1Origin of the given blockID from the database.
func ReadL1Origin(db ethdb.KeyValueReader, blockID *big.Int) (*L1Origin, error) {
	data, _ := db.Get(l1OriginKey(blockID))
	if len(data) == 0 {
		return nil, nil
	}

	// First try to decode the new version (with new fields).
	l1Origin := new(L1Origin)
	if err := rlp.Decode(bytes.NewReader(data), l1Origin); err != nil {
		// If decoding the new version fails, try to decode the legacy version (without new fields).
		l1OriginLegacy := new(L1OriginLegacy)
		if err := rlp.Decode(bytes.NewReader(data), &l1OriginLegacy); err != nil {
			return nil, fmt.Errorf("invalid legacy L1Origin RLP bytes: %w", err)
		}

		// If decoding legacy version succeeds, manually
		// construct the new L1Origin with default values for the new fields.
		l1Origin = &L1Origin{
			BlockID:       l1OriginLegacy.BlockID,
			L2BlockHash:   l1OriginLegacy.L2BlockHash,
			L1BlockHeight: l1OriginLegacy.L1BlockHeight,
			L1BlockHash:   l1OriginLegacy.L1BlockHash,
			// default value, as the new fields are not present in the legacy version.
			BatchID:      common.Big0,
			EndOfBlock:   false,
			EndOfPreconf: false,
			Preconfer:    common.Address{},
		}
	}

	return l1Origin, nil
}

// WriteHeadL1Origin stores the given L1Origin as the last L1Origin.
func WriteHeadL1Origin(db ethdb.KeyValueWriter, blockID *big.Int) {
	data, _ := (*math.HexOrDecimal256)(blockID).MarshalText()
	if err := db.Put(headL1OriginKey, data); err != nil {
		log.Crit("Failed to store head L1Origin", "error", err)
	}
}

// ReadHeadL1Origin retrieves the last L1Origin from database.
func ReadHeadL1Origin(db ethdb.KeyValueReader) (*big.Int, error) {
	data, _ := db.Get(headL1OriginKey)
	if len(data) == 0 {
		return nil, nil
	}

	blockID := new(math.HexOrDecimal256)
	if err := blockID.UnmarshalText(data); err != nil {
		log.Error("Unmarshal L1Origin unmarshal error", "error", err)
		return nil, fmt.Errorf("invalid L1Origin unmarshal: %w", err)
	}

	return (*big.Int)(blockID), nil
}
