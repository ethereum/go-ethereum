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
	BlockID            *big.Int    `json:"blockID" gencodec:"required"`
	L2BlockHash        common.Hash `json:"l2BlockHash"`
	L1BlockHeight      *big.Int    `json:"l1BlockHeight" rlp:"optional"`
	L1BlockHash        common.Hash `json:"l1BlockHash" rlp:"optional"`
	IsForcedInclusion  bool        `json:"isForcedInclusion" rlp:"optional"`
	Signature          [65]byte    `json:"signature" rlp:"optional"` // signature of the envelope via p2p
	BuildPayloadArgsID [8]byte     `json:"buildPayloadArgsID" rlp:"optional"`
}

// L1OriginLegacyTwo represents a second legacy L1Origin of a L2 block.
type L1OriginLegacyTwo struct {
	BlockID            *big.Int    `json:"blockID" gencodec:"required"`
	L2BlockHash        common.Hash `json:"l2BlockHash"`
	L1BlockHeight      *big.Int    `json:"l1BlockHeight" rlp:"optional"`
	L1BlockHash        common.Hash `json:"l1BlockHash" rlp:"optional"`
	BuildPayloadArgsID [8]byte     `json:"buildPayloadArgsID" rlp:"optional"`
}

// L1OriginLegacy represents a legacy L1Origin of a L2 block.
type L1OriginLegacy struct {
	BlockID       *big.Int    `json:"blockID" gencodec:"required"`
	L2BlockHash   common.Hash `json:"l2BlockHash"`
	L1BlockHeight *big.Int    `json:"l1BlockHeight" rlp:"optional"`
	L1BlockHash   common.Hash `json:"l1BlockHash" rlp:"optional"`
}

type l1OriginMarshaling struct {
	BlockID       *math.HexOrDecimal256
	L1BlockHeight *math.HexOrDecimal256
}

// IsPreconfBlock returns true if the L1Origin is for a preconfirmation block.
// A preconfirmation block is defined as one where the L1BlockHeight is either nil or zero.
func (l *L1Origin) IsPreconfBlock() bool {
	return l.L1BlockHeight == nil || l.L1BlockHeight.Cmp(common.Big0) == 0
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

// ReadL1Origin retrieves the given L2 block's L1Origin from database.
func ReadL1Origin(db ethdb.KeyValueReader, blockID *big.Int) (*L1Origin, error) {
	data, _ := db.Get(l1OriginKey(blockID))

	if len(data) == 0 {
		return nil, nil
	}

	// try to decode standard L1Origin first
	var o L1Origin
	if err := rlp.Decode(bytes.NewReader(data), &o); err == nil {
		return &o, nil
	}

	// 3) try second legacy version, which includes BuildPayloadArgsID
	var o2 L1OriginLegacyTwo
	if err := rlp.Decode(bytes.NewReader(data), &o2); err == nil {
		return &L1Origin{
			BlockID:            o2.BlockID,
			L2BlockHash:        o2.L2BlockHash,
			L1BlockHeight:      o2.L1BlockHeight,
			L1BlockHash:        o2.L1BlockHash,
			BuildPayloadArgsID: o2.BuildPayloadArgsID,
			// new fields default to zero/false:
			IsForcedInclusion: false,
			Signature:         [65]byte{},
		}, nil
	}

	// 4) try original legacy version (no BuildPayloadArgsID, IsForcedInclusio nor Signature)
	var o1 L1OriginLegacy
	if err := rlp.Decode(bytes.NewReader(data), &o1); err == nil {
		return &L1Origin{
			BlockID:       o1.BlockID,
			L2BlockHash:   o1.L2BlockHash,
			L1BlockHeight: o1.L1BlockHeight,
			L1BlockHash:   o1.L1BlockHash,
			// new fields default to zero/false:
			IsForcedInclusion:  false,
			Signature:          [65]byte{},
			BuildPayloadArgsID: [8]byte{},
		}, nil
	}

	// 5) nothing worked
	return nil, fmt.Errorf("invalid L1Origin RLP bytes: failed new, legacyTwo and legacyOne decodes")
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
