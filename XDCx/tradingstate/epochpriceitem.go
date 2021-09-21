package tradingstate

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/globalsign/mgo/bson"
)

type EpochPriceItem struct {
	Epoch     uint64      `bson:"epoch" json:"epoch"`
	Orderbook common.Hash `bson:"orderbook" json:"orderbook"`
	Hash      common.Hash `bson:"hash" json:"hash"`
	Price     *big.Int    `bson:"price" json:"price"`
}

type EpochPriceItemBSON struct {
	Epoch     string `bson:"epoch" json:"epoch"`
	Orderbook string `bson:"orderbook" json:"orderbook"`
	Hash      string `bson:"hash" json:"hash"` // Keccak256Hash of Epoch and orderbook, used as an index of this collection
	Price     string `bson:"price" json:"price"`
}

func (item *EpochPriceItem) GetBSON() (interface{}, error) {
	return EpochPriceItemBSON{
		Epoch:     strconv.FormatUint(item.Epoch, 10),
		Orderbook: item.Orderbook.Hex(),
		Price:     item.Price.String(),
		Hash:      item.Hash.Hex(),
	}, nil
}

func (item *EpochPriceItem) SetBSON(raw bson.Raw) error {
	decoded := new(EpochPriceItemBSON)

	err := raw.Unmarshal(decoded)
	if err != nil {
		return fmt.Errorf("failed to decode EpochPriceItem. Err: %v", err)
	}
	epochNumber, err := strconv.ParseUint(decoded.Epoch, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse EpochPriceItem.Epoch. Err: %v", err)
	}
	item.Epoch = epochNumber
	item.Orderbook = common.HexToHash(decoded.Orderbook)
	item.Hash = common.HexToHash(decoded.Hash)
	if decoded.Price != "" {
		item.Price = ToBigInt(decoded.Price)
	}
	return nil
}

func (item *EpochPriceItem) ComputeHash() common.Hash {
	return crypto.Keccak256Hash(new(big.Int).SetUint64(item.Epoch).Bytes(), item.Orderbook.Bytes())
}
